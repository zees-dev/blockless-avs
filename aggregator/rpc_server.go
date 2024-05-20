package aggregator

import (
	"context"
	"errors"
	"net/http"
	"net/rpc"

	csavs "github.com/zees-dev/blockless-avs/contracts/bindings/BlocklessAVS"
	"github.com/zees-dev/blockless-avs/core"

	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
	sdktypes "github.com/Layr-Labs/eigensdk-go/types"
	"github.com/zees-dev/blockless-avs/aggregator/types"
)

var (
	TaskNotFoundError400                     = errors.New("400. Task not found")
	OperatorNotPartOfTaskQuorum400           = errors.New("400. Operator not part of quorum")
	TaskResponseDigestNotFoundError500       = errors.New("500. Failed to get task response digest")
	UnknownErrorWhileVerifyingSignature400   = errors.New("400. Failed to verify signature")
	SignatureVerificationFailed400           = errors.New("400. Signature verification failed")
	CallToGetCheckSignaturesIndicesFailed500 = errors.New("500. Failed to get check signatures indices")
)

func (agg *Aggregator) startServer(ctx context.Context) error {
	if err := rpc.Register(agg); err != nil {
		agg.logger.Fatal("Format of service TaskManager isn't correct. ", "err", err)
	}
	rpc.HandleHTTP()

	if err := http.ListenAndServe(agg.serverIpPortAddr, nil); err != nil {
		agg.logger.Fatal("ListenAndServe", "err", err)
	}
	return nil
}

type SignedOracleResponse struct {
	PriceResponse csavs.IBlocklessAVSPrice
	BlsSignature  bls.Signature
	OperatorId    sdktypes.OperatorId
}

// rpc endpoint which is called by operator
// reply doesn't need to be checked. If there are no errors, the task response is accepted
// rpc framework forces a reply type to exist, so we put bool as a placeholder
func (agg *Aggregator) ProcessSignedOracleResponse(signedOracleResponse *SignedOracleResponse, reply *bool) error {
	agg.logger.Infof("Received signed oracle response: %#v", signedOracleResponse)

	oracleResponseDigest, err := core.GetPriceDigest(&signedOracleResponse.PriceResponse)
	if err != nil {
		agg.logger.Error("Failed to get oracle response digest", "err", err)
		return TaskResponseDigestNotFoundError500
	}

	oracleReq, err := agg.processOracleUpdateRequest(signedOracleResponse)
	if err != nil {
		agg.logger.Error("Failed to process oracle update request", "err", err)
		return err
	}

	agg.oracleResponsesMu.Lock()
	if _, ok := agg.oracleResponses[agg.oracleRequestIndex]; !ok {
		agg.oracleResponses[agg.oracleRequestIndex] = make(map[sdktypes.TaskResponseDigest]csavs.IBlocklessAVSOracleRequest)
	}
	if _, ok := agg.oracleResponses[agg.oracleRequestIndex][oracleResponseDigest]; !ok {
		agg.oracleResponses[agg.oracleRequestIndex][oracleResponseDigest] = *oracleReq
	}
	agg.oracleResponsesMu.Unlock()

	err = agg.blsAggregationService.ProcessNewSignature(
		context.Background(), agg.oracleRequestIndex, oracleResponseDigest,
		&signedOracleResponse.BlsSignature, signedOracleResponse.OperatorId,
	)
	if err != nil {
		agg.logger.Error("Failed to process new signature", "err", err)
	}
	return err
}

func (agg *Aggregator) processOracleUpdateRequest(signedOracleResponse *SignedOracleResponse) (*csavs.IBlocklessAVSOracleRequest, error) {
	currentBlock, err := agg.clients.EthHttpClient.BlockNumber(context.Background())
	if err != nil {
		agg.logger.Error("Failed to get current block number", "err", err)
		return nil, err
	}

	// TODO: this may need to be provided from the oeprator
	quorumNumbers := types.QUORUM_NUMBERS
	quorumThresholdPercentage := types.QUORUM_THRESHOLD_NUMERATOR

	// TODO: remove this afterwards
	// explicitly convert QuorumNums to []byte
	byteSlice := make([]byte, len(quorumNumbers))
	for i, num := range quorumNumbers {
		byteSlice[i] = byte(num) // Explicit conversion from QuorumNum (uint8) to byte
	}

	agg.oracleResponsesMu.Lock()
	agg.prices[agg.oracleRequestIndex] = signedOracleResponse.PriceResponse
	agg.oracleResponsesMu.Unlock()

	// TODO: introduce `QuorumNumbers []byte and QuorumThresholdPercentage uint32` to the initial HTTP POST request
	quorumThresholdPercentages := make(sdktypes.QuorumThresholdPercentages, len(quorumNumbers))
	for i := range quorumNumbers {
		quorumThresholdPercentages[i] = sdktypes.QuorumThresholdPercentage(quorumThresholdPercentage)
	}
	// TODO(samlaf): we use seconds for now, but we should ideally pass a blocknumber to the blsAggregationService
	// and it should monitor the chain and only expire the task aggregation once the chain has reached that block number.
	taskTimeToExpiry := taskChallengeWindowBlock * blockTimeSeconds
	var quorumNums sdktypes.QuorumNums
	for _, quorumNum := range quorumNumbers {
		quorumNums = append(quorumNums, sdktypes.QuorumNum(quorumNum))
	}
	err = agg.blsAggregationService.InitializeNewTask(
		agg.oracleRequestIndex,
		uint32(currentBlock),
		quorumNums,
		quorumThresholdPercentages,
		taskTimeToExpiry,
	)
	if err != nil {
		agg.logger.Error("Failed to initialize new task", "err", err)
		return nil, err
	}

	return &csavs.IBlocklessAVSOracleRequest{
		Symbol:                    signedOracleResponse.PriceResponse.Symbol,
		ReferenceBlockNumber:      uint32(currentBlock),
		QuorumNumbers:             byteSlice,
		QuorumThresholdPercentage: uint8(quorumThresholdPercentage),
	}, nil
}

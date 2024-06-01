package aggregator

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	csavs "github.com/zees-dev/blockless-avs/contracts/bindings/BlocklessAVS"
	"github.com/zees-dev/blockless-avs/core"
	"github.com/zees-dev/blockless-avs/core/chainio"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients"
	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
	"github.com/Layr-Labs/eigensdk-go/logging"
	"github.com/Layr-Labs/eigensdk-go/services/avsregistry"
	blsagg "github.com/Layr-Labs/eigensdk-go/services/bls_aggregation"
	oprsinfoserv "github.com/Layr-Labs/eigensdk-go/services/operatorsinfo"
	sdktypes "github.com/Layr-Labs/eigensdk-go/types"
)

const (
	// number of blocks after which a task is considered expired
	// this hardcoded here because it's also hardcoded in the contracts, but should
	// ideally be fetched from the contracts
	taskChallengeWindowBlock = 100
	blockTimeSeconds         = 12 * time.Second
	avsName                  = "blocklessAVS"

	QUORUM_THRESHOLD_NUMERATOR   = sdktypes.QuorumThresholdPercentage(100)
	QUORUM_THRESHOLD_DENOMINATOR = sdktypes.QuorumThresholdPercentage(100)
	QUERY_FILTER_FROM_BLOCK      = uint64(1)
)

// we only use a single quorum (quorum 0) for blockless-avs
var QUORUM_NUMBERS = sdktypes.QuorumNums{0}

type BlockNumber = uint32
type TaskIndex = uint32
type OperatorInfo struct {
	OperatorPubkeys sdktypes.OperatorPubkeys
	OperatorAddr    common.Address
}

// Aggregator sends tasks (numbers to square) onchain, then listens for operator signed TaskResponses.
// It aggregates responses signatures, and if any of the TaskResponses reaches the QuorumThresholdPercentage for each quorum
// (currently we only use a single quorum of the ERC20Mock token), it sends the aggregated TaskResponse and signature onchain.
//
// The signature is checked in the BLSSignatureChecker.sol contract, which expects a
//
//	struct NonSignerStakesAndSignature {
//		uint32[] nonSignerQuorumBitmapIndices;
//		BN254.G1Point[] nonSignerPubkeys;
//		BN254.G1Point[] quorumApks;
//		BN254.G2Point apkG2;
//		BN254.G1Point sigma;
//		uint32[] quorumApkIndices;
//		uint32[] totalStakeIndices;
//		uint32[][] nonSignerStakeIndices; // nonSignerStakeIndices[quorumNumberIndex][nonSignerIndex]
//	}
//
// A task can only be responded onchain by having enough operators sign on it such that their stake in each quorum reaches the QuorumThresholdPercentage.
// In order to verify this onchain, the Registry contracts store the history of stakes and aggregate pubkeys (apks) for each operators and each quorum. These are
// updated everytime an operator registers or deregisters with the BLSRegistryCoordinatorWithIndices.sol contract, or calls UpdateStakes() on the StakeRegistry.sol contract,
// after having received new delegated shares or having delegated shares removed by stakers queuing withdrawals. Each of these pushes to their respective datatype array a new entry.
//
// This is true for quorumBitmaps (represent the quorums each operator is opted into), quorumApks (apks per quorum), totalStakes (total stake per quorum), and nonSignerStakes (stake per quorum per operator).
// The 4 "indices" in NonSignerStakesAndSignature basically represent the index at which to fetch their respective data, given a blockNumber at which the task was created.
// Note that different data types might have different indices, since for eg QuorumBitmaps are updated for operators registering/deregistering, but not for UpdateStakes.
// Thankfully, we have deployed a helper contract BLSOperatorStateRetriever.sol whose function getCheckSignaturesIndices() can be used to fetch the indices given a block number.
//
// The 4 other fields nonSignerPubkeys, quorumApks, apkG2, and sigma, however, must be computed individually.
// apkG2 and sigma are just the aggregated signature and pubkeys of the operators who signed the task response (aggregated over all quorums, so individual signatures might be duplicated).
// quorumApks are the G1 aggregated pubkeys of the operators who signed the task response, but one per quorum, as opposed to apkG2 which is summed over all quorums.
// nonSignerPubkeys are the G1 pubkeys of the operators who did not sign the task response, but were opted into the quorum at the blocknumber at which the task was created.
// Upon sending a task onchain (or receiving a NewTaskCreated Event if the tasks were sent by an external task generator), the aggregator can get the list of all operators opted into each quorum at that
// block number by calling the getOperatorState() function of the BLSOperatorStateRetriever.sol contract.
type Aggregator struct {
	logger           logging.Logger
	serverIpPortAddr string
	clients          *clients.Clients
	avsWriter        chainio.AvsWriterer
	avsSubscriber    chainio.AvsSubscriberer
	// aggregation related fields
	blsAggregationService blsagg.BlsAggregationService

	// oracle price related fields
	oracleRequestIndex  TaskIndex
	prices              map[TaskIndex]csavs.IBlocklessAVSPrice
	oracleResponses     map[TaskIndex]map[sdktypes.TaskResponseDigest]csavs.IBlocklessAVSOracleRequest
	oracleResponsesMu   sync.RWMutex
	oracleResponsesChan chan *csavs.ContractBlocklessAVSOracleUpdate
}

// NewAggregator creates a new Aggregator with the provided config.
func NewAggregator(c *core.Config) (*Aggregator, error) {

	avsReader, err := chainio.BuildAvsReaderFromConfig(c)
	if err != nil {
		c.Logger.Error("Cannot create avsReader", "err", err)
		return nil, err
	}

	avsWriter, err := chainio.BuildAvsWriterFromConfig(c)
	if err != nil {
		c.Logger.Errorf("Cannot create avsWriter", "err", err)
		return nil, err
	}
	avsSubscriber, err := chainio.BuildAvsSubscriberFromConfig(c)
	if err != nil {
		c.Logger.Errorf("Cannot create avsSubscriber", "err", err)
		return nil, err
	}

	chainioConfig := clients.BuildAllConfig{
		EthHttpUrl:                 c.EthHttpRpcUrl,
		EthWsUrl:                   c.EthWsRpcUrl,
		RegistryCoordinatorAddr:    c.BlocklessAVSRegistryCoordinatorAddr.String(),
		OperatorStateRetrieverAddr: c.OperatorStateRetrieverAddr.String(),
		AvsName:                    avsName,
		PromMetricsIpPortAddress:   ":9090",
	}
	clients, err := clients.BuildAll(chainioConfig, c.EcdsaPrivateKey, c.Logger)
	if err != nil {
		c.Logger.Errorf("Cannot create sdk clients", "err", err)
		return nil, err
	}

	operatorPubkeysService := oprsinfoserv.NewOperatorsInfoServiceInMemory(context.Background(), clients.AvsRegistryChainSubscriber, clients.AvsRegistryChainReader, c.Logger)
	avsRegistryService := avsregistry.NewAvsRegistryServiceChainCaller(avsReader, operatorPubkeysService, c.Logger)
	blsAggregationService := blsagg.NewBlsAggregatorService(avsRegistryService, c.Logger)

	return &Aggregator{
		logger:                c.Logger,
		serverIpPortAddr:      c.AggregatorServerIpPortAddr,
		clients:               clients,
		avsWriter:             avsWriter,
		avsSubscriber:         avsSubscriber,
		blsAggregationService: blsAggregationService,

		prices:              make(map[TaskIndex]csavs.IBlocklessAVSPrice),
		oracleResponses:     make(map[TaskIndex]map[sdktypes.TaskResponseDigest]csavs.IBlocklessAVSOracleRequest),
		oracleResponsesChan: make(chan *csavs.ContractBlocklessAVSOracleUpdate),
	}, nil
}

func (agg *Aggregator) Start(ctx context.Context) error {
	agg.logger.Infof("Starting aggregator")

	subOracleUpdates := agg.avsSubscriber.SubscribeToOracleUpdateResponses(agg.oracleResponsesChan)
	for {
		select {
		case <-ctx.Done():
			return nil
		case blsAggServiceResp := <-agg.blsAggregationService.GetResponseChannel():
			agg.logger.Info("Received response from blsAggregationService", "blsAggServiceResp", blsAggServiceResp)
			agg.sendAggregatedOracleResponseToContract(blsAggServiceResp)
		case err := <-subOracleUpdates.Err():
			agg.logger.Error("Error in websocket subscription for OracleUpdate", "err", err)
			subOracleUpdates.Unsubscribe()
			subOracleUpdates = agg.avsSubscriber.SubscribeToOracleUpdateResponses(agg.oracleResponsesChan)
		case oracleUpd := <-agg.oracleResponsesChan:
			agg.logger.Info("Received oracle update successfully!; oracleUpd: %#v", oracleUpd)
			// TODO: update metrics
			// agg.metrics.IncNumOracleUpdatesReceived()
		}
	}
}

func (agg *Aggregator) sendAggregatedOracleResponseToContract(blsAggServiceResp blsagg.BlsAggregationServiceResponse) {
	// TODO: check if blsAggServiceResp contains an err
	if blsAggServiceResp.Err != nil {
		agg.logger.Error("BlsAggregationServiceResponse contains an error", "err", blsAggServiceResp.Err)
		// panicing to help with debugging (fail fast), but we shouldn't panic if we run this in production
		panic(blsAggServiceResp.Err)
	}
	nonSignerPubkeys := []csavs.BN254G1Point{}
	for _, nonSignerPubkey := range blsAggServiceResp.NonSignersPubkeysG1 {
		nonSignerPubkeys = append(nonSignerPubkeys, core.ConvertToBN254G1Point(nonSignerPubkey))
	}
	quorumApks := []csavs.BN254G1Point{}
	for _, quorumApk := range blsAggServiceResp.QuorumApksG1 {
		quorumApks = append(quorumApks, core.ConvertToBN254G1Point(quorumApk))
	}
	nonSignerStakesAndSignature := csavs.IBLSSignatureCheckerNonSignerStakesAndSignature{
		NonSignerPubkeys:             nonSignerPubkeys,
		QuorumApks:                   quorumApks,
		ApkG2:                        core.ConvertToBN254G2Point(blsAggServiceResp.SignersApkG2),
		Sigma:                        core.ConvertToBN254G1Point(blsAggServiceResp.SignersAggSigG1.G1Point),
		NonSignerQuorumBitmapIndices: blsAggServiceResp.NonSignerQuorumBitmapIndices,
		QuorumApkIndices:             blsAggServiceResp.QuorumApkIndices,
		TotalStakeIndices:            blsAggServiceResp.TotalStakeIndices,
		NonSignerStakeIndices:        blsAggServiceResp.NonSignerStakeIndices,
	}

	agg.logger.Info("Threshold reached. Sending aggregated response onchain.",
		"taskIndex", blsAggServiceResp.TaskIndex,
	)
	agg.oracleResponsesMu.Lock()
	price := agg.prices[blsAggServiceResp.TaskIndex]
	oracleResponse := agg.oracleResponses[blsAggServiceResp.TaskIndex][blsAggServiceResp.TaskResponseDigest]
	agg.oracleResponsesMu.Unlock()
	_, err := agg.avsWriter.SendAggregatedOracleResponse(context.Background(), oracleResponse, price, nonSignerStakesAndSignature)
	if err != nil {
		agg.logger.Error("Aggregator failed to respond to task", "err", err)
	}
}

type SignedOracleResponse struct {
	PriceResponse csavs.IBlocklessAVSPrice
	BlsSignature  bls.Signature
	OperatorId    sdktypes.OperatorId
}

func (agg *Aggregator) ProcessSignedOracleResponse(signedOracleResponse *SignedOracleResponse) error {
	agg.logger.Infof("Received signed oracle response: %#v", signedOracleResponse)

	oracleResponseDigest, err := core.GetPriceDigest(&signedOracleResponse.PriceResponse)
	if err != nil {
		return errors.Wrap(err, "Failed to get oracle response digest")
	}

	oracleReq, err := agg.processOracleUpdateRequest(signedOracleResponse)
	if err != nil {
		return errors.Wrap(err, "Failed to process oracle update request")
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
		return errors.Wrap(err, "Failed to process new signature")
	}

	return nil
}

func (agg *Aggregator) processOracleUpdateRequest(signedOracleResponse *SignedOracleResponse) (*csavs.IBlocklessAVSOracleRequest, error) {
	currentBlock, err := agg.clients.EthHttpClient.BlockNumber(context.Background())
	if err != nil {
		agg.logger.Error("Failed to get current block number", "err", err)
		return nil, err
	}

	// TODO: this may need to be provided from the oeprator
	quorumNumbers := QUORUM_NUMBERS
	quorumThresholdPercentage := QUORUM_THRESHOLD_NUMERATOR

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

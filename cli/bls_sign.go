package main

import (
	"context"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/zees-dev/blockless-avs/aggregator"
	csavs "github.com/zees-dev/blockless-avs/contracts/bindings/BlocklessAVS"
	"github.com/zees-dev/blockless-avs/core"
	"github.com/zees-dev/blockless-avs/core/chainio"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients/eth"
	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
)

// SignResponse signs the oracle response
// TODO: ideally this should occur within DAPP or worker node
func SignResponse(logger *zerolog.Logger, id string, price6Decimals uint64) (*aggregator.SignedOracleResponse, error) {
	// TODO: potentially load from config
	// load BLS key from file
	blsKeyPassword, ok := os.LookupEnv("OPERATOR_BLS_KEY_PASSWORD")
	if !ok {
		logger.Warn().Msg("OPERATOR_BLS_KEY_PASSWORD env var not set. using empty string")
	}
	blsKeyPair, err := bls.ReadPrivateKeyFromFile("config-files/keys/test.bls.key.json", blsKeyPassword)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot parse bls private key")
	}

	// TODO: potentially load from
	ethRpcClient, err := eth.NewClient("http://localhost:8545")
	if err != nil {
		// logger.Errorf("Cannot create http ethclient", "err", err)
		return nil, errors.Wrap(err, "Cannot create http ethclient")
	}

	// get current block timestamp
	block, err := ethRpcClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting latest block")
	}
	blockTimestamp := block.Time()

	// construct oracle struct
	oraclePrice := &csavs.IBlocklessAVSPrice{
		Symbol:    id,
		Price:     big.NewInt(int64(price6Decimals)),
		Timestamp: uint32(blockTimestamp),
	}

	// calculate hash of abi encoded price
	priceHash, err := core.GetPriceDigest(oraclePrice)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting oracle price digest")
	}

	// sign the message
	blsSignature := blsKeyPair.SignMessage(priceHash)

	avsReader, err := chainio.BuildAvsReader(
		common.HexToAddress("0x75c68e69775fA3E9DD38eA32E554f6BF259C1135"), // registryCoordinator
		common.HexToAddress("0xCd7c00Ac6dc51e8dCc773971Ac9221cC582F3b1b"), // operatorStateRetriever
		ethRpcClient, core.NewZeroLogger(core.Development),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create AvsReader")
	}
	operatorId, err := avsReader.GetOperatorId(&bind.CallOpts{}, common.HexToAddress("0x860B6912C2d0337ef05bbC89b0C2CB6CbAEAB4A5"))
	if err != nil {
		return nil, errors.Wrap(err, "Operator ID not found")
	}

	// construct SignedOracleResponse
	// string=operatorId, [32]uint8=[0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0], string=operatorAddr, string=0x860B6912C2d0337ef05bbC89b0C2CB6CbAEAB4A5
	signedOracleResponse := &aggregator.SignedOracleResponse{
		PriceResponse: *oraclePrice,
		BlsSignature:  *blsSignature,
		OperatorId:    operatorId,
	}
	logger.Debug().Msgf("Signed oracle response", "signedOracleResponse", signedOracleResponse)

	return signedOracleResponse, nil
}

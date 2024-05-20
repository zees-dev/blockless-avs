package chainio

import (
	"context"

	csavs "github.com/zees-dev/blockless-avs/contracts/bindings/BlocklessAVS"
	"github.com/zees-dev/blockless-avs/core/config"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients/avsregistry"
	"github.com/Layr-Labs/eigensdk-go/chainio/clients/eth"
	"github.com/Layr-Labs/eigensdk-go/chainio/txmgr"
	logging "github.com/Layr-Labs/eigensdk-go/logging"
)

type AvsWriterer interface {
	avsregistry.AvsRegistryWriter

	// RaiseChallenge(
	// 	ctx context.Context,
	// 	task csavs.IIncredibleSquaringTaskManagerTask,
	// 	taskResponse csavs.IIncredibleSquaringTaskManagerTaskResponse,
	// 	taskResponseMetadata csavs.IIncredibleSquaringTaskManagerTaskResponseMetadata,
	// 	pubkeysOfNonSigningOperators []csavs.BN254G1Point,
	// ) (*types.Receipt, error)

	SendAggregatedOracleResponse(ctx context.Context,
		oracleResponse csavs.IBlocklessAVSOracleRequest,
		price csavs.IBlocklessAVSPrice,
		nonSignerStakesAndSignature csavs.IBLSSignatureCheckerNonSignerStakesAndSignature,
	) (*types.Receipt, error)
}

type AvsWriter struct {
	avsregistry.AvsRegistryWriter
	AvsContractBindings *AvsManagersBindings
	logger              logging.Logger
	TxMgr               txmgr.TxManager
	client              eth.Client
}

var _ AvsWriterer = (*AvsWriter)(nil)

func BuildAvsWriterFromConfig(c *config.Config) (*AvsWriter, error) {
	return BuildAvsWriter(c.TxMgr, c.BlocklessAVSRegistryCoordinatorAddr, c.OperatorStateRetrieverAddr, *c.EthHttpClient, c.Logger)
}

func BuildAvsWriter(txMgr txmgr.TxManager, registryCoordinatorAddr, operatorStateRetrieverAddr gethcommon.Address, ethHttpClient eth.Client, logger logging.Logger) (*AvsWriter, error) {
	avsServiceBindings, err := NewAvsManagersBindings(registryCoordinatorAddr, operatorStateRetrieverAddr, ethHttpClient, logger)
	if err != nil {
		logger.Error("Failed to create contract bindings", "err", err)
		return nil, err
	}
	avsRegistryWriter, err := avsregistry.BuildAvsRegistryChainWriter(registryCoordinatorAddr, operatorStateRetrieverAddr, logger, ethHttpClient, txMgr)
	if err != nil {
		return nil, err
	}
	return NewAvsWriter(avsRegistryWriter, avsServiceBindings, logger, txMgr), nil
}
func NewAvsWriter(avsRegistryWriter avsregistry.AvsRegistryWriter, avsServiceBindings *AvsManagersBindings, logger logging.Logger, txMgr txmgr.TxManager) *AvsWriter {
	return &AvsWriter{
		AvsRegistryWriter:   avsRegistryWriter,
		AvsContractBindings: avsServiceBindings,
		logger:              logger,
		TxMgr:               txMgr,
	}
}

func (w *AvsWriter) SendAggregatedOracleResponse(
	ctx context.Context,
	oracleResponse csavs.IBlocklessAVSOracleRequest,
	price csavs.IBlocklessAVSPrice,
	nonSignerStakesAndSignature csavs.IBLSSignatureCheckerNonSignerStakesAndSignature,
) (*types.Receipt, error) {
	txOpts, err := w.TxMgr.GetNoSendTxOpts()
	if err != nil {
		w.logger.Errorf("Error getting tx opts")
		return nil, err
	}
	tx, err := w.AvsContractBindings.ServiceManager.ContractBlocklessAVSTransactor.UpdateOraclePrice(txOpts, oracleResponse, price, nonSignerStakesAndSignature)
	if err != nil {
		w.logger.Error("Error submitting SendAggregatedOracleResponse tx while calling updateOraclePrice", "err", err)
		return nil, err
	}
	receipt, err := w.TxMgr.Send(ctx, tx)
	if err != nil {
		w.logger.Errorf("Error submitting UpdateOraclePrice tx")
		return nil, err
	}
	return receipt, nil
}

// func (w *AvsWriter) RaiseChallenge(
// 	ctx context.Context,
// 	task cstaskmanager.IIncredibleSquaringTaskManagerTask,
// 	taskResponse cstaskmanager.IIncredibleSquaringTaskManagerTaskResponse,
// 	taskResponseMetadata cstaskmanager.IIncredibleSquaringTaskManagerTaskResponseMetadata,
// 	pubkeysOfNonSigningOperators []cstaskmanager.BN254G1Point,
// ) (*types.Receipt, error) {
// 	txOpts, err := w.TxMgr.GetNoSendTxOpts()
// 	if err != nil {
// 		w.logger.Errorf("Error getting tx opts")
// 		return nil, err
// 	}
// 	tx, err := w.AvsContractBindings.TaskManager.RaiseAndResolveChallenge(txOpts, task, taskResponse, taskResponseMetadata, pubkeysOfNonSigningOperators)
// 	if err != nil {
// 		w.logger.Errorf("Error assembling RaiseChallenge tx")
// 		return nil, err
// 	}
// 	receipt, err := w.TxMgr.Send(ctx, tx)
// 	if err != nil {
// 		w.logger.Errorf("Error submitting CreateNewTask tx")
// 		return nil, err
// 	}
// 	return receipt, nil
// }

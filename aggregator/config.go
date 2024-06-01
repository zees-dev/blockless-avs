package aggregator

import (
	"context"
	"errors"
	"os"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients/eth"
	"github.com/Layr-Labs/eigensdk-go/chainio/clients/wallet"
	"github.com/Layr-Labs/eigensdk-go/chainio/txmgr"
	"github.com/Layr-Labs/eigensdk-go/logging"
	"github.com/Layr-Labs/eigensdk-go/signerv2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
	"github.com/zees-dev/blockless-avs/core"

	sdklogging "github.com/Layr-Labs/eigensdk-go/logging"
	sdkutils "github.com/Layr-Labs/eigensdk-go/utils"
)

// These are read from OperatorConfigFileFlag
type AggregatorConfigRaw struct {
	Environment                sdklogging.LogLevel `yaml:"environment"`
	EthRpcUrl                  string              `yaml:"eth_rpc_url"`
	EthWsUrl                   string              `yaml:"eth_ws_url"`
	AggregatorServerIpPortAddr string              `yaml:"aggregator_server_ip_port_address"`
	RegisterOperatorOnStartup  bool                `yaml:"register_operator_on_startup"`
}

// These are read from BlocklessAVSDeploymentFileFlag
type BlocklessAVSDeploymentRaw struct {
	Addresses BlocklessAVSContractsRaw `json:"addresses"`
}
type BlocklessAVSContractsRaw struct {
	RegistryCoordinatorAddr    string `json:"registryCoordinator"`
	OperatorStateRetrieverAddr string `json:"operatorStateRetriever"`
}

// NewConfig parses config file to read from from flags or environment variables
// Note: This config is shared by challenger and aggregator and so we put in the core.
// Operator has a different config and is meant to be used by the operator CLI.
func NewAggregatorConfig(ctx *cli.Context, logger logging.Logger) (*core.Config, error) {
	configFilePath := ctx.String(AggregatorConfigFileFlag.Name)
	blocklessAVSDeploymentFilePath := ctx.String(BlocklessAVSDeploymentFileFlag.Name)
	ecdsaPrivateKeyString := ctx.String(EcdsaPrivateKeyFlag.Name)

	var aggConfigRaw AggregatorConfigRaw
	if configFilePath != "" {
		sdkutils.ReadYamlConfig(configFilePath, &aggConfigRaw)
	}

	var blocklessAVSDeploymentRaw BlocklessAVSDeploymentRaw
	if _, err := os.Stat(blocklessAVSDeploymentFilePath); errors.Is(err, os.ErrNotExist) {
		panic("Path " + blocklessAVSDeploymentFilePath + " does not exist")
	}
	sdkutils.ReadJsonConfig(blocklessAVSDeploymentFilePath, &blocklessAVSDeploymentRaw)

	// panic if required address fields are missing
	operatorStateRetrieverAddr := common.HexToAddress(blocklessAVSDeploymentRaw.Addresses.OperatorStateRetrieverAddr)
	if operatorStateRetrieverAddr == common.HexToAddress("") {
		panic("Config: BLSOperatorStateRetrieverAddr is required")
	}
	blocklessAVSRegistryCoordinatorAddr := common.HexToAddress(blocklessAVSDeploymentRaw.Addresses.RegistryCoordinatorAddr)
	if blocklessAVSRegistryCoordinatorAddr == common.HexToAddress("") {
		panic("Config: BLSOperatorStateRetrieverAddr is required")
	}

	ethRpcClient, err := eth.NewClient(aggConfigRaw.EthRpcUrl)
	if err != nil {
		logger.Error("Cannot create http ethclient", "err", err)
		return nil, err
	}

	ethWsClient, err := eth.NewClient(aggConfigRaw.EthWsUrl)
	if err != nil {
		logger.Error("Cannot create ws ethclient", "err", err)
		return nil, err
	}

	if ecdsaPrivateKeyString[:2] == "0x" {
		ecdsaPrivateKeyString = ecdsaPrivateKeyString[2:]
	}
	ecdsaPrivateKey, err := crypto.HexToECDSA(ecdsaPrivateKeyString)
	if err != nil {
		logger.Errorf("Cannot parse ecdsa private key", "err", err)
		return nil, err
	}

	aggregatorAddr, err := sdkutils.EcdsaPrivateKeyToAddress(ecdsaPrivateKey)
	if err != nil {
		logger.Error("Cannot get operator address", "err", err)
		return nil, err
	}

	chainId, err := ethRpcClient.ChainID(context.Background())
	if err != nil {
		logger.Error("Cannot get chainId", "err", err)
		return nil, err
	}

	signerV2, _, err := signerv2.SignerFromConfig(signerv2.Config{PrivateKey: ecdsaPrivateKey}, chainId)
	if err != nil {
		panic(err)
	}
	skWallet, err := wallet.NewPrivateKeyWallet(ethRpcClient, signerV2, aggregatorAddr, logger)
	if err != nil {
		panic(err)
	}
	txMgr := txmgr.NewSimpleTxManager(skWallet, ethRpcClient, logger, aggregatorAddr)

	config := &core.Config{
		EcdsaPrivateKey:                     ecdsaPrivateKey,
		Logger:                              logger,
		EthWsRpcUrl:                         aggConfigRaw.EthWsUrl,
		EthHttpRpcUrl:                       aggConfigRaw.EthRpcUrl,
		EthHttpClient:                       &ethRpcClient,
		EthWsClient:                         &ethWsClient,
		OperatorStateRetrieverAddr:          operatorStateRetrieverAddr,
		BlocklessAVSRegistryCoordinatorAddr: blocklessAVSRegistryCoordinatorAddr,
		AggregatorServerIpPortAddr:          aggConfigRaw.AggregatorServerIpPortAddr,
		RegisterOperatorOnStartup:           aggConfigRaw.RegisterOperatorOnStartup,
		SignerFn:                            signerV2,
		TxMgr:                               txMgr,
		AggregatorAddress:                   aggregatorAddr,
	}
	return config, nil
}

var (
	AggregatorConfigFileFlag = &cli.StringFlag{
		Name:       "config",
		Usage:      "Load configuration from `FILE`",
		Value:      "config-files/aggregator.yaml",
		Required:   true,
		HasBeenSet: true,
	}
	BlocklessAVSDeploymentFileFlag = &cli.StringFlag{
		Name:     "blockless-avs-deployment",
		Required: true,
		Usage:    "Load blockless avs contract addresses from `FILE`",
	}
	EcdsaPrivateKeyFlag = &cli.StringFlag{
		Name:     "ecdsa-private-key",
		Usage:    "Ethereum private key",
		Required: true,
		EnvVars:  []string{"ECDSA_PRIVATE_KEY"},
	}
)

var AggregatorFlags = []cli.Flag{
	AggregatorConfigFileFlag,
	BlocklessAVSDeploymentFileFlag,
	EcdsaPrivateKeyFlag,
}

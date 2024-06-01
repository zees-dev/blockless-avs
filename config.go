package config

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/Layr-Labs/eigensdk-go/logging"
	b7sConfig "github.com/blocklessnetwork/b7s/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"github.com/zees-dev/blockless-avs/aggregator"
)

const CoreConfigKey = "coreConfig"
const AggregatorConfigKey = "aggregatorConfig"
const OperatorConfigKey = "operatorConfig"

type CoreConfig struct {
	AppName string
	DevMode bool
	Logger  logging.Logger

	BlocklessConfig *b7sConfig.Config
}

func GetCoreConfig(ctx *cli.Context) *CoreConfig {
	if ctx.App.Metadata[CoreConfigKey] == nil {
		panic("Core config not initialized.")
	}
	return ctx.App.Metadata[CoreConfigKey].(*CoreConfig)
}

type AggregatorConfig struct {
	*CoreConfig
	Aggregator *aggregator.Aggregator
}

func GetAggregatorConfig(ctx *cli.Context) *AggregatorConfig {
	if ctx.App.Metadata[AggregatorConfigKey] == nil {
		panic("Aggregator config not initialized.")
	}
	return ctx.App.Metadata[AggregatorConfigKey].(*AggregatorConfig)
}

type AVSOperator interface {
	Start(ctx context.Context) error
	RegisterOperatorWithEigenlayer() error
	DepositIntoStrategy(strategyAddr common.Address, amount *big.Int) error
	RegisterOperatorWithAvs(operatorEcdsaKeyPair *ecdsa.PrivateKey) error
	PrintOperatorStatus() error
}

type OperatorConfig struct {
	*CoreConfig
	NodeConfig *NodeConfig
	Operator   AVSOperator
	Headless   bool
}

func GetOperatorConfig(ctx *cli.Context) *OperatorConfig {
	if ctx.App.Metadata[OperatorConfigKey] == nil {
		panic("Operator config config not initialized.")
	}
	return ctx.App.Metadata[OperatorConfigKey].(*OperatorConfig)
}

type NodeConfig struct {
	// used to set the logger level (true = info, false = debug)
	Production                    bool   `yaml:"production"`
	OperatorAddress               string `yaml:"operator_address"`
	OperatorStateRetrieverAddress string `yaml:"operator_state_retriever_address"`
	AVSRegistryCoordinatorAddress string `yaml:"avs_registry_coordinator_address"`
	TokenStrategyAddr             string `yaml:"token_strategy_addr"`
	AVSServiceManagerAddress      string `yaml:"avs_service_manager_addr"`
	EthRpcUrl                     string `yaml:"eth_rpc_url"`
	EthWsUrl                      string `yaml:"eth_ws_url"`
	BlsPrivateKeyStorePath        string `yaml:"bls_private_key_store_path"`
	EcdsaPrivateKeyStorePath      string `yaml:"ecdsa_private_key_store_path"`
	AggregatorServerIpPortAddress string `yaml:"aggregator_server_ip_port_address"`
	RegisterOperatorOnStartup     bool   `yaml:"register_operator_on_startup"`
	EigenMetricsIpPortAddress     string `yaml:"eigen_metrics_ip_port_address"`
	EnableMetrics                 bool   `yaml:"enable_metrics"`
	NodeApiIpPortAddress          string `yaml:"node_api_ip_port_address"`
	EnableNodeApi                 bool   `yaml:"enable_node_api"`
}

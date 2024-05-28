package config

import (
	"github.com/Layr-Labs/eigensdk-go/logging"
	b7sConfig "github.com/blocklessnetwork/b7s/config"
	"github.com/urfave/cli/v2"
	"github.com/zees-dev/blockless-avs/aggregator"
	"github.com/zees-dev/blockless-avs/operator"
	"github.com/zees-dev/blockless-avs/types"
)

const CoreConfigKey = "coreConfig"
const AggregatorConfigKey = "aggregatorConfig"
const OperatorConfigKey = "operatorConfig"

type CoreConfig struct {
	AppName  string
	Headless bool
	DevMode  bool
	Logger   logging.Logger

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

type OperatorConfig struct {
	*CoreConfig
	NodeConfig *types.NodeConfig
	Operator   *operator.Operator
}

func GetOperatorConfig(ctx *cli.Context) *OperatorConfig {
	if ctx.App.Metadata[OperatorConfigKey] == nil {
		panic("Operator config config not initialized.")
	}
	return ctx.App.Metadata[OperatorConfigKey].(*OperatorConfig)
}

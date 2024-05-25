package config

import (
	"github.com/Layr-Labs/eigensdk-go/logging"
	b7sConfig "github.com/blocklessnetwork/b7s/config"
	"github.com/urfave/cli/v2"
	"github.com/zees-dev/blockless-avs/operator"
	"github.com/zees-dev/blockless-avs/types"
)

const AppConfigKey = "appConfig"
const LoggerKey = "logger"

type AppConfig struct {
	// AVSFlags config.Config

	AppName         string
	Headless        bool
	DevMode         bool
	Logger          logging.Logger
	NodeConfig      *types.NodeConfig
	Operator        *operator.Operator
	BlocklessConfig *b7sConfig.Config
}

func GetAppConfig(ctx *cli.Context) *AppConfig {
	if ctx.App.Metadata[AppConfigKey] == nil {
		panic("App config not initialized.")
	}
	return ctx.App.Metadata[AppConfigKey].(*AppConfig)
}

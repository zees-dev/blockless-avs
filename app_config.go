package config

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"github.com/zees-dev/blockless-avs/operator"
	"github.com/zees-dev/blockless-avs/types"
)

const AppConfigKey = "appConfig"

type AppConfig struct {
	// AVSFlags config.Config

	AppName    string
	Headless   bool
	DevMode    bool
	Logger     *zerolog.Logger
	NodeConfig *types.NodeConfig
	Operator   *operator.Operator

	lock sync.RWMutex
}

func GetAppConfig(ctx *cli.Context) *AppConfig {
	if ctx.App.Metadata[AppConfigKey] == nil {
		panic("App config not initialized.")
	}
	return ctx.App.Metadata[AppConfigKey].(*AppConfig)
}

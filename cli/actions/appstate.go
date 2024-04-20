package actions

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	avs "github.com/zees-dev/blockless-avs/node/pkg"
)

const AppStateKey = "appState"

type AppState struct {
	Logger zerolog.Logger
	AVSCfg *avs.Cfg
	lock   sync.RWMutex
}

func GetAppState(ctx *cli.Context) *AppState {
	if ctx.App.Metadata[AppStateKey] == nil {
		panic("App state not initialized.")
	}
	return ctx.App.Metadata[AppStateKey].(*AppState)
}

package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/zees-dev/blockless-avs/cli/actions"
	"github.com/zees-dev/blockless-avs/core/config"
)

func main() {
	app := cli.NewApp()

	app.Name = "Blockless AVS"
	app.Usage = "TODO"

	// init app state, store in context
	app.Before = func(c *cli.Context) error {
		logger := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
		c.App.Metadata[actions.AppStateKey] = &actions.AppState{
			Logger: logger,
		}
		return nil
	}

	app.Commands = []*cli.Command{
		{
			Name:   "run-avs",
			Usage:  "Starts the server",
			Action: actions.RunAVS,
			// Flags: []cli.Flag{config.ConfigFileFlag},
		},
		{
			Name:    "register-operator-with-eigenlayer",
			Aliases: []string{"rel"},
			Usage:   "registers operator with eigenlayer (this should be called via eigenlayer cli, not plugin, but keeping here for convenience for now)",
			Action:  actions.RegisterOperatorWithEigenlayer,
			Flags:   []cli.Flag{config.ConfigFileFlag},
		},
		{
			Name:    "deposit-into-strategy",
			Aliases: []string{"dis"},
			Usage:   "deposit tokens into a strategy",
			Action:  actions.DepositIntoStrategy,
			Flags: []cli.Flag{
				config.ConfigFileFlag,
				// &cli.StringFlag{
				// 	Name:     "strategy-addr",
				// 	Usage:    "Address of Strategy contract to deposit into",
				// 	Required: true,
				// },
				&cli.StringFlag{
					Name:     "amount",
					Usage:    "amount of tokens to deposit into strategy",
					Required: true,
				},
			},
		},
		{
			Name:    "register-operator-with-avs",
			Aliases: []string{"rowa"},
			Usage:   "registers bls keys with pubkey-compendium, opts into slashing by avs service-manager, and registers operators with avs registry",
			Action:  actions.RegisterOperatorWithAvs,
			Flags:   []cli.Flag{config.ConfigFileFlag},
		},
		{
			Name:    "deregister-operator-with-avs",
			Aliases: []string{"dowa"},
			Action: func(ctx *cli.Context) error {
				log.Fatal().Msg("Command not implemented.")
				return nil
			},
			Flags: []cli.Flag{config.ConfigFileFlag},
		},
		{
			Name:    "print-operator-status",
			Aliases: []string{"pos"},
			Usage:   "prints operator status as viewed from incredible squaring contracts",
			Action:  actions.PrintOperatorStatus,
			Flags:   []cli.Flag{config.ConfigFileFlag},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run app")
	}
}

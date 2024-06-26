package main

import (
	"errors"
	"math/big"
	"os"

	sdkecdsa "github.com/Layr-Labs/eigensdk-go/crypto/ecdsa"
	sdkutils "github.com/Layr-Labs/eigensdk-go/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	avs "github.com/zees-dev/blockless-avs"
	"github.com/zees-dev/blockless-avs/aggregator"
	"github.com/zees-dev/blockless-avs/core"
	"github.com/zees-dev/blockless-avs/operator"
)

const AppName = "Blockless AVS"

var (
	DevModeFlag = &cli.BoolFlag{
		Name:       "devmode",
		Required:   true,
		Usage:      "Run in development mode",
		Value:      true,
		HasBeenSet: true,
	}
	/* Operator Flags */
	OperatorConfigFileFlag = &cli.StringFlag{
		Name:       "config",
		Usage:      "Load configuration from `FILE`",
		Value:      "config-files/operator.anvil.yaml",
		Required:   true,
		HasBeenSet: true,
	}
	HeadlessFlag = &cli.BoolFlag{
		Name:       "headless",
		Required:   true,
		Usage:      "Run blockless node in headless mode",
		Value:      true,
		HasBeenSet: true,
	}
)

func main() {
	app := cli.NewApp()

	app.Name = AppName
	app.Usage = "TODO"

	// globally required flags
	app.Flags = []cli.Flag{
		DevModeFlag,
	}

	// init app state, store in context
	app.Before = func(ctx *cli.Context) error {
		logger := core.NewZeroLogger(core.Development)

		devMode := ctx.Bool(DevModeFlag.Name)

		ctx.App.Metadata[avs.CoreConfigKey] = &avs.CoreConfig{
			AppName: AppName,
			Logger:  logger,
			// NodeConfig:      &nodeConfig,
			// Operator:        operator,
			DevMode: devMode,
		}
		return nil
	}

	app.Commands = []*cli.Command{
		{
			Name:   "run-avs-aggregator",
			Usage:  "start blockless AVS aggregator (head)",
			Action: RunAVS,
			Before: func(ctx *cli.Context) error {
				coreConfig := ctx.App.Metadata[avs.CoreConfigKey].(*avs.CoreConfig)

				// parse operator specific flags
				config, err := aggregator.NewAggregatorConfig(ctx, coreConfig.Logger)
				if err != nil {
					return err
				}
				// configJson, err := json.MarshalIndent(config, "", "  ")
				// if err != nil {
				// 	config.Logger.Fatalf(err.Error())
				// }
				// fmt.Println("Config:", string(configJson))
				agg, err := aggregator.NewAggregator(config)
				if err != nil {
					return err
				}

				// parse blockless node specific flags
				b7sConfig := ParseBlocklesssFlags(ctx)
				coreConfig.BlocklessConfig = &b7sConfig

				ctx.App.Metadata[avs.AggregatorConfigKey] = &avs.AggregatorConfig{
					CoreConfig: coreConfig,
					Aggregator: agg,
				}
				return nil
			},
			Flags: append(aggregator.AggregatorFlags, BlocklessFlags...),
		},
		{
			Name:   "run-avs-operator",
			Usage:  "start blockless operator AVS node (worker)",
			Action: RunAVS,
			Before: func(ctx *cli.Context) error {
				coreConfig := ctx.App.Metadata[avs.CoreConfigKey].(*avs.CoreConfig)

				headless := ctx.Bool(HeadlessFlag.Name)

				// parse operator specific flags
				configPath := ctx.String(OperatorConfigFileFlag.Name)
				nodeConfig := avs.NodeConfig{}
				if err := sdkutils.ReadYamlConfig(configPath, &nodeConfig); err != nil {
					return err
				}
				operator, err := operator.NewOperatorFromConfig(coreConfig.Logger, nodeConfig)
				if err != nil {
					return err
				}

				// parse blockless node specific flags
				b7sConfig := ParseBlocklesssFlags(ctx)
				coreConfig.BlocklessConfig = &b7sConfig

				ctx.App.Metadata[avs.OperatorConfigKey] = &avs.OperatorConfig{
					CoreConfig: coreConfig,
					Operator:   operator,
					NodeConfig: &nodeConfig,
					Headless:   headless,
				}
				return nil
			},
			Flags: append(BlocklessFlags, OperatorConfigFileFlag, HeadlessFlag),
		},
		{
			Name:    "register-operator-with-eigenlayer",
			Aliases: []string{"rel"},
			Usage:   "registers operator with eigenlayer (this should be called via eigenlayer cli, not plugin, but keeping here for convenience for now)",
			Action: func(ctx *cli.Context) error {
				logger := ctx.App.Metadata[avs.CoreConfigKey].(*avs.CoreConfig).Logger

				configPath := ctx.String(OperatorConfigFileFlag.Name)

				nodeConfig := avs.NodeConfig{}
				if err := sdkutils.ReadYamlConfig(configPath, &nodeConfig); err != nil {
					return err
				}
				operator, err := operator.NewOperatorFromConfig(logger, nodeConfig)
				if err != nil {
					return err
				}
				return operator.RegisterOperatorWithEigenlayer()
			},
			Flags: []cli.Flag{OperatorConfigFileFlag},
		},
		{
			Name:    "deposit-into-strategy",
			Aliases: []string{"dis"},
			Usage:   "deposit tokens into a strategy",
			Action: func(ctx *cli.Context) error {
				logger := ctx.App.Metadata[avs.CoreConfigKey].(*avs.CoreConfig).Logger

				configPath := ctx.String(OperatorConfigFileFlag.Name)
				nodeConfig := avs.NodeConfig{}
				if err := sdkutils.ReadYamlConfig(configPath, &nodeConfig); err != nil {
					return err
				}
				operator, err := operator.NewOperatorFromConfig(logger, nodeConfig)
				if err != nil {
					return err
				}

				strategyAddrStr := nodeConfig.TokenStrategyAddr
				strategyAddr := common.HexToAddress(strategyAddrStr)
				amountStr := ctx.String("amount")
				amount, ok := new(big.Int).SetString(amountStr, 10)
				if !ok {
					logger.Error("Error converting amount to big.Int")
					return errors.New("Error converting amount to big.Int")
				}
				return operator.DepositIntoStrategy(strategyAddr, amount)
			},
			Flags: []cli.Flag{
				OperatorConfigFileFlag,
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
			Action: func(ctx *cli.Context) error {
				logger := ctx.App.Metadata[avs.CoreConfigKey].(*avs.CoreConfig).Logger

				configPath := ctx.String(OperatorConfigFileFlag.Name)

				nodeConfig := avs.NodeConfig{}
				if err := sdkutils.ReadYamlConfig(configPath, &nodeConfig); err != nil {
					return err
				}
				operator, err := operator.NewOperatorFromConfig(logger, nodeConfig)
				if err != nil {
					return err
				}

				ecdsaKeyPassword, ok := os.LookupEnv("OPERATOR_ECDSA_KEY_PASSWORD")
				if !ok {
					logger.Info("OPERATOR_ECDSA_KEY_PASSWORD env var not set. using empty string")
				}
				operatorEcdsaPrivKey, err := sdkecdsa.ReadKey(
					nodeConfig.EcdsaPrivateKeyStorePath,
					ecdsaKeyPassword,
				)
				if err != nil {
					return err
				}
				return operator.RegisterOperatorWithAvs(operatorEcdsaPrivKey)
			},
			Flags: []cli.Flag{OperatorConfigFileFlag},
		},
		{
			Name:    "deregister-operator-with-avs",
			Aliases: []string{"dowa"},
			Action: func(ctx *cli.Context) error {
				panic("not implemented")
			},
			Flags: []cli.Flag{OperatorConfigFileFlag},
		},
		{
			Name:    "print-operator-status",
			Aliases: []string{"pos"},
			Usage:   "prints operator status as viewed from blockless-avs contracts",
			Action: func(ctx *cli.Context) error {
				logger := ctx.App.Metadata[avs.CoreConfigKey].(*avs.CoreConfig).Logger

				configPath := ctx.String(OperatorConfigFileFlag.Name)
				nodeConfig := avs.NodeConfig{}
				if err := sdkutils.ReadYamlConfig(configPath, &nodeConfig); err != nil {
					return err
				}
				operator, err := operator.NewOperatorFromConfig(logger, nodeConfig)
				if err != nil {
					return err
				}
				return operator.PrintOperatorStatus()
			},
			Flags: []cli.Flag{OperatorConfigFileFlag},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run app")
	}
}

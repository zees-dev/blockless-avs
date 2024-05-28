package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/zees-dev/blockless-avs/aggregator"
	"github.com/zees-dev/blockless-avs/core/config"
)

var (
	// Version is the version of the binary.
	Version   string
	GitCommit string
	GitDate   string
)

func main() {
	app := cli.NewApp()
	app.Flags = config.AggregatorFlags
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "blockless-avs-aggregator"
	app.Usage = "Blockless AVS Aggregator"
	app.Description = "Service that aggregates signatures and submits result on-chain."

	app.Action = aggregatorMain
	if err := app.Run(os.Args); err != nil {
		log.Fatalln("Application failed.", "Message:", err)
	}
}

func aggregatorMain(ctx *cli.Context) error {
	log.Println("Initializing Aggregator")
	config, err := config.NewConfig(ctx)
	if err != nil {
		return err
	}
	configJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		config.Logger.Fatalf(err.Error())
	}
	fmt.Println("Config:", string(configJson))

	agg, err := aggregator.NewAggregator(config)
	if err != nil {
		return err
	}

	if err = agg.Start(context.Background()); err != nil {
		return err
	}

	return nil
}

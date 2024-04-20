package actions

import (
	"github.com/urfave/cli/v2"

	"context"
	"fmt"
	"net"
	"os"
	"os/signal"

	avs "github.com/zees-dev/blockless-avs/node/pkg"
)

func RunAVS(c *cli.Context) error {
	app := GetAppState(c)
	cfg := avs.ParseFlags()
	app.AVSCfg = cfg
	logger := app.Logger

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	failed := make(chan struct{})

	logger.Info().Bool("headless mode", !app.AVSCfg.Headless).Msg("the server is starting")

	// assets, err := fs.Sub(embeddedFiles, "assets")
	// if err != nil {
	// 	logger.Info().Err(err).Msg("Failed to locate embedded assets")
	// 	return
	// }

	// Get the port the server is listening on.
	// Listen on a random port.
	listenHost := "localhost"
	if app.AVSCfg.Headless {
		listenHost = "0.0.0.0"
	}

	// listenAddr := fmt.Sprintf("%s:%s", listenHost, cfg.API)
	listenAddr := fmt.Sprintf("%s:%s", listenHost, "8080")
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to listen on a port: %v")
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://localhost:%s", fmt.Sprint(port))
	logger.Info().Str("serverUrl", serverURL).Msgf("Production server listening on %s", serverURL)

	// Register API routes.
	avs.RegisterAPIRoutes(*app.AVSCfg)

	// Create the main context for p2p
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	select {
	case <-sig:
		logger.Info().Msg("Blockless AVS stopping")
	case <-done:
		logger.Info().Msg("Blockless AVS done")
	case <-failed:
		logger.Info().Msg("Blockless AVS aborted")
	}

	// If we receive a second interrupt signal, exit immediately.
	go func() {
		<-sig
		logger.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	return nil
}

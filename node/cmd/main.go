//go:build !dev && !app
// +build !dev,!app

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	cli "github.com/urfave/cli/v2"
	avs "github.com/zees-dev/blockless-avs/pkg"
)

// //go:embed assets/*
// var embeddedFiles embed.FS
var logger zerolog.Logger
var cfg *avs.Cfg

func init() {
	cfg = avs.ParseFlags()
	// cfg.appname = "My dApp"

	logger = zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	if !cfg.Headless {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

func main() {
	app := &cli.App{
		Name:  "Blockless AVS",
		Usage: "TODO",
		Before: func(c *cli.Context) error {
			logger = zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
			// TODO: initialize config
			// 	return initializeConfig(c)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:   "serve",
				Usage:  "Starts the server",
				Action: runServer,
				// Flags:  serverFlags(),
			},
			// {
			// 	Name:    "register-operator-with-eigenlayer",
			// 	Usage:   "registers operator with eigenlayer (this should be called via eigenlayer cli, not plugin, but keeping here for convenience for now)",
			// 	Action:  actions.RegisterOperatorWithEigenlayer,
			// 	Aliases: []string{"rel"},
			// },
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "$HOME/.appName.yaml",
				Usage: "config file to use",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run app")
	}
}

func runServer(c *cli.Context) error {
	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	failed := make(chan struct{})

	// logger.Info().Bool("headless mode", !cfg.headless).Msg("the server is starting")

	// assets, err := fs.Sub(embeddedFiles, "assets")
	// if err != nil {
	// 	logger.Info().Err(err).Msg("Failed to locate embedded assets")
	// 	return
	// }

	// Get the port the server is listening on.
	// Listen on a random port.
	listenHost := "localhost"
	if cfg.Headless {
		listenHost = "0.0.0.0"
	}

	listenAddr := fmt.Sprintf("%s:%s", listenHost, cfg.API)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to listen on a port: %v")
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://localhost:%s", fmt.Sprint(port))
	logger.Info().Str("serverUrl", serverURL).Msgf("Production server listening on %s", serverURL)

	// // Use the http.FileServer to serve the embedded assets.
	// http.Handle("/", http.FileServer(http.FS(assets)))

	// Register API routes.
	avs.RegisterAPIRoutes(*cfg)

	// Create the main context for p2p
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Open the pebble peer database.
	pdb, err := pebble.Open(cfg.PeerDatabasePath, &pebble.Options{Logger: &avs.PebbleNoopLogger{}})
	if err != nil {
		log.Error().Err(err).Str("db", cfg.PeerDatabasePath).Msg("could not open pebble peer database")
	}
	defer pdb.Close()

	// Open the pebble function database.
	fdb, err := pebble.Open(cfg.FunctionDatabasePath, &pebble.Options{Logger: &avs.PebbleNoopLogger{}})
	if err != nil {
		log.Error().Err(err).Str("db", cfg.FunctionDatabasePath).Msg("could not open pebble function database")
	}
	defer fdb.Close()

	// Boot P2P Network
	avs.RunP2P(ctx, logger, *cfg, done, failed, pdb, fdb)

	if !cfg.Headless {
		logger.Info().Msg("Opening browser")
		go func() {
			waitForServer(serverURL)
			openbrowser(serverURL)
		}()
	}

	// Start API in a separate goroutine.
	go func() {
		logger.Info().Str("port", cfg.API).Msg("Node API starting")
		err := http.Serve(listener, nil)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Warn().Err(err).Msg("Closed Server")
			close(failed)
		}
	}()

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

func waitForServer(url string) {
	for {
		// Attempt to connect to the server.
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close() // Don't forget to close the response body.
			logger.Info().Msg("App is Running. CTRL+C to quit.")
			return
		}
		// Close the unsuccessful response body to avoid leaking resources.
		if resp != nil {
			resp.Body.Close()
		}
		// Wait for a second before trying again.
		time.Sleep(1 * time.Second)
	}
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		logger.Fatal().Err(err)
	}
}

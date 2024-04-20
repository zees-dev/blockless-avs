package actions

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

	"github.com/urfave/cli/v2"
	"github.com/zees-dev/blockless-avs/core"
	avs "github.com/zees-dev/blockless-avs/node/pkg"
)

// //go:embed assets/*
// var embeddedFiles embed.FS

func RunAVS(c *cli.Context) error {
	app := core.GetAppConfig(c)
	avs.ParseFlags(app) // Parse flags (again) and set the AVS config in the app state.
	logger := app.Logger

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	failed := make(chan struct{})

	logger.Info().Bool("headless mode", !app.Headless).Msg("the server is starting")

	// assets, err := fs.Sub(embeddedFiles, "assets")
	// if err != nil {
	// 	logger.Info().Err(err).Msg("Failed to locate embedded assets")
	// 	return
	// }

	// Get the port the server is listening on.
	// Listen on a random port.
	listenHost := "127.0.0.1"
	if app.Headless {
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
	serverURL := fmt.Sprintf("http://127.0.0.1:%s", fmt.Sprint(port))

	// // Use the http.FileServer to serve the embedded assets.
	// http.Handle("/", http.FileServer(http.FS(assets)))

	// Register API routes.
	avs.RegisterAPIRoutes(app)

	// Create the main context for p2p
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// // TODO: B7s setup below
	// // Open the pebble peer database.
	// pdb, err := pebble.Open(cfg.PeerDatabasePath, &pebble.Options{Logger: &avs.PebbleNoopLogger{}})
	// if err != nil {
	// 	log.Error().Err(err).Str("db", cfg.PeerDatabasePath).Msg("could not open pebble peer database")
	// }
	// defer pdb.Close()

	// // Open the pebble function database.
	// fdb, err := pebble.Open(cfg.FunctionDatabasePath, &pebble.Options{Logger: &avs.PebbleNoopLogger{}})
	// if err != nil {
	// 	log.Error().Err(err).Str("db", cfg.FunctionDatabasePath).Msg("could not open pebble function database")
	// }
	// defer fdb.Close()

	// // Boot P2P Network
	// avs.RunP2P(ctx, logger, *cfg, done, failed, pdb, fdb)

	// if !app.Headless {
	// 	logger.Info().Msg("Opening browser")
	// 	go func() {
	// 		waitForServer(serverURL)
	// 		openbrowser(serverURL)
	// 	}()
	// }

	// Start API in a separate goroutine.
	go func() {
		logger.Info().Str("serverUrl", serverURL).Str("port", "8080").Msgf("Server listening on %s", serverURL)
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

func waitForServer(app *core.AppConfig, url string) {
	for {
		// Attempt to connect to the server.
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close() // Don't forget to close the response body.
			app.Logger.Info().Msg("App is Running. CTRL+C to quit.")
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

func openbrowser(app *core.AppConfig, url string) {
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
		app.Logger.Fatal().Err(err)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"github.com/urfave/cli/v2"
	avs "github.com/zees-dev/blockless-avs"
	node "github.com/zees-dev/blockless-avs/node/pkg"
)

// //go:embed assets/*
// var embeddedFiles embed.FS

func RunAVS(c *cli.Context) error {
	app := avs.GetAppConfig(c)
	node.ParseFlags(app) // Parse flags (again) and set the AVS config in the app state.
	logger := app.Logger

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	failed := make(chan struct{})

	logger.Info().Bool("headless mode", !app.Headless).Msg("the server is starting")

	router := http.NewServeMux()

	// assets, err := fs.Sub(embeddedFiles, "assets")
	// if err != nil {
	// 	logger.Info().Err(err).Msg("Failed to locate embedded assets")
	// 	return
	// }

	// // Use the http.FileServer to serve the embedded assets.
	// mux.Handle("/", http.FileServer(http.FS(assets)))

	// Register API routes.
	node.RegisterAPIRoutes(app, router)

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
	v1 := http.NewServeMux()
	v1.Handle("/v1/", http.StripPrefix("/v1", router))
	middlewares := Middlewares(app)
	server := &http.Server{
		Addr:    ":8080",
		Handler: middlewares(v1),
	}
	go func() {
		logger.Info().Msgf("Server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Warn().Err(err).Msg("Closed Server")
			close(failed)
		}
	}()

	select {
	case <-sig:
		logger.Info().Msg("Blockless AVS stopping")
	case <-done:
		logger.Info().Msg("Blockless AVS P2P done")
	case <-failed:
		logger.Info().Msg("Blockless AVS P2P aborted")
	}

	// If we receive a second interrupt signal, exit immediately.
	go func() {
		<-sig
		logger.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	return nil
}

type wrappedWriter struct {
	http.ResponseWriter
	status int
}

func (w *wrappedWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func Middlewares(app *avs.AppConfig) func(next http.Handler) http.Handler {
	logging := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &wrappedWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			app.Logger.Info().Int("status", wrapped.status).Str("method", r.Method).Str("path", r.URL.Path).Dur("duration", time.Since(start)).Msg("http request")
		})
	}

	// TODO: Add more middlewares here.

	return logging
}

func waitForServer(app *avs.AppConfig, url string) {
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

func openbrowser(app *avs.AppConfig, url string) {
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

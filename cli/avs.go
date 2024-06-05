package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/blocklessnetwork/b7s/executor"
	"github.com/blocklessnetwork/b7s/fstore"
	"github.com/blocklessnetwork/b7s/host"
	"github.com/blocklessnetwork/b7s/models/blockless"
	b7sNode "github.com/blocklessnetwork/b7s/node"
	"github.com/blocklessnetwork/b7s/peerstore"
	"github.com/blocklessnetwork/b7s/store"

	"github.com/cockroachdb/pebble"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	avs "github.com/zees-dev/blockless-avs"
	"github.com/zees-dev/blockless-avs/core"
	node "github.com/zees-dev/blockless-avs/node/pkg"
)

// //go:embed assets/*
// var embeddedFiles embed.FS

type PebbleNoopLogger struct{}

func (p *PebbleNoopLogger) Infof(_ string, _ ...any)  {}
func (p *PebbleNoopLogger) Fatalf(_ string, _ ...any) {}

func RunAVS(c *cli.Context) error {
	app := avs.GetCoreConfig(c)

	logger := app.Logger.(*core.ZeroLogger).Inner()

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	failed := make(chan struct{})

	logger.Info().Str("app_name", app.AppName).Bool("dev_mode", app.DevMode).Msg("starting avs..")

	// Determine node role.
	role, err := (func() (blockless.NodeRole, error) {
		switch app.BlocklessConfig.Role {
		case blockless.HeadNodeLabel:
			return blockless.HeadNode, nil
		case blockless.WorkerNodeLabel:
			return blockless.WorkerNode, nil
		default:
			return 0, errors.New("invalid node role")
		}
	})()
	if err != nil {
		logger.Error().Err(err).Str("role", app.BlocklessConfig.Role).Msg("invalid node role specified")
		return err
	}

	// Convert workspace path to an absolute one.
	workspace, err := filepath.Abs(app.BlocklessConfig.Workspace)
	if err != nil {
		logger.Error().Err(err).Str("path", app.BlocklessConfig.Workspace).Msg("could not determine absolute path for workspace")
		return err
	}
	app.BlocklessConfig.Workspace = workspace
	logger.Info().Str("workspace", app.BlocklessConfig.Workspace).Msg("workspace path")

	// Open the pebble peer database.
	logger.Info().Str("peer_database_path", app.BlocklessConfig.PeerDB)
	pdb, err := pebble.Open(app.BlocklessConfig.PeerDB, &pebble.Options{Logger: &PebbleNoopLogger{}})
	if err != nil {
		logger.Error().Err(err).Str("db", app.BlocklessConfig.PeerDB).Msg("could not open pebble peer database")
		return err
	}
	defer pdb.Close()

	// Create a new peer store.
	pstore := store.New(pdb)
	peerstore := peerstore.New(pstore)

	// Get the list of dial back peers.
	peers, err := peerstore.Peers()
	if err != nil {
		logger.Error().Err(err).Msg("could not get list of dial-back peers")
		return err
	}

	// Get the list of boot nodes addresses.
	bootNodeAddrs, err := func() ([]multiaddr.Multiaddr, error) {
		var out []multiaddr.Multiaddr
		for _, addr := range app.BlocklessConfig.BootNodes {
			addr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				return nil, fmt.Errorf("could not parse multiaddress (addr: %s): %w", addr, err)
			}
			out = append(out, addr)
		}
		return out, nil
	}()
	if err != nil {
		logger.Error().Err(err).Msg("could not get boot node addresses")
		return err
	}

	// Create libp2p host.
	host, err := host.New(
		*logger,
		app.BlocklessConfig.Connectivity.Address,
		app.BlocklessConfig.Connectivity.Port,
		host.WithPrivateKey(app.BlocklessConfig.Connectivity.PrivateKey),
		host.WithBootNodes(bootNodeAddrs),
		host.WithDialBackPeers(peers),
		host.WithDialBackAddress(app.BlocklessConfig.Connectivity.DialbackAddress),
		host.WithDialBackPort(app.BlocklessConfig.Connectivity.DialbackPort),
		host.WithDialBackWebsocketPort(app.BlocklessConfig.Connectivity.WebsocketDialbackPort),
		host.WithWebsocket(app.BlocklessConfig.Connectivity.Websocket),
		host.WithWebsocketPort(app.BlocklessConfig.Connectivity.WebsocketPort),
	)
	if err != nil {
		logger.Error().Err(err).Str("key", app.BlocklessConfig.Connectivity.PrivateKey).Msg("could not create host")
		return err
	}
	defer host.Close()

	logger.Info().
		Str("id", host.ID().String()).
		Strs("addresses", host.Addresses()).
		Int("boot_nodes", len(bootNodeAddrs)).
		Int("dial_back_peers", len(peers)).
		Msg("created b7s host")

	// Set node options.
	opts := []b7sNode.Option{
		b7sNode.WithRole(role),
		b7sNode.WithConcurrency(app.BlocklessConfig.Concurrency),
		b7sNode.WithAttributeLoading(app.BlocklessConfig.LoadAttributes),
	}

	// Open the pebble function database.
	logger.Info().Str("function_db_path", app.BlocklessConfig.FunctionDB)
	fdb, err := pebble.Open(app.BlocklessConfig.FunctionDB, &pebble.Options{Logger: &PebbleNoopLogger{}})
	if err != nil {
		logger.Error().Err(err).Str("db", app.BlocklessConfig.FunctionDB).Msg("could not open pebble function database")
		return err
	}

	// TODO:
	// If this is a worker node, initialize an executor.
	if role == blockless.WorkerNode {
		// Only worker nodes startup operator
		operatorApp := avs.GetOperatorConfig(c)

		// TODO: remove all redundant operator package code; we only need the signing bit (until this moves to DAPP)!
		// go func() {
		// 	// TODO: check operator registration
		// 	logger.Info().Msg("starting operator...")
		// 	if err := operatorApp.Operator.Start(c.Context); err != nil {
		// 		panic(err)
		// 	} else {
		// 		logger.Info().Msg("started operator")
		// 	}
		// }()

		// ensure runtime path and binary are set
		// TODO: convert relative runtime path to an absolute path.
		logger.Info().Str("runtime_path", app.BlocklessConfig.Worker.RuntimePath).Str("runtime_cli", app.BlocklessConfig.Worker.RuntimeCLI).Msg("worker node detected")
		if app.BlocklessConfig.Worker.RuntimePath == "" || app.BlocklessConfig.Worker.RuntimeCLI == "" {
			return errors.New("runtime path and binary are required for worker nodes")
		}

		// TODO: only worker nodes can run the browser.
		if !operatorApp.Headless {
			logger.Info().Msg("Opening browser...")
			// 	go func() {
			// 		waitForServer(serverURL)
			// 		openbrowser(serverURL)
			// 	}()
		}

		// Executor options.
		execOptions := []executor.Option{
			executor.WithWorkDir(app.BlocklessConfig.Workspace),
			executor.WithRuntimeDir(app.BlocklessConfig.Worker.RuntimePath),
			executor.WithExecutableName(app.BlocklessConfig.Worker.RuntimeCLI),
		}

		// TODO: limiter is not implemented yet.
		// check if limiter is required
		// if app.BlocklessConfig.Worker.CPUPercentageLimit != 1.0 || app.BlocklessConfig.Worker.MemoryLimitKB > 0 {
		// 	limiter, err := limits.New(limits.WithCPUPercentage(app.BlocklessConfig.Worker.CPUPercentageLimit), limits.WithMemoryKB(app.BlocklessConfig.Worker.MemoryLimitKB))
		// 	if err != nil {
		// 		logger.Error().Err(err).Msg("could not create resource limiter")
		// 		return err
		// 	}
		// 	defer func() {
		// 		if err = limiter.Shutdown(); err != nil {
		// 			log.Error().Err(err).Msg("could not shutdown resource limiter")
		// 		}
		// 	}()
		// 	execOptions = append(execOptions, executor.WithLimiter(limiter))
		// }

		// Create an executor.
		executor, err := executor.New(*logger, execOptions...)
		if err != nil {
			logger.Error().
				Err(err).
				Str("workspace", app.BlocklessConfig.Workspace).
				Str("runtime", app.BlocklessConfig.Worker.RuntimePath).
				Str("cli", app.BlocklessConfig.Worker.RuntimeCLI).
				Msg("could not create an executor")
			return err
		}

		opts = append(opts, b7sNode.WithExecutor(executor))
		opts = append(opts, b7sNode.WithWorkspace(app.BlocklessConfig.Workspace))
	}

	defer fdb.Close()
	functionStore := store.New(fdb)
	fstore := fstore.New(*logger, functionStore, app.BlocklessConfig.Workspace)

	// Instantiate node.
	b7s, err := b7sNode.New(*logger, host, peerstore, fstore, opts...)
	if err != nil {
		logger.Error().Err(err).Msg("could not create node")
		return err
	}

	// Start node main loop in a separate goroutine.
	go func() {
		logger.Info().Str("role", role.String()).Msg("blockless node starting..")
		if err := b7s.Run(c.Context); err != nil {
			logger.Error().Err(err).Msg("Blockless Node failed")
			close(failed)
		} else {
			close(done)
		}
		logger.Info().Msg("blockless Node stopped")
	}()

	// If we're a head node - start the REST API.
	if role == blockless.HeadNode {
		aggregatorApp := avs.GetAggregatorConfig(c)

		// start aggregator in the background
		go aggregatorApp.Aggregator.Start(c.Context)

		if app.BlocklessConfig.Connectivity.Address == "" {
			logger.Error().Err(err).Msg("REST API address is required")
			return err
		}
		if app.BlocklessConfig.Connectivity.Port == 0 {
			logger.Error().Err(err).Msg("REST API port is required")
			return err
		}

		// Create a channel to receive function execution results.
		sub := make(chan node.ExecuteFunctionResponse)

		// Create function registry; whitelist specific CIDs.
		// TODO: get this from operator config and/or CLI flags
		whitelistedCIDs := []string{"bafybeic222vtsk64qid6gtjfw33wvt27d76pshadk4c6yagcz2kidvjpkm"}
		functionReg := node.NewFunctionRegistry(&sub)
		functionReg.RegisterFunctions(whitelistedCIDs...)

		// submit results to the AVS.
		go func() {
			res := <-sub
			// TODO: migrate the following logic into another function
			err := func() error {
				// TODO: update and use the mapping from registry to determine on-chain contract function params to call
				if res.FunctionID != whitelistedCIDs[0] {
					return fmt.Errorf("received result for unknown function ID: %s", res.FunctionID)
				}

				logger.Info().Str("function_id", res.FunctionID).Str("request_id", res.RequestID).Msg("received function execution result")
				logger.Info().Any("results", res.Results).Msg("results")

				// TODO: convert the response to publishable data

				signedRes, err := SignResponse(logger, "bitcoin", 69000000000)
				if err != nil {
					return errors.Wrap(err, "could not sign oracle response")
				}

				if err = aggregatorApp.Aggregator.ProcessSignedOracleResponse(signedRes); err != nil {
					return errors.Wrap(err, "could not process signed oracle response")
				}

				// logger.Info().Any("signed_oracle_response", signedOracleResponse).Msg("oracle response signed")

				// TODO: integrate aggrator components into head-node functionality

				// TODO: on-chain submission of result(s)

				return nil
			}()
			if err != nil {
				logger.Error().Err(err).Msg("could not process function execution result")
			}
		}()

		// Register API routes.
		router := http.NewServeMux()
		node.RegisterAPIRoutes(b7s, app, router, functionReg)

		// Start API in a separate goroutine.
		v1 := http.NewServeMux()
		v1.Handle("/api/v1/", http.StripPrefix("/api/v1", router))
		middlewares := Middlewares(app)
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", app.BlocklessConfig.Connectivity.Address, app.BlocklessConfig.Connectivity.Port),
			Handler: middlewares(v1),
		}
		go func() {
			logger.Info().
				Str("address", app.BlocklessConfig.Connectivity.Address).
				Uint("port", app.BlocklessConfig.Connectivity.Port).
				Msg("head node server started...")
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Warn().Err(err).Msg("Closed Server")
				close(failed)
			}
		}()
	}

	// Wait for a signal to stop the server.
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

func Middlewares(app *avs.CoreConfig) func(next http.Handler) http.Handler {
	logging := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &wrappedWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			app.Logger.Info("status", wrapped.status, "method", r.Method, "path", r.URL.Path, "duration", time.Since(start), "http request")
		})
	}

	// TODO: Add more middlewares here.

	return logging
}

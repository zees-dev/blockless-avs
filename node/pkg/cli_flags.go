package pkg

import "github.com/urfave/cli/v2"

func serverFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "log-level",
			Value:   "info",
			Usage:   "log level to use",
			Aliases: []string{"l"},
		},

		// Node configuration.
		&cli.StringFlag{
			Name:    "role",
			Value:   defaultRole,
			Usage:   "role this node will have in the Blockless protocol (head or worker)",
			Aliases: []string{"r"},
		},
		&cli.StringFlag{
			Name:  "peer-db",
			Value: defaultPeerDB,
			Usage: "path to the database used for persisting peer data",
		},
		&cli.StringFlag{
			Name:  "function-db",
			Value: defaultFunctionDB,
			Usage: "path to the database used for persisting function data",
		},
		&cli.UintFlag{
			Name:    "concurrency",
			Value:   defaultConcurrency,
			Usage:   "maximum number of requests node will process in parallel",
			Aliases: []string{"c"},
		},
		&cli.StringFlag{
			Name:  "rest-api",
			Value: "0",
			Usage: "address where the web server will listen on",
		},
		&cli.StringFlag{
			Name:  "workspace",
			Value: "./workspace",
			Usage: "directory that the node can use for file storage",
		},
		&cli.StringFlag{
			Name:  "runtime-path",
			Usage: "runtime path (used by the worker node)",
		},
		&cli.StringFlag{
			Name:  "runtime-cli",
			Usage: "runtime path (used by the worker node)",
		},
		&cli.BoolFlag{
			Name:  "attributes",
			Usage: "node should try to load its attribute data from IPFS",
		},
		&cli.StringSliceFlag{
			Name:  "topic",
			Usage: "topics node should subscribe to",
		},

		// Host configuration.
		&cli.StringFlag{
			Name:  "private-key",
			Usage: "private key that the b7s host will use",
		},
		&cli.StringFlag{
			Name:    "address",
			Value:   defaultAddress,
			Usage:   "address that the b7s host will use",
			Aliases: []string{"a"},
		},
		&cli.UintFlag{
			Name:    "port",
			Value:   defaultPort,
			Usage:   "port that the b7s host will use",
			Aliases: []string{"p"},
		},
		&cli.StringSliceFlag{
			Name:  "boot-nodes",
			Usage: "list of addresses that this node will connect to on startup, in multiaddr format",
		},

		// For external IPs.
		&cli.StringFlag{
			Name:  "dialback-address",
			Value: defaultAddress,
			Usage: "external address that the b7s host will advertise",
		},
		&cli.UintFlag{
			Name:  "dialback-port",
			Value: defaultPort,
			Usage: "external port that the b7s host will advertise",
		},
		&cli.UintFlag{
			Name:  "websocket-dialback-port",
			Value: defaultPort,
			Usage: "external port that the b7s host will advertise for websocket connections",
		},

		// Websocket connection.
		&cli.BoolFlag{
			Name:    "websocket",
			Value:   defaultUseWebsocket,
			Usage:   "should the node use websocket protocol for communication",
			Aliases: []string{"w"},
		},
		&cli.UintFlag{
			Name:  "websocket-port",
			Value: defaultPort,
			Usage: "port to use for websocket connections",
		},

		// Limit configuration.
		&cli.Float64Flag{
			Name:  "cpu-percentage-limit",
			Value: 1.0,
			Usage: "amount of CPU time allowed for Blockless Functions in the 0-1 range, 1 being unlimited",
		},
		&cli.Int64Flag{
			Name:  "memory-limit",
			Value: 0,
			Usage: "memory limit (kB) for Blockless Functions",
		},

		// AVS/dApp flags
		&cli.BoolFlag{
			Name:  "headless",
			Value: true,
			Usage: "Run in headless mode without opening the browser",
		},
		&cli.BoolFlag{
			Name:    "devmode",
			Usage:   "Run in development mode",
			Aliases: []string{"d"},
		},
	}
}

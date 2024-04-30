package pkg

import (
	"github.com/blocklessnetwork/b7s/config"
	"github.com/blocklessnetwork/b7s/models/blockless"
	"github.com/blocklessnetwork/b7s/node"
	"github.com/urfave/cli/v2"
)

// Default values.
const (
	defaultPort         = 0
	defaultAddress      = "0.0.0.0"
	defaultPeerDB       = "./node/peer-db"
	defaultFunctionDB   = "./node/function-db"
	defaultConcurrency  = uint(node.DefaultConcurrency)
	defaultUseWebsocket = false
	defaultRole         = "worker"
	defaultWorkspace    = "./node/workspace"
)

var (
	Role = &cli.StringFlag{
		Name:       "role",
		Required:   true,
		Usage:      "role this note will have in the Blockless protocol (head or worker)",
		Value:      blockless.HeadNodeLabel,
		HasBeenSet: true,
	}
	PeerDatabasePath = &cli.StringFlag{
		Name:       "peer-db",
		Required:   true,
		Usage:      "path to the database used for persisting peer data",
		Value:      defaultPeerDB,
		HasBeenSet: true,
	}
	FunctionDatabasePath = &cli.StringFlag{
		Name:       "function-db",
		Required:   true,
		Usage:      "path to the database used for persisting function data",
		Value:      defaultFunctionDB,
		HasBeenSet: true,
	}
	Workspace = &cli.StringFlag{
		Name:       "workspace",
		Required:   true,
		Usage:      "directory that the node can use for file storage",
		Value:      defaultWorkspace,
		HasBeenSet: true,
	}
	Concurrency = &cli.UintFlag{
		Name:       "concurrency",
		Required:   true,
		Usage:      "maximum number of requests node will process in parallel",
		Value:      defaultConcurrency,
		HasBeenSet: true,
	}
	LoadAttributes = &cli.BoolFlag{
		Name:       "attributes",
		Required:   true,
		Usage:      "node should try to load its attribute data from IPFS",
		Value:      false,
		HasBeenSet: true,
	}
	PrivateKey = &cli.StringFlag{
		Name:       "private-key",
		Required:   true,
		Usage:      "private key that the b7s host will use",
		Value:      "",   // TODO
		HasBeenSet: true, // TODO
	}
	HostAddress = &cli.StringFlag{
		Name:       "address",
		Required:   true,
		Usage:      "address that the b7s host will use",
		Value:      defaultAddress,
		HasBeenSet: true,
	}
	HostPort = &cli.UintFlag{
		Name:       "port",
		Required:   true,
		Usage:      "port that the b7s host will use",
		Value:      defaultPort,
		HasBeenSet: true,
	}
	BootNodes = &cli.StringSliceFlag{
		Name: "boot-nodes",
		// Required:   true,
		Usage: "list of addresses that this node will connect to on startup, in multiaddr format",
		// Value:      nil,
		// HasBeenSet: true,
	}
	DialBackAddress = &cli.StringFlag{
		Name: "dialback-address",
		// Required:   true,
		Usage: "external address that the b7s host will advertise",
		// Value:      "",
		// HasBeenSet: true,
	}
	DialBackPort = &cli.UintFlag{
		Name: "dialback-port",
		// Required:   true,
		Usage: "external port that the b7s host will advertise",
		// Value:      defaultPort,
		// HasBeenSet: true,
	}
	DialBackWebsocketPort = &cli.UintFlag{
		Name: "websocket-dialback-port",
		// Required:   true,
		Usage: "external port that the b7s host will advertise for websocket connections",
		// Value:      defaultPort,
		// HasBeenSet: true,
	}
	Websocket = &cli.BoolFlag{
		Name: "websocket",
		// Required:   true,
		Usage: "should the node use websocket protocol for communication",
		// Value:      defaultUseWebsocket,
		// HasBeenSet: true,
	}
	WebsocketPort = &cli.UintFlag{
		Name: "websocket-port",
		// Required:   true,
		Usage: "port to use for websocket connections",
		// Value:      defaultPort,
		// HasBeenSet: true,
	}
	CPUPercentage = &cli.Float64Flag{
		Name: "cpu-percentage-limit",
		// Required:   true,
		Usage:      "amount of CPU time allowed for Blockless Functions in the 0-1 range, 1 being unlimited",
		Value:      1.0,
		HasBeenSet: true,
	}
	MemoryMaxKB = &cli.Int64Flag{
		Name: "memory-limit",
		// Required:   true,
		Usage:      "memory limit (kB) for Blockless Functions",
		Value:      0,
		HasBeenSet: true,
	}
)

func ParseFlags(c *cli.Context) config.Config {
	role := c.String(Role.Name)
	peerDB := c.String(PeerDatabasePath.Name)
	functionDB := c.String(FunctionDatabasePath.Name)
	concurrency := c.Uint(Concurrency.Name)
	loadAttributes := c.Bool(LoadAttributes.Name)
	privateKey := c.String(PrivateKey.Name)
	hostAddress := c.String(HostAddress.Name)
	hostPort := c.Uint(HostPort.Name)
	bootNodes := c.StringSlice(BootNodes.Name)
	dialbackAddress := c.String(DialBackAddress.Name)
	dialbackPort := c.Uint(DialBackPort.Name)
	websocket := c.Bool(Websocket.Name)
	websocketPort := c.Uint(WebsocketPort.Name)
	websocketDialbackPort := c.Uint(DialBackWebsocketPort.Name)
	// cpuPercentage := c.Float64(CPUPercentage.Name)
	// memoryMaxKB := c.Int64(MemoryMaxKB.Name)

	// pflag.StringVar(&cfg.RuntimePath, "runtime-path", "", "runtime path (used by the worker node)")
	// pflag.StringVar(&cfg.RuntimeCLI, "runtime-cli", "", "runtime path (used by the worker node)")

	return config.Config{
		Role:           role,
		PeerDB:         peerDB,
		FunctionDB:     functionDB,
		Concurrency:    concurrency,
		LoadAttributes: loadAttributes,
		BootNodes:      bootNodes,
		Connectivity: config.Connectivity{
			PrivateKey:            privateKey,
			Address:               hostAddress,
			Port:                  hostPort,
			DialbackAddress:       dialbackAddress,
			DialbackPort:          dialbackPort,
			Websocket:             websocket,
			WebsocketPort:         websocketPort,
			WebsocketDialbackPort: websocketDialbackPort,
		},
	}
}

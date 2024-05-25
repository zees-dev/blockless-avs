package pkg

import (
	"github.com/blocklessnetwork/b7s/config"
	"github.com/blocklessnetwork/b7s/models/blockless"
	"github.com/blocklessnetwork/b7s/node"
	"github.com/urfave/cli/v2"
)

// Default values.
const (
	defaultPort         = 6000
	defaultAddress      = "0.0.0.0"
	defaultPeerDB       = "./data/peer-db"
	defaultFunctionDB   = "./data/function-db"
	defaultConcurrency  = uint(node.DefaultConcurrency)
	defaultUseWebsocket = false
	defaultRole         = "worker"
	defaultWorkspace    = "./data/workspace"
	// defaultRuntimePath  = "./node/workspace"
	defaultRuntimePath = "/Users/z/Desktop/blockless/blocklessnetwork/runtime/target/debug"
	defaultRuntimeCLI  = "bls-runtime"
)

var (
	Role = &cli.StringFlag{
		Name:       "role",
		Required:   true,
		Usage:      "role this note will have in the Blockless protocol (head or worker)",
		Value:      blockless.WorkerNodeLabel,
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
		Usage:      "p2p port that the b7s host will use; head nodes also use this for api server",
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
	RuntimePath = &cli.StringFlag{
		Name: "runtime-path",
		// Required:   true,
		Usage:      "Blockless Runtime location (used by the worker node)",
		Value:      defaultRuntimePath,
		HasBeenSet: true,
	}
	RuntimeCLI = &cli.StringFlag{
		Name: "runtime-cli",
		// Required:   true,
		Usage:      "Blockless Runtime binary (used by the worker node)",
		Value:      defaultRuntimeCLI,
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

	// TODO: fix this
	// worker config
	// runtimePath := c.String(RuntimePath.Name)
	// runtimeCLI := c.String(RuntimeCLI.Name)
	// cpuPercentage := c.Float64(CPUPercentage.Name)
	// memoryMaxKB := c.Int64(MemoryMaxKB.Name)

	return config.Config{
		Role:           role,
		PeerDB:         peerDB,
		FunctionDB:     functionDB,
		Concurrency:    concurrency,
		Workspace:      defaultWorkspace,
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
		Worker: config.Worker{
			// RuntimePath: runtimePath,
			// RuntimeCLI:  runtimeCLI,
			RuntimePath: defaultRuntimePath,
			RuntimeCLI:  defaultRuntimeCLI,
		},
	}
}

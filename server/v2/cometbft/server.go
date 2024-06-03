package cometbft

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	abciserver "github.com/cometbft/cometbft/abci/server"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"cosmossdk.io/core/log"
	"cosmossdk.io/core/transaction"
	serverv2 "cosmossdk.io/server/v2"
	"cosmossdk.io/server/v2/appmanager"
	"cosmossdk.io/server/v2/cometbft/handlers"
	cometlog "cosmossdk.io/server/v2/cometbft/log"
	"cosmossdk.io/server/v2/cometbft/mempool"
	"cosmossdk.io/server/v2/cometbft/types"
	"cosmossdk.io/store/v2/snapshots"

	corectx "cosmossdk.io/core/context"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/pelletier/go-toml/v2"
)

const (
	flagWithComet     = "with-comet"
	flagAddress       = "address"
	flagTransport     = "transport"
	flagTraceStore    = "trace-store"
	flagCPUProfile    = "cpu-profile"
	FlagMinGasPrices  = "minimum-gas-prices"
	FlagQueryGasLimit = "query-gas-limit"
	FlagHaltHeight    = "halt-height"
	FlagHaltTime      = "halt-time"
	FlagTrace         = "trace"
)

var _ serverv2.ServerModule[transaction.Tx] = (*CometBFTServer[transaction.Tx])(nil)

type CometBFTServer[T transaction.Tx] struct {
	Node   *node.Node
	App    *Consensus[T]
	logger log.Logger

	config    Config
	cleanupFn func()
}

// App is an interface that represents an application in the CometBFT server.
// It provides methods to access the app manager, logger, and store.
type App[T transaction.Tx] interface {
	GetApp() *appmanager.AppManager[T]
	GetLogger() log.Logger
	GetStore() types.Store
}

func New[T transaction.Tx](home string, txCodec transaction.Codec[T]) *CometBFTServer[T] {
	// Write default cmt config
	configPath := filepath.Join(home, "config")
	configFilePath := filepath.Join(configPath, "config.toml")

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		if err := os.MkdirAll(configPath, os.ModePerm); err != nil {
			return nil
		}
	}

	cometConfig := cmtcfg.DefaultConfig()
	cmtcfg.WriteConfigFile(configFilePath, cometConfig)

	consensus := &Consensus[T]{txCodec: txCodec}
	return &CometBFTServer[T]{
		App: consensus,
	}
}

func (s *CometBFTServer[T]) Init(appI serverv2.App[T], v *viper.Viper, logger log.Logger) (serverv2.ServerModule[T], error) {
	app := appI.Application
	store := appI.Store.(types.Store)

	cfg := Config{CmtConfig: serverv2.GetConfigFromViper(v), ConsensusAuthority: app.GetConsensusAuthority()}
	logger = logger.With("module", "cometbft-server")

	// create noop mempool
	mempool := mempool.NoOpMempool[T]{}

	// create consensus
	// txCodec should be in server from New()
	consensus := NewConsensus[T](app.GetAppManager(), mempool, store, cfg, s.App.txCodec, logger)

	consensus.SetPrepareProposalHandler(handlers.NoOpPrepareProposal[T]())
	consensus.SetProcessProposalHandler(handlers.NoOpProcessProposal[T]())
	consensus.SetVerifyVoteExtension(handlers.NoOpVerifyVoteExtensionHandler())
	consensus.SetExtendVoteExtension(handlers.NoOpExtendVote())

	// TODO: set these; what is the appropriate presence of the Store interface here?
	var ss snapshots.StorageSnapshotter
	var sc snapshots.CommitSnapshotter

	snapshotStore, err := GetSnapshotStore(cfg.CmtConfig.RootDir)
	if err != nil {
		panic(err)
	}

	sm := snapshots.NewManager(snapshotStore, snapshots.SnapshotOptions{}, sc, ss, nil, logger) // TODO: set options somehow
	consensus.SetSnapshotManager(sm)

	s.config = cfg
	s.App = consensus
	s.logger = logger

	return &CometBFTServer[T]{
		logger: logger,
		App:    consensus,
		config: cfg,
	}, nil
}

func NewCometBFTServer[T transaction.Tx](
	app *appmanager.AppManager[T],
	store types.Store,
	logger log.Logger,
	cfg Config,
	txCodec transaction.Codec[T],
) *CometBFTServer[T] {
	logger = logger.With("module", "cometbft-server")

	// create noop mempool
	mempool := mempool.NoOpMempool[T]{}

	// create consensus
	consensus := NewConsensus[T](app, mempool, store, cfg, txCodec, logger)

	consensus.SetPrepareProposalHandler(handlers.NoOpPrepareProposal[T]())
	consensus.SetProcessProposalHandler(handlers.NoOpProcessProposal[T]())
	consensus.SetVerifyVoteExtension(handlers.NoOpVerifyVoteExtensionHandler())
	consensus.SetExtendVoteExtension(handlers.NoOpExtendVote())

	// TODO: set these; what is the appropriate presence of the Store interface here?
	var ss snapshots.StorageSnapshotter
	var sc snapshots.CommitSnapshotter

	snapshotStore, err := GetSnapshotStore(cfg.CmtConfig.RootDir)
	if err != nil {
		panic(err)
	}

	sm := snapshots.NewManager(snapshotStore, snapshots.SnapshotOptions{}, sc, ss, nil, logger) // TODO: set options somehow
	consensus.SetSnapshotManager(sm)

	return &CometBFTServer[T]{
		logger: logger,
		App:    consensus,
		config: cfg,
	}
}

func (s *CometBFTServer[T]) Name() string {
	return "cometbft"
}

func (s *CometBFTServer[T]) Start(ctx context.Context) error {
	viper := ctx.Value(corectx.ViperContextKey{}).(*viper.Viper)
	cometConfig := serverv2.GetConfigFromViper(viper)

	wrappedLogger := cometlog.CometLoggerWrapper{Logger: s.logger}
	if s.config.Standalone {
		svr, err := abciserver.NewServer(s.config.Addr, s.config.Transport, s.App)
		if err != nil {
			return fmt.Errorf("error creating listener: %w", err)
		}

		svr.SetLogger(wrappedLogger)

		return svr.Start()
	}

	nodeKey, err := p2p.LoadOrGenNodeKey(cometConfig.NodeKeyFile())
	if err != nil {
		return err
	}

	s.Node, err = node.NewNode(
		ctx,
		cometConfig,
		pvm.LoadOrGenFilePV(cometConfig.PrivValidatorKeyFile(), cometConfig.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(s.App),
		getGenDocProvider(cometConfig),
		cmtcfg.DefaultDBProvider,
		node.DefaultMetricsProvider(cometConfig.Instrumentation),
		wrappedLogger,
	)
	if err != nil {
		return err
	}

	s.cleanupFn = func() {
		if s.Node != nil && s.Node.IsRunning() {
			_ = s.Node.Stop()
		}
	}

	return s.Node.Start()
}

func (s *CometBFTServer[T]) Stop(_ context.Context) error {
	defer s.cleanupFn()
	if s.Node != nil {
		return s.Node.Stop()
	}
	return nil
}

// func (s *CometBFTServer[T]) Config() any {
// 	return cmtcfg.DefaultConfig()
// }

// func (s *CometBFTServer[T]) WriteConfig(configPath string) error {
// 	cfg := s.Config()
// 	b, err := toml.Marshal(cfg)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal config: %w", err)
// 	}

// 	if err := os.WriteFile(filepath.Join(configPath, "config.toml"), b, 0o666); err != nil {
// 		return fmt.Errorf("failed to write config: %w", err)
// 	}
// 	return nil
// }

// returns a function which returns the genesis doc from the genesis file.
func getGenDocProvider(cfg *cmtcfg.Config) func() (node.ChecksummedGenesisDoc, error) {
	return func() (node.ChecksummedGenesisDoc, error) {
		appGenesis, err := genutiltypes.AppGenesisFromFile(cfg.GenesisFile())
		if err != nil {
			return node.ChecksummedGenesisDoc{
				Sha256Checksum: []byte{},
			}, err
		}

		gen, err := appGenesis.ToGenesisDoc()
		if err != nil {
			return node.ChecksummedGenesisDoc{
				Sha256Checksum: []byte{},
			}, err
		}
		genbz, err := gen.AppState.MarshalJSON()
		if err != nil {
			return node.ChecksummedGenesisDoc{
				Sha256Checksum: []byte{},
			}, err
		}

		bz, err := json.Marshal(genbz)
		if err != nil {
			return node.ChecksummedGenesisDoc{
				Sha256Checksum: []byte{},
			}, err
		}
		sum := sha256.Sum256(bz)

		return node.ChecksummedGenesisDoc{
			GenesisDoc:     gen,
			Sha256Checksum: sum[:],
		}, nil
	}
}

func (s *CometBFTServer[T]) StartCmdFlags() pflag.FlagSet {
	flags := *pflag.NewFlagSet("cometbft", pflag.ExitOnError)
	flags.Bool(flagWithComet, true, "Run abci app embedded in-process with CometBFT")
	flags.String(flagAddress, "tcp://127.0.0.1:26658", "Listen address")
	flags.String(flagTransport, "socket", "Transport protocol: socket, grpc")
	flags.String(flagTraceStore, "", "Enable KVStore tracing to an output file")
	flags.String(FlagMinGasPrices, "", "Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)")
	flags.Uint64(FlagQueryGasLimit, 0, "Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.")
	flags.Uint64(FlagHaltHeight, 0, "Block height at which to gracefully halt the chain and shutdown the node")
	flags.Uint64(FlagHaltTime, 0, "Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node")
	flags.String(flagCPUProfile, "", "Enable CPU profiling and write to the provided file")
	flags.Bool(FlagTrace, false, "Provide full stack traces for errors in ABCI Log")
	return flags
}

func (s *CometBFTServer[T]) CLICommands() serverv2.CLIConfig {
	return serverv2.CLIConfig{
		Commands: []*cobra.Command{
			s.StatusCommand(),
			s.ShowNodeIDCmd(),
			s.ShowValidatorCmd(),
			s.ShowAddressCmd(),
			s.VersionCmd(),
			s.QueryBlockCmd(),
			s.QueryBlocksCmd(),
			s.QueryBlockResultsCmd(),
			cmtcmd.ResetAllCmd,
			cmtcmd.ResetStateCmd,
		},
	}
}

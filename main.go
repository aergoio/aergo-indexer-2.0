package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	indx "github.com/aergoio/aergo-indexer-2.0/indexer"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-lib/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "indexer",
		Short: "Aergo Indexer",
		Long:  "Aergo Metadata Indexer",
		Run:   rootRun,
	}

	runMode                string
	checkMode              bool
	cleanMode              bool
	host                   string
	port                   int32
	dbURL                  string
	prefix                 string
	networkType            string
	aergoAddress           string
	startFrom              uint64
	stopAt                 uint64
	batchTime              int32
	bulkSize               int32
	minerNum               int
	grpcNum                int
	whiteListAddress       []string
	whiteListBlockInterval uint64

	logger  *log.Logger
	indexer *indx.Indexer
)

func init() {
	fs := rootCmd.PersistentFlags()
	fs.StringVarP(&host, "host", "H", "localhost", "host address of aergo server")
	fs.Int32VarP(&port, "port", "p", 7845, "port number of aergo server")
	fs.StringVarP(&aergoAddress, "aergo", "A", "", "host and port of aergo server. Alternative to setting host and port separately.")
	fs.StringVarP(&dbURL, "dburl", "E", "localhost:9200", "Database URL")
	fs.StringVarP(&prefix, "prefix", "P", "", "prefix used for index names. if not set, use network type.")
	fs.StringVarP(&networkType, "network", "N", "testnet", "network type. mainnet or testnet")

	fs.BoolVar(&checkMode, "check", false, "check and fix indices of range of heights")
	fs.BoolVar(&cleanMode, "clean", false, "clean unexpected data in index")
	fs.StringVarP(&runMode, "mode", "M", "", "indexer running mode. Alternative to setting check, clean, onsync separately.")
	fs.Uint64Var(&startFrom, "from", 0, "start syncing from this block number")
	fs.Uint64Var(&stopAt, "to", 0, "stop syncing at this block number")
	fs.Int32Var(&bulkSize, "bulk", 4000, "bulk size")
	fs.Int32Var(&batchTime, "batch", 60, "batch duration")
	fs.IntVar(&minerNum, "miner", 32, "number of miner")
	fs.IntVar(&grpcNum, "grpc", 16, "number of miner")

	fs.StringSliceVarP(&whiteListAddress, "whitelist", "W", []string{}, "address for indexing whitelist balance, onsync only")
	fs.Uint64VarP(&whiteListBlockInterval, "whitelist_block_interval", "B", 1000, "block interval for indexing whitelist balance, onsync only")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func rootRun(cmd *cobra.Command, args []string) {
	logger = log.NewLogger("indexer")
	logger.Info().Msg("Starting indexer for SCAN 2.0 ...")

	var clusterMode bool
	if networkType == "mainnet" {
		clusterMode = true // init es mappings with cluster mode ( mainnet only )
	}
	doc.InitEsMappings(clusterMode)

	// init indexer
	indexer, err := indx.NewIndexer(
		indx.SetServerAddr(getServerAddress()),
		indx.SetDBAddr(dbURL),
		indx.SetPrefix(prefix),
		indx.SetNetworkType(networkType),
		indx.SetRunMode(getRunMode()),
		indx.SetLogger(logger),
		indx.SetBulkSize(bulkSize),
		indx.SetBatchTime(batchTime),
		indx.SetMinerNum(minerNum),
		indx.SetGrpcNum(grpcNum),
		indx.SetWhiteListAddresses(whiteListAddress),
		indx.SetWhiteListBlockInterval(whiteListBlockInterval),
	)
	if err != nil {
		logger.Warn().Err(err).Str("dbURL", dbURL).Msg("Could not start indexer")
		return
	}

	// start indexer
	exitOnComplete := indexer.Start(startFrom, stopAt)
	if exitOnComplete == true {
		return
	}

	handleKillSig(func() {
		indexer.Stop()
	}, logger)

	for {
		time.Sleep(time.Second)
	}
}

func getServerAddress() string {
	if len(aergoAddress) > 0 {
		return aergoAddress
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func getRunMode() string {
	if len(runMode) > 0 {
		return runMode
	} else if checkMode {
		return "check"
	} else if cleanMode {
		return "clean"
	}
	return "onsync"
}

func handleKillSig(handler func(), logger *log.Logger) {
	sigChannel := make(chan os.Signal, 1)

	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		for signal := range sigChannel {
			logger.Info().Msgf("Receive signal %s, Shutting down...", signal)
			handler()
			os.Exit(1)
		}
	}()
}

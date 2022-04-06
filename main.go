package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	indx "github.com/kjunblk/aergo-indexer/indexer"
	"github.com/kjunblk/aergo-indexer/types"
	"github.com/aergoio/aergo-lib/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	rootCmd = &cobra.Command{
		Use:   "indexer",
		Short: "Aergo Indexer",
		Long:  "Aergo Metadata Indexer",
		Run:   rootRun,
	}

	checkMode	bool
	rebuildMode	bool
	host            string
	port            int32
	dbURL           string
	indexNamePrefix string
	aergoAddress    string
	startFrom       uint64
	stopAt          uint64
	batchTime       int32
	bulkSize        int32
	minerNum	int

	logger *log.Logger

	client  types.AergoRPCServiceClient
	indexer *indx.Indexer
)

func init() {
	fs := rootCmd.PersistentFlags()
	fs.StringVarP(&host, "host", "H", "localhost", "host address of aergo server")
	fs.Int32VarP(&port, "port", "p", 7845, "port number of aergo server")
	fs.StringVarP(&aergoAddress, "aergo", "A", "", "host and port of aergo server. Alternative to setting host and port separately.")
	fs.StringVarP(&dbURL, "dburl", "E", "http://localhost:9200", "Database URL")
	fs.StringVarP(&indexNamePrefix, "prefix", "X", "testnet_", "prefix used for index names")
	fs.BoolVar(&checkMode, "check", false, "check and fix indices of range of heights")
	fs.BoolVar(&rebuildMode, "rebuild", false, "reindex all with batch job")
	fs.Uint64VarP(&startFrom, "from", "", 0, "start syncing from this block number")
	fs.Uint64VarP(&stopAt, "to", "", 0, "stop syncing at this block number")
	fs.Int32VarP(&bulkSize, "bulk", "", 0, "bulk size")
	fs.Int32VarP(&batchTime, "batch", "", 0, "batch duration")
	fs.IntVarP(&minerNum, "miner", "", 0, "number of miner")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func rootRun(cmd *cobra.Command, args []string) {

	logger = log.NewLogger("indexer")
	logger.Info().Msg("Starting indexer for SCAN 2.0 ...")

	client = waitForClient(getServerAddress())
	indexer, err := indx.NewIndexer(client, logger, dbURL, indexNamePrefix)

	if err != nil {
		logger.Warn().Err(err).Str("dbURL", dbURL).Msg("Could not start indexer")
		return
	} else {
		indexer.BulkSize = bulkSize
		indexer.BatchTime = time.Duration(batchTime) * time.Second
		indexer.StartHeight = startFrom
		indexer.MinerNum = int(minerNum)
	}

	if (checkMode) {
		err = indexer.RunCheckIndex(startFrom, stopAt)
		return
	} else if (rebuildMode) {
		err = indexer.Rebuild()
		return
	} else {
		err = indexer.OnSync(startFrom, stopAt)
	}

	if err != nil {
		logger.Warn().Err(err).Str("dbURL", dbURL).Msg("Could not start indexer")
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


func waitForClient(serverAddr string) types.AergoRPCServiceClient {
	var conn *grpc.ClientConn
	var err error

	for {
		ctx := context.Background()
		maxMsgSize := 1024 * 1024 * 10 // 10mb
		conn, err = grpc.DialContext(ctx, serverAddr,
			grpc.WithInsecure(),
			grpc.WithBlock(),
			grpc.WithTimeout(5*time.Second),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)),
		)

		if err == nil && conn != nil {
			break
		}

		logger.Info().Str("serverAddr", serverAddr).Err(err).Msg("Could not connect to aergo server, retrying")
		time.Sleep(time.Second)
	}
	logger.Info().Str("serverAddr", serverAddr).Msg("Connected to aergo server")
	return types.NewAergoRPCServiceClient(conn)
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


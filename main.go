package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aergoio/aergo-indexer-2.0/indexer"
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

	runMode    string
	checkMode  bool
	onsyncMode bool
	fix        bool

	host                    string
	port                    int32
	dbURL                   string
	prefix                  string
	aergoAddress            string
	cluster                 bool
	from                    uint64
	to                      uint64
	cccvNftServerType       string
	balanceWhitelist        []string
	tokenVerifyAddress      string
	contractVerifyAddress   string
	tokenVerifyWhitelist    []string
	contractVerifyWhitelist []string

	logger *log.Logger
)

func init() {
	fs := rootCmd.PersistentFlags()
	fs.StringVarP(&host, "host", "H", "localhost", "host address of aergo server")
	fs.Int32VarP(&port, "port", "p", 7845, "port number of aergo server")
	fs.StringVarP(&aergoAddress, "aergo", "A", "", "host and port of aergo server. Alternative to setting host and port separately")
	fs.StringVarP(&dbURL, "dburl", "E", "localhost:9200", "Database URL")
	fs.StringVarP(&prefix, "prefix", "P", "testnet", "index name prefix")
	fs.BoolVarP(&cluster, "cluster", "C", false, "elasticsearch cluster type")
	fs.BoolVar(&fix, "fix", false, "fix mode to overwrite data")
	fs.BoolVar(&checkMode, "check", true, "check indices of range of heights")
	fs.BoolVar(&onsyncMode, "onsync", true, "onsync data in indices")
	fs.StringVarP(&runMode, "mode", "M", "", "indexer running mode(all,check,onsync) Alternative to setting check, onsync separately")
	fs.Uint64Var(&from, "from", 0, "start syncing from this block number")
	fs.Uint64Var(&to, "to", 0, "stop syncing at this block number")

	fs.StringVar(&cccvNftServerType, "cccv", "", "indexing cccv nft by network type.(mainnet,testnet)")
	fs.StringVarP(&tokenVerifyAddress, "token", "t", "", "address for query verified token")
	fs.StringVarP(&contractVerifyAddress, "contract", "c", "", "address for query contract code")
	fs.StringSliceVarP(&balanceWhitelist, "balance_whitelist", "W", []string{}, "whitelist for update account balance")
	fs.StringArrayVar(&tokenVerifyWhitelist, "token_whitelist", []string{}, "whitelist for update verified token")
	fs.StringArrayVar(&contractVerifyWhitelist, "contract_whitelist", []string{}, "whitelist for update verified contract")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func rootRun(cmd *cobra.Command, args []string) {
	logger = log.NewLogger("indexer")
	logger.Info().Msg("Starting indexer for SCAN 2.0 ...")

	doc.InitEsMappings(cluster)

	// init indexer
	indx, err := indexer.NewIndexer(
		indexer.SetServerAddr(getServerAddress()),
		indexer.SetDBAddr(dbURL),
		indexer.SetPrefix(prefix),
		indexer.SetNetworkTypeForCccv(cccvNftServerType),
		indexer.SetRunMode(getRunMode()),
		indexer.SetFix(fix),
		indexer.SetLogger(logger),
		indexer.SetWhiteListAddresses(balanceWhitelist),
		indexer.SetTokenVerifyAddress(tokenVerifyAddress),
		indexer.SetContractVerifyAddress(contractVerifyAddress),
		indexer.SetTokenVerifyWhitelist(tokenVerifyWhitelist),
		indexer.SetContractVerifyWhitelist(contractVerifyWhitelist),
	)
	if err != nil {
		logger.Warn().Err(err).Msg("Could not start indexer")
		return
	}

	// start indexer
	exitOnComplete := indx.Start(from, to)
	if exitOnComplete == true {
		return
	}

	interrupt := handleKillSig(func() {
		indx.Stop()
	}, logger)

	// Wait main routine to stop
	<-interrupt.C
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
	} else if onsyncMode && checkMode {
		return "all"
	} else if checkMode {
		return "check"
	}
	return "onsync"
}

type interrupt struct {
	C chan struct{}
}

func handleKillSig(handler func(), logger *log.Logger) interrupt {
	i := interrupt{
		C: make(chan struct{}),
	}

	sigChannel := make(chan os.Signal, 1)

	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		for signal := range sigChannel {
			logger.Info().Msgf("Receive signal %s, Shutting down...", signal)
			handler()
			close(i.C)
		}
	}()
	return i
}

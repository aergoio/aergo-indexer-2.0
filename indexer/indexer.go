package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/aergoio/aergo-lib/log"
)

// Indexer hold all state information
type Indexer struct {
	// config
	log                *log.Logger
	dbAddr             string
	serverAddr         string
	prefix             string
	runMode            string
	networkTypeForCccv string
	indexNamePrefix    string
	aliasNamePrefix    string
	lastHeight         uint64
	cccvNftAddress     []byte
	bulkSize           int32
	batchTime          time.Duration
	minerNum           int
	grpcNum            int
	whitelistAddresses []string
	tokenVerifyAddr    []byte
	contractVerifyAddr []byte

	db         db.DbController
	grpcClient *client.AergoClientController
	stream     types.AergoRPCService_ListBlockStreamClient
	bulk       *Bulk
	cache      *Cache
}

// NewIndexer creates new Indexer instance
func NewIndexer(options ...IndexerOptionFunc) (*Indexer, error) {
	var err error
	ctx := context.Background()

	// set default options
	svc := &Indexer{
		log:       log.NewLogger(""),
		bulkSize:  4000,
		batchTime: 60 * time.Second,
		minerNum:  32,
		grpcNum:   16,
	}

	// overwrite options on it
	for _, option := range options {
		if err = option(svc); err != nil {
			return nil, err
		}
	}

	// connect server
	svc.log.Info().Str("serverAddr", svc.serverAddr).Msg("Attempting to connect to the Aergo server")
	svc.grpcClient = svc.WaitForServer(ctx)
	svc.log.Info().Str("serverAddr", svc.serverAddr).Msg("Successfully connected to the Aergo server")

	// connect db
	svc.log.Info().Str("dbURL", svc.dbAddr).Msg("Attempting to connect to the database")
	svc.db, err = svc.WaitForDatabase(ctx)
	if err != nil {
		return nil, err
	}
	svc.log.Info().Str("dbURL", svc.dbAddr).Msg("Successfully connected to the database")

	// set bulk, cache
	svc.bulk = NewBulk(svc)
	svc.cache = NewCache(svc)

	return svc, nil
}

// Start setups the indexer
func (ns *Indexer) Start(startFrom uint64, stopAt uint64) (exitOnComplete bool) {
	ns.log.Info().Msg("Start Indexer")

	if err := ns.InitIndex(); err != nil {
		ns.log.Error().Err(err).Msg("Index check failed. Chain info is not valid. please check aergo server info or reset")
		return true
	}

	ns.initCccvNft()
	ns.lastHeight = uint64(ns.GetBestBlock()) - 1

	switch ns.runMode {
	case "all":
		ns.OnSync()
		ns.Check(startFrom, stopAt)
		return false
	case "check":
		ns.Check(startFrom, stopAt)
		return true
	case "onsync":
		ns.OnSync()
		return false
	default:
		ns.log.Warn().Str("mode", ns.runMode).Msg("Invalid run mode")
		return true
	}
}

// Stops the indexer
func (ns *Indexer) Stop() {
	if ns.stream != nil {
		ns.stream.CloseSend()
		ns.stream = nil
	}

	ns.log.Info().Msg("Stop Indexer")
}

func (ns *Indexer) WaitForServer(ctx context.Context) *client.AergoClientController {
	var err error
	var aergoClient *client.AergoClientController
	for {
		aergoClient, err = client.NewAergoClient(ns.serverAddr, ctx)
		if err == nil && aergoClient != nil {
			break
		}
		ns.log.Info().Str("serverAddr", ns.serverAddr).Err(err).Msg("Could not connect to aergo server, retrying")
		time.Sleep(time.Second)
	}
	return aergoClient
}

func (ns *Indexer) WaitForDatabase(ctx context.Context) (*db.ElasticsearchDbController, error) {
	dbController, err := db.NewElasticsearchDbController(ctx, ns.dbAddr)
	if err != nil {
		return nil, err
	}
	// wait until elasticsearch client is ready
	for {
		if ok := dbController.HealthCheck(ctx); ok == true {
			break
		}
		ns.log.Info().Str("serverAddr", ns.serverAddr).Err(err).Msg("Could not connect to es database, retrying")
		time.Sleep(time.Second)
	}
	return dbController, nil
}

func (ns *Indexer) InitIndex() error {
	// init index prefix
	ns.aliasNamePrefix = fmt.Sprintf("%s_", ns.prefix)
	ns.indexNamePrefix = fmt.Sprintf("%s%s_", ns.aliasNamePrefix, time.Now().UTC().Format("2006-01-02_15-04-05"))

	// create index
	for {
		err := ns.CreateIndexIfNotExists("chain_info")
		if err == nil {
			break
		}
		ns.log.Info().Str("serverAddr", ns.serverAddr).Err(err).Msg("Could not create index, retrying...")
		time.Sleep(time.Second)
	}

	// check chain info
	err := ns.ValidChainInfo()
	if err != nil {
		ns.log.Error().Err(err).Msg("Chain info is not valid. please check aergo server info or reset")
		return err
	}

	// create other indexes
	ns.CreateIndexIfNotExists("block")
	ns.CreateIndexIfNotExists("tx")
	ns.CreateIndexIfNotExists("name")
	ns.CreateIndexIfNotExists("event")
	ns.CreateIndexIfNotExists("token")
	ns.CreateIndexIfNotExists("token_verified")
	ns.CreateIndexIfNotExists("contract")
	ns.CreateIndexIfNotExists("token_transfer")
	ns.CreateIndexIfNotExists("account_tokens")
	ns.CreateIndexIfNotExists("nft")
	ns.CreateIndexIfNotExists("account_balance")

	return nil
}

// GetBestBlock retrieves the current best block from the aergo client
func (ns *Indexer) GetBestBlock() uint64 {
	blockNo, err := ns.grpcClient.GetBestBlock()
	if err != nil {
		ns.log.Warn().Err(err).Msg("Failed to query node's block height")
		return 0
	} else {
		return blockNo
	}
}

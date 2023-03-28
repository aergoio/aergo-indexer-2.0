package indexer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/aergoio/aergo-lib/log"
)

// Indexer hold all state information
type Indexer struct {
	db         db.DbController
	grpcClient *client.AergoClientController

	stream             types.AergoRPCService_ListBlockStreamClient
	MChannel           chan BlockInfo
	BChannel           ChanInfoType
	RChannel           []chan BlockInfo
	SynDone            chan bool
	accToken           sync.Map
	peerId             sync.Map
	whiteListAddresses sync.Map
	indexNamePrefix    string
	aliasNamePrefix    string
	lastHeight         uint64
	cccvNftAddress     []byte

	// for bulk
	bulkSize  int32
	batchTime time.Duration
	minerNum  int
	grpcNum   int

	// config by user
	log                *log.Logger
	dbAddr             string
	serverAddr         string
	prefix             string
	runMode            string
	checkMode          bool
	cleanMode          bool
	NetworkTypeForCccv string
}

// NewIndexer creates new Indexer instance
func NewIndexer(options ...IndexerOptionFunc) (*Indexer, error) {
	var err error

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

	// conn server, db
	svc.grpcClient = svc.WaitForClient(svc.serverAddr)
	svc.db, err = db.NewElasticsearchDbController(svc.dbAddr)
	if err != nil {
		return nil, err
	}
	svc.log.Info().Str("dbURL", svc.dbAddr).Msg("Initialized database connection")

	// init index prefix, cccv
	svc.initIndexPrefix()

	return svc, nil
}

// Start setups the indexer
func (ns *Indexer) Start(startFrom uint64, stopAt uint64) (exitOnComplete bool) {
	ns.CreateIndexIfNotExists("block")
	ns.CreateIndexIfNotExists("tx")
	ns.CreateIndexIfNotExists("name")
	ns.CreateIndexIfNotExists("token")
	ns.CreateIndexIfNotExists("contract")
	ns.CreateIndexIfNotExists("token_transfer")
	ns.CreateIndexIfNotExists("account_tokens")
	ns.CreateIndexIfNotExists("nft")
	ns.CreateIndexIfNotExists("account_balance")

	ns.initCccvNft()
	ns.lastHeight = uint64(ns.GetBestBlockFromClient()) - 1

	switch ns.runMode {
	case "check":
		ns.RunCheckIndex(startFrom, stopAt)
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

	os.Exit(0)
}

func (ns *Indexer) WaitForClient(serverAddr string) *client.AergoClientController {
	var err error
	ctx := context.Background()
	var aergoClient *client.AergoClientController
	for {
		aergoClient, err = client.NewAergoClient(serverAddr, ctx)
		if err == nil && aergoClient != nil {
			break
		}
		ns.log.Info().Str("serverAddr", serverAddr).Err(err).Msg("Could not connect to aergo server, retrying")
		time.Sleep(time.Second)
	}
	ns.log.Info().Str("serverAddr", serverAddr).Msg("Connected to aergo server")

	return aergoClient
}

func (ns *Indexer) initIndexPrefix() {
	ns.aliasNamePrefix = fmt.Sprintf("%s_", ns.prefix)
	ns.indexNamePrefix = fmt.Sprintf("%s%s_", ns.aliasNamePrefix, time.Now().UTC().Format("2006-01-02_15-04-05"))
}

// CreateIndexIfNotExists creates the indices and aliases in ES
func (ns *Indexer) CreateIndexIfNotExists(documentType string) {
	aliasName := ns.aliasNamePrefix + documentType

	// Check for existing index to find out current indexNamePrefix
	exists, indexNamePrefix, err := ns.db.GetExistingIndexPrefix(aliasName, documentType)
	if err != nil {
		ns.log.Error().Err(err).Msg("Error when checking for alias")
	}

	if exists {
		ns.log.Info().Str("aliasName", aliasName).Str("indexNamePrefix", indexNamePrefix).Msg("Alias found")
		ns.indexNamePrefix = indexNamePrefix
		return
	}

	// Create new index
	indexName := ns.indexNamePrefix + documentType
	err = ns.db.CreateIndex(indexName, documentType)
	if err != nil {
		ns.log.Error().Err(err).Str("indexName", indexName).Msg("Error when creating index")
	} else {
		ns.log.Info().Str("indexName", indexName).Msg("Created index")
	}

	// Update alias, only when initializing and not reindexing
	err = ns.db.UpdateAlias(aliasName, indexName)
	if err != nil {
		ns.log.Error().Err(err).Str("aliasName", aliasName).Str("indexName", indexName).Msg("Error when updating alias")
	} else {
		ns.log.Info().Str("aliasName", aliasName).Str("indexName", indexName).Msg("Updated alias")
	}
	return
}

// UpdateAliasForType updates aliases
func (ns *Indexer) UpdateAliasForType(documentType string) {
	aliasName := ns.aliasNamePrefix + documentType
	indexName := ns.indexNamePrefix + documentType
	err := ns.db.UpdateAlias(aliasName, indexName)
	if err != nil {
		ns.log.Warn().Err(err).Str("aliasName", aliasName).Str("indexName", indexName).Msg("Error when updating alias")
	} else {
		ns.log.Info().Err(err).Str("aliasName", aliasName).Str("indexName", indexName).Msg("Updated alias")
	}
}

// GetBestBlockFromClient retrieves the current best block from the aergo client
func (ns *Indexer) GetBestBlockFromClient() uint64 {
	blockNo, err := ns.grpcClient.GetBestBlock()
	if err != nil {
		ns.log.Warn().Err(err).Msg("Failed to query node's block height")
		return 0
	} else {
		return blockNo
	}
}

// GetBestBlockFromDb retrieves the current best block from the db
func (ns *Indexer) GetBestBlockFromDb() (uint64, error) {
	block, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "block",
		SortField: "no",
		SortAsc:   false,
	}, func() doc.DocType {
		block := new(doc.EsBlock)
		block.BaseEsType = new(doc.BaseEsType)
		return block
	})
	if err != nil {
		return 0, err
	}
	if block == nil {
		return 0, errors.New("best block not found")
	}
	return block.(*doc.EsBlock).BlockNo, nil
}

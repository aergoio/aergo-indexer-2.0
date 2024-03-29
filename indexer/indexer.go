package indexer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/aergoio/aergo-lib/log"
)

// Indexer hold all state information
type Indexer struct {
	// config
	log                     *log.Logger
	dbAddr                  string
	serverAddr              string
	prefix                  string
	runMode                 string
	fix                     bool
	networkTypeForCccv      string
	indexNamePrefix         string
	aliasNamePrefix         string
	lastHeight              uint64
	cccvNftAddress          []byte
	bulkSize                int32
	batchTime               time.Duration
	minerNum                int
	grpcNum                 int
	tokenVerifyAddr         []byte
	contractVerifyAddr      []byte
	balanceWhitelist        []string
	tokenVerifyWhitelist    []string
	contractVerifyWhitelist []string

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
	ns.CreateIndexIfNotExists("contract")
	ns.CreateIndexIfNotExists("token_transfer")
	ns.CreateIndexIfNotExists("account_tokens")
	ns.CreateIndexIfNotExists("nft")
	ns.CreateIndexIfNotExists("account_balance")
	ns.CreateIndexIfNotExists("whitelist")

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

func (ns *Indexer) ValidChainInfo() error {
	chainInfoFromNode, err := ns.grpcClient.GetChainInfo() // get chain info from node
	if err != nil {
		return err
	}

	document, err := ns.db.SelectOne(db.QueryParams{ // get chain info from db
		IndexName: ns.indexNamePrefix + "chain_info",
		SortField: "version",
		SortAsc:   true,
		From:      0,
	}, func() doc.DocType {
		chainInfo := new(doc.EsChainInfo)
		chainInfo.BaseEsType = new(doc.BaseEsType)
		return chainInfo
	})
	if err != nil {
		ns.log.Info().Err(err).Msg("Could not query chain info, add new one.")
	}
	if document == nil { // if empty in db, put new chain info
		chainInfo := doc.EsChainInfo{
			BaseEsType: &doc.BaseEsType{
				Id: chainInfoFromNode.Id.Magic,
			},
			Mainnet:   chainInfoFromNode.Id.Mainnet,
			Public:    chainInfoFromNode.Id.Public,
			Consensus: chainInfoFromNode.Id.Consensus,
			Version:   uint64(chainInfoFromNode.Id.Version),
		}
		err = ns.db.Insert(&chainInfo, ns.indexNamePrefix+"chain_info")
		if err != nil {
			return err
		}
	} else {
		chainInfoFromDb := document.(*doc.EsChainInfo)
		if chainInfoFromDb.Id != chainInfoFromNode.Id.Magic ||
			chainInfoFromDb.Consensus != chainInfoFromNode.Id.Consensus ||
			chainInfoFromDb.Public != chainInfoFromNode.Id.Public ||
			chainInfoFromDb.Mainnet != chainInfoFromNode.Id.Mainnet ||
			chainInfoFromDb.Version != uint64(chainInfoFromNode.Id.Version) { // valid chain info
			return errors.New("chain info is not matched")
		}
	}
	return nil
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

// CreateIndexIfNotExists creates the indices and aliases in ES
func (ns *Indexer) CreateIndexIfNotExists(documentType string) error {
	aliasName := ns.aliasNamePrefix + documentType

	// Check for existing index to find out current indexNamePrefix
	exists, indexNamePrefix, err := ns.db.GetExistingIndexPrefix(aliasName, documentType)
	if err != nil {
		ns.log.Error().Err(err).Msg("Error when checking for alias")
		return err
	}

	if exists {
		ns.log.Info().Str("aliasName", aliasName).Str("indexNamePrefix", indexNamePrefix).Msg("Alias found")
		ns.indexNamePrefix = indexNamePrefix
		return nil
	}

	// Create new index
	indexName := ns.indexNamePrefix + documentType
	err = ns.db.CreateIndex(indexName, documentType)
	if err != nil {
		ns.log.Error().Err(err).Str("indexName", indexName).Msg("Error when creating index")
		return err
	} else {
		ns.log.Info().Str("indexName", indexName).Msg("Created index")
	}

	// Update alias
	err = ns.db.UpdateAlias(aliasName, indexName)
	if err != nil {
		ns.log.Error().Err(err).Str("aliasName", aliasName).Str("indexName", indexName).Msg("Error when updating alias")
		return err
	} else {
		ns.log.Info().Str("aliasName", aliasName).Str("indexName", indexName).Msg("Updated alias")
	}
	return nil
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

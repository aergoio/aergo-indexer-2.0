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

	stream   types.AergoRPCService_ListBlockStreamClient
	MChannel chan BlockInfo
	BChannel ChanInfoType
	RChannel []chan BlockInfo
	SynDone  chan bool
	accToken sync.Map
	peerId   sync.Map

	// config
	log                    *log.Logger
	prefix                 string
	networkType            string
	runMode                string
	aliasNamePrefix        string
	indexNamePrefix        string
	lastBlockHeight        uint64
	startHeight            uint64
	bulkSize               int32
	batchTime              time.Duration
	minerNum               int
	dbAddr                 string
	serverAddr             string
	grpcNum                int
	whiteListAddresses     sync.Map
	whiteListBlockInterval uint64
}

// NewIndexer creates new Indexer instance
func NewIndexer(options ...IndexerOptionFunc) (*Indexer, error) {
	var err error

	// set default options
	svc := &Indexer{
		log:                log.NewLogger(""),
		networkType:        "",
		aliasNamePrefix:    "",
		indexNamePrefix:    "",
		lastBlockHeight:    0,
		startHeight:        0,
		bulkSize:           0,
		batchTime:          0,
		minerNum:           0,
		dbAddr:             "",
		serverAddr:         "",
		grpcNum:            0,
		whiteListAddresses: sync.Map{},
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
	var err error
	switch ns.runMode {
	case "check":
		err = ns.RunCheckIndex(startFrom, stopAt)
		if err != nil {
			ns.log.Warn().Err(err).Msg("Check failed")
		}
		return true
	case "clean":
		err = ns.RunCleanIndex()
		if err != nil {
			ns.log.Warn().Err(err).Msg("Clean failed")
		}
		return true
	case "onsync":
		err = ns.OnSync()
		if err != nil {
			ns.log.Warn().Err(err).Msg("Could not start indexer")
			return true
		}
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
	if ns.prefix == "" {
		ns.prefix = ns.networkType
	}
	ns.aliasNamePrefix = fmt.Sprintf("%s_", ns.prefix)
	ns.indexNamePrefix = fmt.Sprintf("%s%s_", ns.aliasNamePrefix, time.Now().UTC().Format("2006-01-02_15-04-05"))
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

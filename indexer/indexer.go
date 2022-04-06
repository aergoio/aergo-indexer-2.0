package indexer

import (
	"context"
	"time"
	"fmt"
	"os"
	"errors"
	"github.com/kjunblk/aergo-indexer/indexer/db"
	"github.com/kjunblk/aergo-indexer/types"

	doc "github.com/kjunblk/aergo-indexer/indexer/documents"
	"github.com/aergoio/aergo-lib/log"
)

type ChanInfo struct {
	Type    uint			// 0:stop_bulk, 1:add, 2:commit
        Doc doc.DocType
}

type ChanType struct {
        Block   chan ChanInfo
        Tx      chan ChanInfo
        Name    chan ChanInfo
        Token   chan ChanInfo
        TokenTx chan ChanInfo
        AccTokens chan ChanInfo
        NFT 	chan ChanInfo
}

type BlockInfo struct {
	Type    uint			
        Height  uint64
}

// Indexer hold all state information
type Indexer struct {
	db              *db.ElasticsearchDbController
	grpcClient      types.AergoRPCServiceClient
	aliasNamePrefix string
	indexNamePrefix string
	lastBlockHeight uint64
	log             *log.Logger
	stream          types.AergoRPCService_ListBlockStreamClient
	MChannel	chan BlockInfo
	BChannel	ChanType
	RChannel	[]chan BlockInfo
        SynDone		chan bool
	StartHeight	uint64
	BulkSize	int32
	BatchTime	time.Duration
	MinerNum	int
}

// NewIndexer creates new Indexer instance
func NewIndexer(grpcClient types.AergoRPCServiceClient, logger *log.Logger, dbURL string, namePrefix string) (*Indexer, error) {
	aliasNamePrefix := namePrefix
	var err error

	dbController, err := db.NewElasticsearchDbController(dbURL)
	if err != nil {
		return nil, err
	}

	logger.Info().Str("dbURL", dbURL).Msg("Initialized database connection")
	svc := &Indexer{
		db:              dbController,
		aliasNamePrefix: aliasNamePrefix,
		indexNamePrefix: generateIndexPrefix(aliasNamePrefix),
		grpcClient:	 grpcClient,
		lastBlockHeight: 0,
		log:             logger,
	}

	return svc, nil
}

func generateIndexPrefix(aliasNamePrefix string) string {
	return fmt.Sprintf("%s%s_", aliasNamePrefix, time.Now().UTC().Format("2006-01-02_15-04-05"))
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

// GetNodeBlockHeight updates state from db
func (ns *Indexer) GetNodeBlockHeight() uint64 {
	blockchain, err := ns.grpcClient.Blockchain(context.Background(), &types.Empty{})

	if err != nil {
		ns.log.Warn().Err(err).Msg("Failed to query node's block height")
		return 0
	} else {
		return blockchain.BestHeight
	}
}

// Stop stops the indexer
func (ns *Indexer) Stop() {

        if ns.stream != nil {
               ns.stream.CloseSend()
               ns.stream = nil
        }

	ns.log.Info().Msg("Stop Indexer")

	os.Exit(0)
}

// GetBestBlockFromDb retrieves the current best block from the db
func (ns *Indexer) GetBestBlockFromDb() (*doc.EsBlock, error) {
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
                return nil, err
        }
        if block == nil {
                return nil, errors.New("best block not found")
        }

        return block.(*doc.EsBlock), nil
}


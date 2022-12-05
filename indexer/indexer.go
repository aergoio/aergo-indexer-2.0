package indexer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/aergoio/aergo-lib/log"
	"google.golang.org/grpc"
)

type ChanInfo struct {
	Type uint // 0:stop_bulk, 1:add, 2:commit
	Doc  doc.DocType
}

type ChanType struct {
	Block     chan ChanInfo
	Tx        chan ChanInfo
	Name      chan ChanInfo
	Token     chan ChanInfo
	TokenTx   chan ChanInfo
	AccTokens chan ChanInfo
	NFT       chan ChanInfo
}

type BlockInfo struct {
	Type   uint // 0:stop_miner, 1:bulk, 2:sync
	Height uint64
}

// Indexer hold all state information
type Indexer struct {
	db         *db.ElasticsearchDbController
	grpcClient types.AergoRPCServiceClient

	stream   types.AergoRPCService_ListBlockStreamClient
	MChannel chan BlockInfo
	BChannel ChanType
	RChannel []chan BlockInfo
	SynDone  chan bool
	accToken map[string]bool

	// config
	log             *log.Logger
	aliasNamePrefix string
	indexNamePrefix string
	lastBlockHeight uint64
	startHeight     uint64
	bulkSize        int32
	batchTime       time.Duration
	minerNum        int
	dbAddr          string
	serverAddr      string
	grpcNum         int
}

// NewIndexer creates new Indexer instance
func NewIndexer(options ...IndexerOptionFunc) (*Indexer, error) {
	var err error

	// set default options
	svc := &Indexer{
		log:             log.NewLogger(""),
		aliasNamePrefix: "",
		indexNamePrefix: generateIndexPrefix(""),
		lastBlockHeight: 0,
		startHeight:     0,
		bulkSize:        0,
		batchTime:       0,
		minerNum:        0,
		dbAddr:          "",
		serverAddr:      "",
		grpcNum:         0,
		accToken:        make(map[string]bool),
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

	return svc, nil
}

// Generate aliases of index name
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

func (ns *Indexer) WaitForClient(serverAddr string) types.AergoRPCServiceClient {
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

		ns.log.Info().Str("serverAddr", serverAddr).Err(err).Msg("Could not connect to aergo server, retrying")
		time.Sleep(time.Second)
	}

	ns.log.Info().Str("serverAddr", serverAddr).Msg("Connected to aergo server")

	return types.NewAergoRPCServiceClient(conn)
}

// Start setups the indexer
func (ns *Indexer) Start(runMode string, startFrom uint64, stopAt uint64) (exitOnComplete bool) {
	var err error
	switch runMode {
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
		err = ns.OnSync(startFrom, stopAt)
		if err != nil {
			ns.log.Warn().Err(err).Msg("Could not start indexer")
			return true
		}
		return false
	default:
		ns.log.Warn().Str("mode", runMode).Msg("Invalid run mode")
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

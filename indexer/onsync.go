package indexer

import (
	"fmt"
	"io"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	"github.com/aergoio/aergo-indexer-2.0/types"
)

// Start setups the indexer
func (ns *Indexer) OnSync() error {
	ns.CreateIndexIfNotExists("block")
	ns.CreateIndexIfNotExists("tx")
	ns.CreateIndexIfNotExists("name")
	ns.CreateIndexIfNotExists("token")
	ns.CreateIndexIfNotExists("contract")
	ns.CreateIndexIfNotExists("token_transfer")
	ns.CreateIndexIfNotExists("account_tokens")
	ns.CreateIndexIfNotExists("nft")
	ns.CreateIndexIfNotExists("account_balance")

	ns.init_cccv_nft()

	ns.lastBlockHeight = uint64(ns.GetBestBlockFromClient()) - 1

	BestBlockNo, err := ns.GetBestBlockFromDb()
	if err == nil {
		bulk_size := int(ns.lastBlockHeight - BestBlockNo)
		switch {
		case bulk_size <= 100:
			// small size -> direct insert
			ns.lastBlockHeight = BestBlockNo
		case 100 < bulk_size && bulk_size < 10000:
			// middle size -> bulk insert
			go func() {
				ns.StartBulkChannel()
				ns.InsertBlocksInRange(BestBlockNo, ns.lastBlockHeight)
				ns.StopBulkChannel()
			}()
		default:
			// big size -> External bulk insert
			fmt.Println("PLEASE RUN --check true")
		}
	} else {
		fmt.Println("PLEASE RUN --check true")
	}

	// Get ready to start
	ns.log.Info().Uint64("OnSync block height", ns.lastBlockHeight+1).Msg("Started Indexer")

	// Sync stream
	go ns.StartStream()
	return nil
}

func (ns *Indexer) SleepIndexer(BlockNo uint64) {
	fmt.Println("<------  SLEEP  ------> ", BlockNo)

	return_tag := false

	go func() {
		for {
			switch return_tag {
			case true:
				return
			default:
				_, _ = ns.stream.Recv()
			}
		}
	}()

	CBlockNo := BlockNo

	for {
		time.Sleep(5 * time.Second)

		BestBlockNo, err := ns.GetBestBlockFromDb()
		if err == nil {
			if CBlockNo >= BestBlockNo {
				ns.lastBlockHeight = BestBlockNo
				fmt.Println("<------ WAKE UP ------> ", ns.lastBlockHeight)
				return_tag = true
				return
			} else {
				CBlockNo = BestBlockNo
			}
		}
		fmt.Println("X CB : ", CBlockNo)
	}
}

// StartStream starts the block stream and calls SyncBlock
func (ns *Indexer) StartStream() {
	// SyncBlock indexes new block after checking for skipped blocks and reorgs
	MChannel := make(chan BlockInfo)

	go ns.Miner(MChannel, ns.grpcClient)

	SyncBlock := func(block *types.Block) error {
		newHeight := block.Header.BlockNo
		fmt.Println("->")
		if newHeight < ns.lastBlockHeight { // Rewound 1 or more blocks
			// This needs to be syncronous, otherwise it may
			// delete the block we are just about to add
			ns.DeleteBlocksInRange(newHeight+1, ns.lastBlockHeight)
			ns.lastBlockHeight = newHeight
			return nil
		}

		// indexing
		if newHeight > ns.lastBlockHeight+1 {
			for H := ns.lastBlockHeight + 1; H < newHeight; H++ {
				MChannel <- BlockInfo{2, H}
				fmt.Println("O NB :", H)
			}
		}

		if time.Now().UnixNano()%10 == 0 {
			time.Sleep(1 * time.Second)
			BestBlockNo, err := ns.GetBestBlockFromDb()
			if err == nil && BestBlockNo >= newHeight {
				ns.SleepIndexer(newHeight)
			} else {
				MChannel <- BlockInfo{2, newHeight}
				ns.lastBlockHeight = newHeight
				fmt.Println("O NB :", newHeight)
			}
		} else {
			MChannel <- BlockInfo{2, newHeight}
			ns.lastBlockHeight = newHeight
			fmt.Println("O NB :", newHeight)
		}
		return nil
	}

	for {
		ns.openBlockStream()
		for {
			block, err := ns.stream.Recv()
			if err == io.EOF {
				ns.log.Warn().Msg("Stream ended")
				break
			}
			if err != nil {
				ns.log.Warn().Err(err).Msg("Failed to receive a block")
				break
			}
			SyncBlock(block)
		}

		if ns.stream != nil {
			ns.stream.CloseSend()
			ns.stream = nil
		}
	}
}

func (ns *Indexer) openBlockStream() {
	var err error
	for {
		ns.stream, err = ns.grpcClient.ListBlockStream()
		if err != nil || ns.stream == nil {
			ns.log.Info().Msg("Waiting open stream in 6 seconds")
			time.Sleep(6 * time.Second)
		} else {
			ns.log.Info().Msg("Starting stream ....")
			return
		}
	}
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

func (ns *Indexer) deleteTypeByQuery(typeName string, rangeQuery db.IntegerRangeQuery) {
	deleted, err := ns.db.Delete(db.QueryParams{
		IndexName:    ns.indexNamePrefix + typeName,
		IntegerRange: &rangeQuery,
	})
	if err != nil {
		ns.log.Warn().Err(err).Str("typeName", typeName).Msg("Failed to delete documents")
	} else {
		ns.log.Info().Uint64("deleted", deleted).Str("typeName", typeName).Msg("Deleted documents")
	}
}

// DeleteBlocksInRange deletes previously synced blocks and their txs and names in the range of [fromBlockheight, toBlockHeight]
func (ns *Indexer) DeleteBlocksInRange(fromBlockHeight uint64, toBlockHeight uint64) {
	// node error check
	if (toBlockHeight - fromBlockHeight) > 1000 {
		ns.log.Warn().Msg("Full Node Error!")
		ns.Stop()
	}

	ns.log.Info().Msg(fmt.Sprintf("Rolling back %d blocks [%d..%d]", (1 + toBlockHeight - fromBlockHeight), fromBlockHeight, toBlockHeight))
	ns.deleteTypeByQuery("block", db.IntegerRangeQuery{Field: "no", Min: fromBlockHeight, Max: toBlockHeight})
	ns.deleteTypeByQuery("tx", db.IntegerRangeQuery{Field: "blockno", Min: fromBlockHeight, Max: toBlockHeight})
	ns.deleteTypeByQuery("name", db.IntegerRangeQuery{Field: "blockno", Min: fromBlockHeight, Max: toBlockHeight})
	ns.deleteTypeByQuery("token_transfer", db.IntegerRangeQuery{Field: "blockno", Min: fromBlockHeight, Max: toBlockHeight})
	ns.deleteTypeByQuery("token", db.IntegerRangeQuery{Field: "blockno", Min: fromBlockHeight, Max: toBlockHeight})
	ns.deleteTypeByQuery("nft", db.IntegerRangeQuery{Field: "blockno", Min: fromBlockHeight, Max: toBlockHeight})
}

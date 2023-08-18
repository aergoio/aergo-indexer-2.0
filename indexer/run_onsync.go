package indexer

import (
	"fmt"
	"io"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	"github.com/aergoio/aergo-indexer-2.0/types"
)

// Start setups the indexer
func (ns *Indexer) OnSync() {
	// Get ready to start
	ns.log.Info().Uint64("height", ns.lastHeight+1).Msg("Start Onsync...")

	ns.cache.registerVariables()
	// Sync stream
	go ns.startStream()
}

// startStream starts the block stream and calls SyncBlock
func (ns *Indexer) startStream() {
	// SyncBlock indexes new block after checking for skipped blocks and reorgs
	MChannel := make(chan BlockInfo)

	go ns.Miner(MChannel, ns.grpcClient)

	SyncBlock := func(block *types.Block) error {
		newHeight := block.Header.BlockNo
		if newHeight < ns.lastHeight { // Rewound 1 or more blocks
			// This needs to be syncronous, otherwise it may
			// delete the block we are just about to add
			ns.DeleteBlocksInRange(newHeight+1, ns.lastHeight)
			ns.lastHeight = newHeight
			return nil
		}

		// indexing
		if newHeight > ns.lastHeight+1 {
			for H := ns.lastHeight + 1; H < newHeight; H++ {
				MChannel <- BlockInfo{BlockType_Sync, H}
				fmt.Println(">>> New Block :", H)
			}
		}

		if time.Now().UnixNano()%10 == 0 {
			time.Sleep(1 * time.Second)
			BestBlockNo, err := ns.GetBestBlockFromDb()
			if err == nil && BestBlockNo >= newHeight {
				ns.sleepStream(newHeight)
			} else {
				MChannel <- BlockInfo{BlockType_Sync, newHeight}
				ns.lastHeight = newHeight
				fmt.Println(">>> New Block :", newHeight)
			}
		} else {
			MChannel <- BlockInfo{BlockType_Sync, newHeight}
			ns.lastHeight = newHeight
			fmt.Println(">>> New Block :", newHeight)
		}
		return nil
	}

	for {
		ns.openStream()
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

func (ns *Indexer) openStream() {
	var err error
	for {
		ns.stream, err = ns.grpcClient.ListBlockStream()
		if err != nil || ns.stream == nil {
			ns.log.Info().Msg("Waiting open stream in 6 seconds")
			time.Sleep(6 * time.Second)
		} else {
			ns.log.Info().Msg("Starting stream...")
			return
		}
	}
}

func (ns *Indexer) sleepStream(BlockNo uint64) {
	ns.log.Info().Msgf("Sleep stream... %d", BlockNo)

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
				ns.lastHeight = BestBlockNo
				ns.log.Info().Msgf("Wake up stream %d", ns.lastHeight)
				return_tag = true
				return
			} else {
				CBlockNo = BestBlockNo
			}
		}
		fmt.Println(">>> Sleep Block : ", CBlockNo)
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

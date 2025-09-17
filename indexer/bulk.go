package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
)

type Bulk struct {
	idxer *Indexer

	BChannel ChanInfoType
	RChannel []chan BlockInfo
	commitSync *sync.WaitGroup

	bulkSize  int32
	batchTime time.Duration
	minerNum  int
	grpcNum   int
}

func NewBulk(idxer *Indexer) *Bulk {
	return &Bulk{
		idxer:     idxer,
		bulkSize:  idxer.bulkSize,
		batchTime: idxer.batchTime,
		minerNum:  idxer.minerNum,
		grpcNum:   idxer.grpcNum,
	}
}

func (b *Bulk) InsertBlocksInRange(fromBlockHeight uint64, toBlockHeight uint64) {
	b.idxer.log.Info().Msg(fmt.Sprintf("Indexing %d [%d..%d]", (1 + toBlockHeight - fromBlockHeight), fromBlockHeight, toBlockHeight))

	for blockHeight := toBlockHeight; blockHeight > fromBlockHeight; blockHeight-- {
		if blockHeight%10000 == 0 {
			b.idxer.log.Info().Uint64("Height", blockHeight).Msg("Current Reindex")
		}
		b.RChannel[blockHeight%uint64(b.minerNum)] <- BlockInfo{BlockType_Bulk, blockHeight}
	}
	// last one
	b.RChannel[0] <- BlockInfo{BlockType_Bulk, fromBlockHeight}
}

func (b *Bulk) StartBulkChannel() {
	// Open buffered channels for each indices to prevent commit starvation
	b.BChannel.Block = make(chan ChanInfo, 8192)
	b.BChannel.Tx = make(chan ChanInfo, 8192)
	b.BChannel.Event = make(chan ChanInfo, 8192)
	b.BChannel.Contract = make(chan ChanInfo, 8192)
	b.BChannel.TokenTransfer = make(chan ChanInfo, 8192)
	b.BChannel.AccTokens = make(chan ChanInfo, 8192)
	b.BChannel.InternalOps = make(chan ChanInfo, 8192)
	b.BChannel.ContractCall = make(chan ChanInfo, 8192)
	// Initialize WaitGroup for commit synchronization
	b.commitSync = &sync.WaitGroup{}

	// Start bulk indexers for each index
	go b.BulkIndexer(b.BChannel.Block, b.idxer.indexNamePrefix+"block", b.bulkSize, b.batchTime, true)
	go b.BulkIndexer(b.BChannel.Tx, b.idxer.indexNamePrefix+"tx", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.Event, b.idxer.indexNamePrefix+"event", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.Contract, b.idxer.indexNamePrefix+"contract", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.TokenTransfer, b.idxer.indexNamePrefix+"token_transfer", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.AccTokens, b.idxer.indexNamePrefix+"account_tokens", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.InternalOps, b.idxer.indexNamePrefix+"internal_operations", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.ContractCall, b.idxer.indexNamePrefix+"contract_call", b.bulkSize, b.batchTime, false)

	// Start multiple miners
	GrpcClients := make([]*client.AergoClientController, b.grpcNum)
	for i := 0; i < b.grpcNum; i++ {
		GrpcClients[i] = b.idxer.WaitForServer(context.Background())
	}

	b.RChannel = make([]chan BlockInfo, b.minerNum)
	for i := 0; i < b.minerNum; i++ {
		// Buffer RChannel to prevent miner blocking
		b.RChannel[i] = make(chan BlockInfo, 1024)
		if b.grpcNum > 0 {
			go b.idxer.Miner(b.RChannel[i], GrpcClients[i%b.grpcNum])
		} else {
			go b.idxer.Miner(b.RChannel[i], b.idxer.grpcClient)
		}
	}
}

func (b *Bulk) StopBulkChannel() {
	b.idxer.log.Debug().Msg("grpc channel stop")

	for i := 0; i < b.minerNum; i++ {
		b.RChannel[i] <- BlockInfo{BlockType_StopMiner, 0}
		close(b.RChannel[i])
	}

	// Force commit
	time.Sleep(5 * time.Second)
	b.BChannel.Block <- ChanInfo{ChanType_Commit, nil}
	time.Sleep(5 * time.Second)

	// Send stop messages to each bulk channels
	b.BChannel.Block <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.Tx <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.Event <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.Contract <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.TokenTransfer <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.AccTokens <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.InternalOps <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.ContractCall <- ChanInfo{ChanType_StopBulk, nil}

	// Close bulk channels
	close(b.BChannel.Block)
	close(b.BChannel.Tx)
	close(b.BChannel.Event)
	close(b.BChannel.Contract)
	close(b.BChannel.TokenTransfer)
	close(b.BChannel.AccTokens)
	close(b.BChannel.InternalOps)
	close(b.BChannel.ContractCall)

	b.idxer.log.Info().Msg("Stop Bulk Indexer")
}

func (b *Bulk) BulkIndexer(docChannel chan ChanInfo, indexName string, bulkSize int32, batchTime time.Duration, isBlock bool) {
	bulk := b.idxer.db.InsertBulk(indexName)
	total := int32(0)
	lastCommit := time.Now()

	// Block Channel : Persistent timeout ticker for commits
	var ticker *time.Ticker
	var tickerC <-chan time.Time
	if isBlock {
		ticker = time.NewTicker(batchTime)
		tickerC = ticker.C
		defer ticker.Stop()
	}

	// Commit the documents to the database
	commitBulk := func(sync bool) {
		b.idxer.log.Debug().Str("indexName", indexName).Bool("sync", sync).Bool("isBlock", isBlock).Int32("total", total).Msg("commitBulk called")
		// If there are no documents to commit
		if total == 0 {
			// Signal the main channel
			if sync && !isBlock {
				b.commitSync.Done()
			}
			// Return if there are no documents to commit
			return
		}

		// If committing blocks
		if isBlock {
			b.idxer.log.Debug().Str("indexName", indexName).Msg("Block indexer starting commit coordination")
			// Add the number of channels to the wait group
			b.commitSync.Add(7)

			// Signal other channels to commit
			b.BChannel.Tx <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.Event <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.Contract <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.TokenTransfer <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.AccTokens <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.InternalOps <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.ContractCall <- ChanInfo{ChanType_Commit, nil}
			b.idxer.log.Debug().Msg("Finished sending commit signals, waiting for commitSync")

			// Wait for all other channels to finish their commits
			b.commitSync.Wait()
			b.idxer.log.Debug().Str("indexName", indexName).Msg("All channels finished committing")
		}

		// Commit the documents to the database
		err := bulk.Commit()

		// Signal the main channel that the commit was done
		if sync && !isBlock {
			b.idxer.log.Debug().Str("indexName", indexName).Msg("Calling commitSync.Done() after successful commit")
			b.commitSync.Done()
		}

		if err != nil {
			b.idxer.log.Error().Err(err).Str("indexName", indexName)
			b.StopBulkChannel()
		}

		// Log the commit statistics
		dur := time.Since(lastCommit).Seconds()
		pps := int64(float64(total) / dur)
		b.idxer.log.Info().Str("Commit", indexName).Int32("total", total).Int64("perSecond", pps)

		// Reset the variables for the next commit
		lastCommit = time.Now()
		total = 0
	}

	// Unified processing loop for both block and non-block indexers.
	// For non-block indexers, tickerC is nil, so the ticker case never fires.
	for {
		select {
		case I, ok := <-docChannel:
			if !ok {
				b.idxer.log.Debug().Str("indexName", indexName).Msg("Channel closed, exiting")
				return
			}

			// stop
			if I.Type == ChanType_StopBulk {
				b.idxer.log.Debug().Str("indexName", indexName).Msg("Received StopBulk, exiting")
				return
			}

			// commit
			if I.Type == ChanType_Commit {
				b.idxer.log.Debug().Str("indexName", indexName).Msg("Received commit signal")
				commitBulk(true)
				continue
			}

			// commit if bulk size reached
			if total >= bulkSize {
				b.idxer.log.Debug().Str("indexName", indexName).Int32("total", total).Msg("Bulk size reached, committing")
				commitBulk(false)
			}
			total++
			if total%1000 == 0 {
				b.idxer.log.Debug().Str("indexName", indexName).Int32("total", total).Msg("Processed documents")
			}

			// Only Create Indexing
			bulk.Add(I.Doc)

		case <-tickerC:
			// Timeout-based commit for block indexer
			if total > 0 && time.Since(lastCommit) >= batchTime {
				b.idxer.log.Debug().Str("indexName", indexName).Int32("total", total).Dur("timeSinceLastCommit", time.Since(lastCommit)).Msg("Timeout reached, triggering commit")
				b.BChannel.Block <- ChanInfo{ChanType_Commit, nil}
			}
		}
	}
}

package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
)

type Bulk struct {
	idxer *Indexer

	BChannel ChanInfoType
	RChannel []chan BlockInfo
	SynDone  chan bool

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
		if blockHeight%100000 == 0 {
			b.idxer.log.Info().Uint64("Height", blockHeight).Msg("Current Reindex")
		}
		b.RChannel[blockHeight%uint64(b.minerNum)] <- BlockInfo{BlockType_Bulk, blockHeight}
	}
	// last one
	b.RChannel[0] <- BlockInfo{BlockType_Bulk, fromBlockHeight}
}

func (b *Bulk) StartBulkChannel() {
	// Open channels for each indices
	b.BChannel.Block = make(chan ChanInfo)
	b.BChannel.Tx = make(chan ChanInfo)
	b.BChannel.TokenTransfer = make(chan ChanInfo)
	b.BChannel.AccTokens = make(chan ChanInfo)
	b.SynDone = make(chan bool)

	// Start bulk indexers for each indices
	go b.BulkIndexer(b.BChannel.Block, b.idxer.indexNamePrefix+"block", b.bulkSize, b.batchTime, true)
	go b.BulkIndexer(b.BChannel.Tx, b.idxer.indexNamePrefix+"tx", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.TokenTransfer, b.idxer.indexNamePrefix+"token_transfer", b.bulkSize, b.batchTime, false)
	go b.BulkIndexer(b.BChannel.AccTokens, b.idxer.indexNamePrefix+"account_tokens", b.bulkSize, b.batchTime, false)

	// Start multiple miners
	GrpcClients := make([]*client.AergoClientController, b.grpcNum)
	for i := 0; i < b.grpcNum; i++ {
		GrpcClients[i] = b.idxer.WaitForServer(context.Background())
	}

	b.RChannel = make([]chan BlockInfo, b.minerNum)
	for i := 0; i < b.minerNum; i++ {
		b.idxer.log.Debug().Msg("grpc channel start")
		b.RChannel[i] = make(chan BlockInfo)
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
	b.BChannel.TokenTransfer <- ChanInfo{ChanType_StopBulk, nil}
	b.BChannel.AccTokens <- ChanInfo{ChanType_StopBulk, nil}

	// Close bulk channels
	close(b.BChannel.Block)
	close(b.BChannel.Tx)
	close(b.BChannel.TokenTransfer)
	close(b.BChannel.AccTokens)
	close(b.SynDone)

	b.idxer.log.Info().Msg("Stop Bulk Indexer")
}

func (b *Bulk) BulkIndexer(docChannel chan ChanInfo, indexName string, bulkSize int32, batchTime time.Duration, isBlock bool) {
	bulk := b.idxer.db.InsertBulk(indexName)
	total := int32(0)
	begin := time.Now()

	return_flag := false

	// Block Channel : Time-out Sync
	if isBlock {
		go func() {
			for {
				if return_flag {
					return
				} else {
					time.Sleep(batchTime)
					if total > 0 && time.Now().Sub(begin) > batchTime {
						b.BChannel.Block <- ChanInfo{ChanType_Commit, nil}
					}
				}
			}
		}()
	}

	// Do commit
	commitBulk := func(sync bool) {
		if total == 0 {
			if sync && !isBlock {
				b.SynDone <- true
			}
			return_flag = true
			return
		}

		// Block Channel : wait other channels
		if isBlock {
			b.BChannel.Tx <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.TokenTransfer <- ChanInfo{ChanType_Commit, nil}
			b.BChannel.AccTokens <- ChanInfo{ChanType_Commit, nil}

			for i := 0; i < 3; i++ {
				<-b.SynDone
			}
		}

		err := bulk.Commit()

		if sync && !isBlock {
			b.SynDone <- true
		}

		if err != nil {
			b.idxer.log.Error().Err(err).Str("indexName", indexName)
			b.StopBulkChannel()
		}

		dur := time.Since(begin).Seconds()
		pps := int64(float64(total) / dur)

		b.idxer.log.Info().Str("Commit", indexName).Int32("total", total).Int64("perSecond", pps)

		begin = time.Now()
		total = 0
		return_flag = true
	}

	for I := range docChannel {
		// stop
		if I.Type == ChanType_StopBulk {
			break
		}

		// commit
		if I.Type == ChanType_Commit {
			commitBulk(true)
			continue
		}

		// commit
		if total >= bulkSize {
			commitBulk(false)
		}
		total++

		// Only Create Indexing
		bulk.Add(I.Doc)
	}
}

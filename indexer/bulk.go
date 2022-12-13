package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/olivere/elastic/v7"
)

// FOR BULK INSERT
func (ns *Indexer) InsertBlocksInRange(fromBlockHeight uint64, toBlockHeight uint64) {
	ns.log.Info().Msg(fmt.Sprintf("Indexing %d [%d..%d]", (1 + toBlockHeight - fromBlockHeight), fromBlockHeight, toBlockHeight))

	for blockHeight := toBlockHeight; blockHeight > fromBlockHeight; blockHeight-- {
		if blockHeight%100000 == 0 {
			fmt.Println(">>>>> Current Reindex Height :", blockHeight)
		}
		ns.RChannel[blockHeight%uint64(ns.minerNum)] <- BlockInfo{BlockType_Bulk, blockHeight}
	}
	// last one
	ns.RChannel[0] <- BlockInfo{BlockType_Bulk, fromBlockHeight}
}

// Run Bulk Indexing
func (ns *Indexer) StartBulkChannel() {
	// Open channels for each indices
	ns.BChannel.Block = make(chan ChanInfo)
	ns.BChannel.Tx = make(chan ChanInfo)
	ns.BChannel.TokenTransfer = make(chan ChanInfo)
	ns.BChannel.AccTokens = make(chan ChanInfo)
	ns.BChannel.NFT = make(chan ChanInfo)
	ns.SynDone = make(chan bool)

	// Start bulk indexers for each indices
	go ns.BulkIndexer(ns.BChannel.Block, ns.indexNamePrefix+"block", ns.bulkSize, ns.batchTime, true)
	go ns.BulkIndexer(ns.BChannel.Tx, ns.indexNamePrefix+"tx", ns.bulkSize, ns.batchTime, false)
	go ns.BulkIndexer(ns.BChannel.TokenTransfer, ns.indexNamePrefix+"token_transfer", ns.bulkSize, ns.batchTime, false)
	go ns.BulkIndexer(ns.BChannel.AccTokens, ns.indexNamePrefix+"account_tokens", ns.bulkSize, ns.batchTime, false)
	go ns.BulkIndexer(ns.BChannel.NFT, ns.indexNamePrefix+"nft", ns.bulkSize, ns.batchTime, false)

	// Start multiple miners
	GrpcClients := make([]types.AergoRPCServiceClient, ns.grpcNum)
	for i := 0; i < ns.grpcNum; i++ {
		GrpcClients[i] = ns.WaitForClient(ns.serverAddr)
	}

	ns.RChannel = make([]chan BlockInfo, ns.minerNum)
	for i := 0; i < ns.minerNum; i++ {
		fmt.Println(":::::::::::::::::::::: Start Channels")
		ns.RChannel[i] = make(chan BlockInfo)
		if ns.grpcNum > 0 {
			go ns.Miner(ns.RChannel[i], GrpcClients[i%ns.grpcNum])
		} else {
			go ns.Miner(ns.RChannel[i], ns.grpcClient)
		}
	}
}

// Stop Bulk indexing
func (ns *Indexer) StopBulkChannel() {
	fmt.Println(":::::::::::::::::::::: STOP Channels")

	for i := 0; i < ns.minerNum; i++ {
		ns.RChannel[i] <- BlockInfo{BlockType_StopMiner, 0}
		close(ns.RChannel[i])
	}

	// Force commit
	time.Sleep(5 * time.Second)
	ns.BChannel.Block <- ChanInfo{ChanType_Commit, nil}
	time.Sleep(5 * time.Second)

	// Send stop messages to each bulk channels
	ns.BChannel.Block <- ChanInfo{ChanType_StopBulk, nil}
	ns.BChannel.Tx <- ChanInfo{ChanType_StopBulk, nil}
	//	ns.BChannel.Name <- ChanInfo{ChanType_StopBulk,nil}
	//	ns.BChannel.Token <- ChanInfo{ChanType_StopBulk,nil}
	ns.BChannel.TokenTransfer <- ChanInfo{ChanType_StopBulk, nil}
	ns.BChannel.AccTokens <- ChanInfo{ChanType_StopBulk, nil}
	ns.BChannel.NFT <- ChanInfo{ChanType_StopBulk, nil}

	// Close bulk channels
	close(ns.BChannel.Block)
	close(ns.BChannel.Tx)
	// close(ns.BChannel.Name)
	// close(ns.BChannel.Token)
	close(ns.BChannel.TokenTransfer)
	close(ns.BChannel.AccTokens)
	close(ns.BChannel.NFT)
	close(ns.SynDone)

	ns.log.Info().Msg("Stop Bulk Indexer")
}

// Do Bulk Indexing
func (ns *Indexer) BulkIndexer(docChannel chan ChanInfo, indexName string, bulkSize int32, batchTime time.Duration, isBlock bool) {
	ctx := context.Background()
	bulk := ns.db.Client.Bulk().Index(indexName)
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
						ns.BChannel.Block <- ChanInfo{ChanType_Commit, nil}
					}
				}
			}
		}()
	}

	// Do commit
	commitBulk := func(sync bool) {
		if total == 0 {
			if sync && !isBlock {
				ns.SynDone <- true
			}
			return_flag = true
			return
		}

		// Block Channel : wait other channels
		if isBlock {
			ns.BChannel.Tx <- ChanInfo{ChanType_Commit, nil}
			// ns.BChannel.Name <- ChanInfo{ChanType_Commit, nil}
			// ns.BChannel.Token <- ChanInfo{ChanType_Commit, nil}
			ns.BChannel.TokenTransfer <- ChanInfo{ChanType_Commit, nil}
			ns.BChannel.AccTokens <- ChanInfo{ChanType_Commit, nil}
			ns.BChannel.NFT <- ChanInfo{ChanType_Commit, nil}

			for i := 0; i < 4; i++ {
				<-ns.SynDone
			}
		}

		_, err := bulk.Do(ctx)

		if sync && !isBlock {
			ns.SynDone <- true
		}

		if err != nil {
			ns.log.Error().Err(err).Str("indexName", indexName)
			ns.StopBulkChannel()
		}

		dur := time.Since(begin).Seconds()
		pps := int64(float64(total) / dur)

		ns.log.Info().Str("Commit", indexName).Int32("total", total).Int64("perSecond", pps).Msg("")

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
		bulk.Add(elastic.NewBulkIndexRequest().OpType("create").Id(I.Doc.GetID()).Doc(I.Doc))
		// bulk.Add(elastic.NewBulkUpdateRequest().Id(I.Doc.GetID()).Doc(I.Doc).DocAsUpsert(true))
	}
}

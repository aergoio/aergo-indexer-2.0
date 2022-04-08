package indexer

import (
	"context"
	"time"
	"fmt"
//	"encoding/json"
	"github.com/olivere/elastic/v7"
)


// FOR BULK INSERT
func (ns *Indexer) InsertBlocksInRange(fromBlockHeight uint64, toBlockHeight uint64) {

	ns.log.Info().Msg(fmt.Sprintf("Indexing %d [%d..%d]", (1 + toBlockHeight - fromBlockHeight), fromBlockHeight, toBlockHeight))

	for blockHeight := toBlockHeight; blockHeight > fromBlockHeight; blockHeight-- {
		if (blockHeight % 100000 == 0) {
			fmt.Println(">>>>> Current Reindex Height :", blockHeight)
		}
		ns.RChannel[blockHeight%uint64(ns.MinerNum)]  <- BlockInfo{1,blockHeight}
	}
	// last one
	ns.RChannel[0]  <- BlockInfo{1,fromBlockHeight}
}


func (ns *Indexer) StartBulkChannel () {

	ns.BChannel.Block = make(chan ChanInfo)
	ns.BChannel.Tx = make(chan ChanInfo)
	ns.BChannel.Name = make(chan ChanInfo)
	ns.BChannel.Token = make(chan ChanInfo)
	ns.BChannel.TokenTx = make(chan ChanInfo)
	ns.BChannel.AccTokens = make(chan ChanInfo)
	ns.BChannel.NFT = make(chan ChanInfo)
	ns.SynDone = make(chan bool)

	go ns.BulkIndexer(ns.BChannel.Block, ns.indexNamePrefix+"block", ns.BulkSize, ns.BatchTime, true)
	go ns.BulkIndexer(ns.BChannel.Tx, ns.indexNamePrefix+"tx", ns.BulkSize, ns.BatchTime, false)
	go ns.BulkIndexer(ns.BChannel.Name, ns.indexNamePrefix+"name", ns.BulkSize, ns.BatchTime, false)
	go ns.BulkIndexer(ns.BChannel.Token, ns.indexNamePrefix+"token", ns.BulkSize, ns.BatchTime, false)
	go ns.BulkIndexer(ns.BChannel.TokenTx, ns.indexNamePrefix+"token_transfer", ns.BulkSize, ns.BatchTime, false)
	go ns.BulkIndexer(ns.BChannel.AccTokens, ns.indexNamePrefix+"account_tokens", ns.BulkSize, ns.BatchTime, false)
	go ns.BulkIndexer(ns.BChannel.NFT, ns.indexNamePrefix+"nft", ns.BulkSize, ns.BatchTime, false)

	ns.RChannel = make([]chan BlockInfo, ns.MinerNum)
	for i := 0 ; i < ns.MinerNum ; i ++ {

		fmt.Println(":::::::::::::::::::::: Start Channels")

		ns.RChannel[i] = make(chan BlockInfo)
		go ns.Miner(ns.RChannel[i])
	}
}

func (ns *Indexer) StopBulkChannel () {

	fmt.Println(":::::::::::::::::::::: STOP Channels")

	for i := 0 ; i < ns.MinerNum ; i ++ {
		ns.RChannel[i] <- BlockInfo{0,0}
		close(ns.RChannel[i])
	}

	// last commit 
	time.Sleep(5*time.Second)
	ns.BChannel.Block <- ChanInfo{2,nil}
	time.Sleep(5*time.Second)

	ns.BChannel.Block <- ChanInfo{0,nil}
	ns.BChannel.Tx <- ChanInfo{0,nil}
	ns.BChannel.Name <- ChanInfo{0,nil}
	ns.BChannel.Token <- ChanInfo{0,nil}
	ns.BChannel.TokenTx <- ChanInfo{0,nil}
	ns.BChannel.AccTokens <- ChanInfo{0,nil}
	ns.BChannel.NFT <- ChanInfo{0,nil}

	close(ns.BChannel.Block)
	close(ns.BChannel.Tx)
	close(ns.BChannel.Name)
	close(ns.BChannel.Token)
	close(ns.BChannel.TokenTx)
	close(ns.BChannel.AccTokens)
	close(ns.BChannel.NFT)
	close(ns.SynDone)

	ns.log.Info().Msg("Stop Bulk Indexer")
}


func (ns *Indexer) BulkIndexer(docChannel chan ChanInfo, indexName string, bulkSize int32, batchTime time.Duration, isBlock bool)  {

	ctx := context.Background()
	bulk := ns.db.Client.Bulk().Index(indexName)
	total := int32(0)
	begin := time.Now()

	return_flag := false

	if isBlock {
		go func() {
			for {
				switch  return_flag {
				case true :
					return
				default :

					time.Sleep(batchTime)

					if total > 0 && time.Now().Sub(begin) > batchTime {
//						fmt.Println(">>>>>>> Push blocks <<<<<", total)
						ns.BChannel.Block <- ChanInfo{2,nil}
					}

//					if return_flag { return }
				}
			}
		}()
	}

	commitBulk := func(sync bool) {

		if total == 0 {
			if (sync && !isBlock) {
				ns.SynDone <- true
			}

			return_flag = true

			return
		}

		// Block Channel : wait other Channels
		if isBlock {

			ns.BChannel.Tx		<- ChanInfo{2,nil}
			ns.BChannel.Token	<- ChanInfo{2,nil}
			ns.BChannel.TokenTx	<- ChanInfo{2,nil}
			ns.BChannel.Name	<- ChanInfo{2,nil}
			ns.BChannel.AccTokens	<- ChanInfo{2,nil}
			ns.BChannel.NFT		<- ChanInfo{2,nil}

			for i := 0 ; i < 6 ; i ++ {
				<-ns.SynDone
			}
		}

		_, err := bulk.Do(ctx)

		if sync && !isBlock  { ns.SynDone <- true }

		if err != nil {
			fmt.Println(">>>>>>> error Commit <<<<<", indexName, total, sync)
			ns.log.Error().Err(err).Str("indexName", indexName)
			ns.StopBulkChannel()
		}

		dur := time.Since(begin).Seconds()
		pps := int64(float64(total) / dur)

		ns.log.Info().Str("Commit",indexName).Int32("total", total).Int64("perSecond", pps).Msg("")
		begin = time.Now()

		total = 0

		return_flag = true
	}

	for I := range docChannel {

		// stop
		if I.Type == 0 {
			break
		}

		// commit
		if I.Type == 2 {
			commitBulk(true)
			continue
		}

		// commit
		if total >= bulkSize { commitBulk(false) }

		total ++

		// Only Create Indexing : for NFT
		bulk.Add(elastic.NewBulkIndexRequest().OpType("create").Id(I.Doc.GetID()).Doc(I.Doc))
//		bulk.Add(elastic.NewBulkUpdateRequest().Id(I.Doc.GetID()).Doc(I.Doc).DocAsUpsert(true))
	}
}


package indexer

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
)

type esBlockNo struct {
	*doc.BaseEsType
	BlockNo uint64 `json:"no" db:"no"`
}

// Start setups the indexer
func (ns *Indexer) RunCheckIndex(startFrom uint64, stopAt uint64) error {
	fmt.Println("=======> Start Check index ..")

	aliasName := ns.aliasNamePrefix + "block"
	for {
		_, _, err := ns.db.GetExistingIndexPrefix(aliasName, "block")

		if err == nil {
			break
		}
		time.Sleep(10 * time.Second)
	}

	ns.CreateIndexIfNotExists("block")
	ns.CreateIndexIfNotExists("tx")
	ns.CreateIndexIfNotExists("name")
	ns.CreateIndexIfNotExists("token")
	ns.CreateIndexIfNotExists("contract")
	ns.CreateIndexIfNotExists("token_transfer")
	ns.CreateIndexIfNotExists("account_tokens")
	ns.CreateIndexIfNotExists("nft")

	if ns.aliasNamePrefix == "mainnet_" {
		ns.cccv_nft_mainnet()
	} else if ns.aliasNamePrefix == "testnet_" {
		ns.cccv_nft_testnet()
	}

	if stopAt == 0 {
		ns.lastBlockHeight = uint64(ns.GetNodeBlockHeight()) - 1
	} else {
		ns.lastBlockHeight = stopAt
	}

	ns.fixIndex(startFrom, ns.lastBlockHeight)
	return nil
}

func (ns *Indexer) fixIndex(Start_Pos uint64, End_Pos uint64) {
	ns.log.Info().Uint64("startFrom", Start_Pos).Uint64("stopAt", End_Pos).Msg("Check Block range")
	ns.StartBulkChannel()

	os.WriteFile("./bulk_start_time.txt", []byte(time.Now().String()), 0644)

	var block doc.DocType
	var err error

	scroll := ns.db.Scroll(db.QueryParams{
		IndexName:    ns.indexNamePrefix + "block",
		TypeName:     "_doc",
		SelectFields: []string{"no"},
		Size:         10000,
		SortField:    "no",
		SortAsc:      false,
		From:         int(Start_Pos),
		To:           int(End_Pos),
	}, func() doc.DocType {
		block := new(esBlockNo)
		block.BaseEsType = new(doc.BaseEsType)

		return block
	})

	prevBlockNo := End_Pos + 1
	missingBlocks := uint64(0)
	blockNo := Start_Pos + 1
	for {
		block, err = scroll.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ns.log.Warn().Err(err).Msg("Failed to query block numbers")
			continue
		}
		blockNo = block.(*esBlockNo).BlockNo

		if blockNo%100000 == 0 {
			fmt.Println(">>> Check Block :", blockNo)
		}
		if blockNo >= prevBlockNo {
			continue
		}
		if blockNo <= Start_Pos {
			break
		}
		if blockNo < prevBlockNo-1 {
			missingBlocks = missingBlocks + (prevBlockNo - blockNo - 1)
			ns.InsertBlocksInRange(blockNo+1, prevBlockNo-1)
		}
		prevBlockNo = blockNo
	}

	if blockNo != Start_Pos && prevBlockNo > Start_Pos {
		missingBlocks = missingBlocks + (prevBlockNo - Start_Pos)
		ns.InsertBlocksInRange(Start_Pos, prevBlockNo-1)
	}

	ns.StopBulkChannel()
	ns.log.Info().Uint64("missing", missingBlocks).Msg("Done with consistency check")
	os.WriteFile("./bulk_end_time.txt", []byte(time.Now().String()), 0644)
}

package indexer

import (
	"io"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	tx "github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
)

// Start setups the indexer
func (ns *Indexer) Check(startFrom uint64, stopAt uint64) {
	ns.log.Info().Uint64("from", startFrom).Uint64("to", stopAt).Msg("Start Check...")

	if stopAt == 0 {
		stopAt = ns.GetBestBlock() - 1
	}
	if ns.fix == true {
		ns.fixIndex(startFrom, stopAt)
	} else {
		ns.checkIndex(startFrom, stopAt)
	}

	// remove clean index logic
	// err := ns.cleanIndex()
	// if err != nil {
	// ns.log.Warn().Err(err).Msg("Failed to clean unexpected data")
	// }
}

func (ns *Indexer) checkIndex(startFrom uint64, stopAt uint64) {
	ns.log.Info().Uint64("startFrom", startFrom).Uint64("stopAt", stopAt).Msg("Check Block range")
	ns.bulk.StartBulkChannel()

	var block doc.DocType
	var err error

	scroll := ns.db.Scroll(db.QueryParams{
		IndexName:    ns.indexNamePrefix + "block",
		TypeName:     "_doc",
		SelectFields: []string{"no"},
		Size:         10000,
		SortField:    "no",
		SortAsc:      false,
		From:         int(startFrom),
		To:           int(stopAt),
	}, func() doc.DocType {
		block := new(doc.EsBlock)
		block.BaseEsType = new(doc.BaseEsType)

		return block
	})

	prevBlockNo := stopAt + 1
	missingBlocks := uint64(0)
	blockNo := startFrom + 1
	for {
		block, err = scroll.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ns.log.Warn().Err(err).Uint64("no", blockNo).Msg("Failed to query block numbers")
			time.Sleep(time.Second)
			continue
		}
		blockNo = block.(*doc.EsBlock).BlockNo

		if blockNo%100000 == 0 {
			ns.log.Info().Uint64("BlockNo", blockNo).Msg("Current Check")
		}
		if blockNo >= prevBlockNo {
			continue
		}
		if blockNo <= startFrom {
			break
		}
		if blockNo < prevBlockNo-1 {
			missingBlocks = missingBlocks + (prevBlockNo - blockNo - 1)
			ns.bulk.InsertBlocksInRange(blockNo+1, prevBlockNo-1)
		}
		prevBlockNo = blockNo
	}

	if blockNo != startFrom && prevBlockNo > startFrom {
		missingBlocks = missingBlocks + (prevBlockNo - startFrom)
		ns.bulk.InsertBlocksInRange(startFrom, prevBlockNo-1)
	}

	ns.bulk.StopBulkChannel()
	ns.log.Info().Uint64("missing", missingBlocks).Msg("Done with consistency check")
}

func (ns *Indexer) fixIndex(startFrom uint64, stopAt uint64) {
	ns.log.Info().Uint64("startFrom", startFrom).Uint64("stopAt", stopAt).Msg("Fix Block range")
	ns.bulk.StartBulkChannel()

	ns.bulk.InsertBlocksInRange(startFrom, stopAt)

	ns.bulk.StopBulkChannel()
	ns.log.Info().Msg("Done with fix")
}

// Start clean the indexer
func (ns *Indexer) cleanIndex() error {
	ns.log.Info().Msg("Clean index Start...")

	// 1. get token list
	tokens := make(map[string]bool)
	ns.ScrollToken(func(tokenDoc *doc.EsToken) {
		tokens[tokenDoc.TokenAddress] = true
	})

	// 2. get token transfer
	ns.ScrollTokenTransfer(func(transferDoc *doc.EsTokenTransfer) {
		if _, ok := tokens[transferDoc.TokenAddress]; !ok {
			ns.log.Info().Str("token", transferDoc.TokenAddress).Msg("Delete token transfer")
			ns.db.Delete(db.QueryParams{
				IndexName: ns.indexNamePrefix + "token_transfer",
				StringMatch: &db.StringMatchQuery{
					Field: "address",
					Value: transferDoc.TokenAddress,
				},
			})

		}
	})

	// 3. get account_tokens
	ns.ScrollAccountTokens(func(accountDoc *doc.EsAccountTokens) {
		if _, ok := tokens[accountDoc.TokenAddress]; !ok {
			ns.log.Info().Str("token", accountDoc.TokenAddress).Msg("Delete account token")
			ns.db.Delete(db.QueryParams{
				IndexName: ns.indexNamePrefix + "account_tokens",
				StringMatch: &db.StringMatchQuery{
					Field: "address",
					Value: accountDoc.TokenAddress,
				},
			})
		}
	})

	// 4. get account_balance
	ns.ScrollBalance(func(balanceDoc *doc.EsAccountBalance) {
		// delete alias account balance only
		if tx.IsBalanceNotResolved(balanceDoc.Id) == true {
			ns.log.Info().Str("id", balanceDoc.Id).Msg("Delete account balance")
			ns.db.Delete(db.QueryParams{
				IndexName: ns.indexNamePrefix + "account_balance",
				StringMatch: &db.StringMatchQuery{
					Field: "_id",
					Value: balanceDoc.Id,
				},
			})
		}
	})

	return nil
}

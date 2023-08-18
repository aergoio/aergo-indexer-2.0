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
	ns.fixIndex(startFrom, stopAt)
	err := ns.cleanIndex()
	if err != nil {
		ns.log.Warn().Err(err).Msg("Failed to clean unexpected data")
	}
}

func (ns *Indexer) fixIndex(startFrom uint64, stopAt uint64) {
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
			ns.log.Warn().Err(err).Msg("Failed to query block numbers")
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

// Start clean the indexer
func (ns *Indexer) cleanIndex() error {
	ns.log.Info().Msg("Clean index Start...")

	// 1. get token list
	scrollToken := ns.db.Scroll(db.QueryParams{
		IndexName:    ns.indexNamePrefix + "token",
		TypeName:     "_doc",
		Size:         10000,
		SelectFields: []string{"name"},
		SortField:    "blockno",
		SortAsc:      true,
	}, func() doc.DocType {
		token := new(doc.EsToken)
		token.BaseEsType = new(doc.BaseEsType)
		return token
	})

	tokens := make(map[string]bool)
	for {
		token, err := scrollToken.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ns.log.Warn().Err(err).Msg("Failed to get token")
			return err
		}
		tokens[token.(*doc.EsToken).Name] = true
	}

	// 2. get token transfer
	scrollTransfer := ns.db.Scroll(db.QueryParams{
		IndexName:    ns.indexNamePrefix + "token_transfer",
		Size:         10000,
		SelectFields: []string{"address"},
		SortField:    "blockno",
		SortAsc:      true,
	}, func() doc.DocType {
		transfer := new(doc.EsTokenTransfer)
		transfer.BaseEsType = new(doc.BaseEsType)
		return transfer
	})

	for {
		transfer, err := scrollTransfer.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ns.log.Warn().Err(err).Msg("Failed to get token transfer")
			return err
		}
		if _, ok := tokens[transfer.(*doc.EsTokenTransfer).TokenAddress]; !ok {
			ns.log.Info().Str("token", transfer.(*doc.EsTokenTransfer).TokenAddress).Msg("Delete token transfer")
			_, err := ns.db.Delete(db.QueryParams{
				IndexName: ns.indexNamePrefix + "token_transfer",
				StringMatch: &db.StringMatchQuery{
					Field: "address",
					Value: transfer.(*doc.EsTokenTransfer).TokenAddress,
				},
			})
			if err != nil {
				ns.log.Warn().Err(err).Msg("Failed to delete token transfer")
				return err
			}
		}
	}

	// 3. get account_tokens
	scrollAccount := ns.db.Scroll(db.QueryParams{
		IndexName:    ns.indexNamePrefix + "account_tokens",
		Size:         10000,
		SelectFields: []string{"address"},
		SortField:    "ts",
		SortAsc:      true,
	}, func() doc.DocType {
		account := new(doc.EsAccountTokens)
		account.BaseEsType = new(doc.BaseEsType)
		return account
	})

	for {
		account, err := scrollAccount.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			ns.log.Warn().Err(err).Msg("Failed to get account token")
			return err
		}
		if _, ok := tokens[account.(*doc.EsAccountTokens).TokenAddress]; !ok {
			ns.log.Info().Str("token", account.(*doc.EsAccountTokens).TokenAddress).Msg("Delete account token")
			ns.db.Delete(db.QueryParams{
				IndexName: ns.indexNamePrefix + "account_tokens",
				StringMatch: &db.StringMatchQuery{
					Field: "address",
					Value: account.(*doc.EsAccountTokens).TokenAddress,
				},
			})
		}
	}

	// 4. get account_balance
	scrollBalance := ns.db.Scroll(db.QueryParams{
		IndexName:    ns.indexNamePrefix + "account_balance",
		Size:         10000,
		SelectFields: []string{"_id"},
		SortField:    "blockno",
		SortAsc:      true,
	}, func() doc.DocType {
		balance := new(doc.EsAccountBalance)
		balance.BaseEsType = new(doc.BaseEsType)
		return balance
	})

	for {
		document, err := scrollBalance.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ns.log.Warn().Err(err).Msg("Failed to get account balance")
			return err
		}
		balance := document.(*doc.EsAccountBalance)

		// delete alias account balance only
		if tx.IsBalanceNotResolved(balance.Id) == true {
			ns.log.Info().Str("id", balance.Id).Msg("Delete account balance")
			ns.db.Delete(db.QueryParams{
				IndexName: ns.indexNamePrefix + "account_balance",
				StringMatch: &db.StringMatchQuery{
					Field: "_id",
					Value: balance.Id,
				},
			})
		}
	}
	return nil
}

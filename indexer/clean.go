package indexer

import (
	"fmt"
	"io"

	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
)

// Start setups the indexer
func (ns *Indexer) RunCleanIndex() error {
	fmt.Println("=======> Start Clean index ..")

	// 1. get token list
	scrollToken := ns.db.Scroll(db.QueryParams{
		IndexName: ns.indexNamePrefix + "token",
		TypeName:  "token",
		Size:      10000,
		SortField: "no",
		SortAsc:   true,
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
		IndexName: ns.indexNamePrefix + "token_transfer",
		TypeName:  "token_transfer",
		Size:      10000,
		SortField: "no",
		SortAsc:   true,
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
				TypeName:  "token_transfer",
				StringMatch: &db.StringMatchQuery{
					Field: "tokenaddress",
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
		IndexName: ns.indexNamePrefix + "account_tokens",
		TypeName:  "account_tokens",
		Size:      10000,
		SortField: "no",
		SortAsc:   true,
	}, func() doc.DocType {
		transfer := new(doc.EsTokenTransfer)
		transfer.BaseEsType = new(doc.BaseEsType)
		return transfer
	})

	for {
		transfer, err := scrollAccount.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			ns.log.Warn().Err(err).Msg("Failed to get account token")
			return err
		}
		if _, ok := tokens[transfer.(*doc.EsTokenTransfer).TokenAddress]; !ok {
			ns.log.Info().Str("token", transfer.(*doc.EsTokenTransfer).TokenAddress).Msg("Delete account token")
			_, err := ns.db.Delete(db.QueryParams{
				IndexName: ns.indexNamePrefix + "account_tokens",
				TypeName:  "account_tokens",
				StringMatch: &db.StringMatchQuery{
					Field: "tokenaddress",
					Value: transfer.(*doc.EsTokenTransfer).TokenAddress,
				},
			})
			if err != nil {
				ns.log.Warn().Err(err).Msg("Failed to delete token transfer")
				return err
			}
		}
	}
	return nil
}
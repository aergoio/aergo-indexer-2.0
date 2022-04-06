package indexer

import (
	"github.com/kjunblk/aergo-indexer/indexer/category"
	doc "github.com/kjunblk/aergo-indexer/indexer/documents"
)

// IndexTxs indexes a list of transactions in bulk
func (ns *Indexer) cccv_nft_mainnet() {

	document :=  doc.EsToken{
		BaseEsType:  &doc.BaseEsType{"Amg5yZU9j5rCYBmCs1TiZ65GpffFBhEBpYyRAyjwXMweouVTeckE"},
		TxId:		"9nCGvpKEY7Yu9zbwCzGwurTzjHKV9qEgH54MtVXY7DpL",
		UpdateBlock:	68592368,
		Name:		"cccv_nft",
		Symbol:		"CNFT",
		Type :		category.ARC2,
	}

	ns.db.Insert(document,ns.indexNamePrefix+"token")
}

func (ns *Indexer) cccv_nft_testnet() {

	document :=  doc.EsToken{
		BaseEsType:  &doc.BaseEsType{"Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF"},
		TxId:		"21J8YmRt3onQYwZnCkwUEk1zV7GsvRMhfFzwdtaeWkyi",
		UpdateBlock:	66638759,
		Name:		"cccv_nft",
		Symbol:		"CNFT",
		Type :		category.ARC2,
	}

	ns.db.Insert(document,ns.indexNamePrefix+"token")
}

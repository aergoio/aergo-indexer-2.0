package indexer

import (
	"bytes"

	"github.com/aergoio/aergo-indexer-2.0/indexer/category"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/types"
)

var cccv_nft_address []byte

func (ns *Indexer) is_cccv_nft(contractAddress []byte) bool {
	return bytes.Equal(contractAddress, cccv_nft_address)
}

func (ns *Indexer) init_cccv_nft_mainnet() {
	cccv_nft_string := "Amg5yZU9j5rCYBmCs1TiZ65GpffFBhEBpYyRAyjwXMweouVTeckE"

	var err error
	cccv_nft_address, err = types.DecodeAddress(cccv_nft_string)
	if err != nil {
		return
	}

	document := doc.EsToken{
		BaseEsType:   &doc.BaseEsType{Id: cccv_nft_string},
		TxId:         "9nCGvpKEY7Yu9zbwCzGwurTzjHKV9qEgH54MtVXY7DpL",
		BlockNo:      68592368,
		Name:         "cccv_nft",
		Name_lower:   "cccv_nft",
		Symbol:       "CNFT",
		Symbol_lower: "cnft",
		Type:         category.ARC2,
		Supply:       "0",
		SupplyFloat:  float32(0),
	}

	ns.db.Insert(document, ns.indexNamePrefix+"token")
}

func (ns *Indexer) init_cccv_nft_testnet() {
	cccv_nft_string := "Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF"

	var err error
	cccv_nft_address, err = types.DecodeAddress(cccv_nft_string)
	if err != nil {
		return
	}

	document := doc.EsToken{
		BaseEsType:   &doc.BaseEsType{Id: cccv_nft_string},
		TxId:         "21J8YmRt3onQYwZnCkwUEk1zV7GsvRMhfFzwdtaeWkyi",
		BlockNo:      66638759,
		Name:         "cccv_nft",
		Name_lower:   "cccv_nft",
		Symbol:       "CNFT",
		Symbol_lower: "cnft",
		Type:         category.ARC2,
		Supply:       "0",
		SupplyFloat:  float32(0),
	}

	ns.db.Insert(document, ns.indexNamePrefix+"token")
}

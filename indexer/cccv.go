package indexer

import (
	"bytes"

	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	tx "github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
)

func (ns *Indexer) isCccvNft(contractAddress []byte) bool {
	return ns.cccvNftAddress != nil && bytes.Equal(contractAddress, ns.cccvNftAddress)
}

func (ns *Indexer) initCccvNft() {
	var cccv_nft_string, txid string
	var blockno uint64
	switch ns.networkTypeForCccv {
	case "mainnet":
		cccv_nft_string = "Amg5yZU9j5rCYBmCs1TiZ65GpffFBhEBpYyRAyjwXMweouVTeckE"
		txid = "9nCGvpKEY7Yu9zbwCzGwurTzjHKV9qEgH54MtVXY7DpL"
		blockno = 68592368
	case "testnet":
		cccv_nft_string = "Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF"
		txid = "21J8YmRt3onQYwZnCkwUEk1zV7GsvRMhfFzwdtaeWkyi"
		blockno = 66638759
	default:
		return
	}

	// init cccv address
	var err error
	ns.cccvNftAddress, err = types.DecodeAddress(cccv_nft_string)
	if err != nil {
		return
	}

	// insert cccv record
	document := &doc.EsToken{
		BaseEsType:   &doc.BaseEsType{Id: cccv_nft_string},
		TxId:         txid,
		BlockNo:      blockno,
		Name:         "cccv_nft",
		Name_lower:   "cccv_nft",
		Symbol:       "CNFT",
		Symbol_lower: "cnft",
		Type:         tx.TokenARC2,
		Supply:       "0",
		SupplyFloat:  float32(0),
	}
	ns.insertToken(BlockType_Sync, document)
}

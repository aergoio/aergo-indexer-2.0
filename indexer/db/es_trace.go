package db

import (
	"reflect"

	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
)

var traceWrite = false

func setTraceWrite(value bool) {
	traceWrite = value
}

func traceESWriteTx(indexName string, document doc.DocType, action string) {
	switch value := document.(type) {
	case *doc.EsTx:
		logger.Debug().Str("action", action).Str("indexName", indexName).Str("hash", value.Id).Msg("Insert TX")
	case *doc.EsContractCall:
		logger.Debug().Str("action", action).Str("indexName", indexName).Str("func", value.Function).Str("TxHash", value.TxHash).Msg("Insert ContractCall")
	case *doc.EsInternalOperations:
		logger.Debug().Str("action", action).Str("indexName", indexName).Str("internalOp", value.Operations).Str("TxHash", value.TxId).Msg("Insert InternalOperations")
	case *doc.EsNFT:
		logger.Debug().Str("action", action).Str("indexName", indexName).Str("NFT", value.Id).Uint64("inBlock", value.BlockNo).Msg("Insert NFT")
	case *doc.EsToken, *doc.EsBlock:
		// do nothing
	default:
		docDataType := reflect.TypeOf(document)
		logger.Debug().Str("action", action).Str("indexName", indexName).Str("typeName", docDataType.String()).Msg("Insert")

	}
}

func toTraceESWrite() bool {
	//
	return !traceWrite
}

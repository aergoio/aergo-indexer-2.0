package indexer

import (
//	"github.com/kjunblk/aergo-indexer-2.0/types"
)

func (ns *Indexer) CreateIndex (documentType string) {

	indexName := ns.indexNamePrefix + documentType
	err := ns.db.CreateIndex(indexName, documentType)

	if err != nil {
		ns.log.Error().Err(err).Str("indexName", indexName).Msg("Error when creating index")
	} else {
		ns.log.Info().Str("indexName", indexName).Msg("Created index")
	}
}


func (ns *Indexer) Rebuild() error {

	ns.log.Warn().Msg("Rebuild all indices. Will sync from scratch and replace index aliases when caught up")

	ns.CreateIndex("tx")
	ns.CreateIndex("name")
	ns.CreateIndex("token")
	ns.CreateIndex("block")
	ns.CreateIndex("contract")
	ns.CreateIndex("token_transfer")
	ns.CreateIndex("account_tokens")
	ns.CreateIndex("nft")

	ns.StartBulkChannel()

	// Get ready to start
	ns.InsertBlocksInRange(ns.StartHeight, uint64(ns.GetNodeBlockHeight()))

	ns.StopBulkChannel()

	ns.UpdateAliasForType("tx")
	ns.UpdateAliasForType("name")
	ns.UpdateAliasForType("token")
	ns.UpdateAliasForType("block")
	ns.UpdateAliasForType("contract")
	ns.UpdateAliasForType("token_transfer")
	ns.UpdateAliasForType("account_tokens")
	ns.UpdateAliasForType("nft")

	return nil
}


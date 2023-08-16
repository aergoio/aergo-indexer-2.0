package indexer

import (
	"errors"

	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
)

func (ns *Indexer) addBlock(blockType BlockType, blockDoc *doc.EsBlock) {
	if blockType == BlockType_Bulk {
		ns.bulk.BChannel.Block <- ChanInfo{ChanType_Add, blockDoc}
	} else {
		err := ns.db.Insert(blockDoc, ns.indexNamePrefix+"block")
		if err != nil {
			ns.log.Error().Str("Id", blockDoc.Id).Err(err).Str("method", "insertBlock").Msg("error while insert")
		}
	}
}

func (ns *Indexer) addTx(blockType BlockType, txDoc *doc.EsTx) {
	if blockType == BlockType_Bulk {
		ns.bulk.BChannel.Tx <- ChanInfo{ChanType_Add, txDoc}
	} else {
		err := ns.db.Insert(txDoc, ns.indexNamePrefix+"tx")
		if err != nil {
			ns.log.Error().Err(err).Str("Id", txDoc.Id).Str("method", "insertTx").Msg("error while insert")
		}
	}
}
func (ns *Indexer) addEvent(eventDoc *doc.EsEvent) {
	err := ns.db.Insert(eventDoc, ns.indexNamePrefix+"event")
	if err != nil {
		ns.log.Error().Err(err).Str("Id", eventDoc.Id).Str("method", "insertEvent").Msg("error while insert")
	}
}

func (ns *Indexer) addContract(contractDoc *doc.EsContract) {
	err := ns.db.Insert(contractDoc, ns.indexNamePrefix+"contract")
	if err != nil {
		ns.log.Error().Err(err).Str("Id", contractDoc.Id).Str("method", "insertContract").Msg("error while insert")
	}
}

func (ns *Indexer) addName(nameDoc *doc.EsName) {
	err := ns.db.Insert(nameDoc, ns.indexNamePrefix+"name")
	if err != nil {
		ns.log.Error().Err(err).Str("Id", nameDoc.Id).Str("method", "insertName").Msg("error while insert")
	}
}

func (ns *Indexer) addToken(tokenDoc *doc.EsToken) {
	err := ns.db.Insert(tokenDoc, ns.indexNamePrefix+"token")
	if err != nil {
		ns.log.Error().Err(err).Str("Id", tokenDoc.Id).Str("method", "insertToken").Msg("error while insert")
	}
}

func (ns *Indexer) addTokenVerified(tokenVerifiedDoc *doc.EsTokenVerified) {
	err := ns.db.Insert(tokenVerifiedDoc, ns.indexNamePrefix+"token_verified")
	if err != nil {
		ns.log.Error().Err(err).Str("Id", tokenVerifiedDoc.Id).Str("method", "insertToken").Msg("error while insert")
	}
}

func (ns *Indexer) addAccountTokens(blockType BlockType, accountTokensDoc *doc.EsAccountTokens) {
	if blockType == BlockType_Bulk {
		if ns.cache.getAccTokens(accountTokensDoc.Id) != true {
			ns.bulk.BChannel.AccTokens <- ChanInfo{ChanType_Add, accountTokensDoc}
		}
	} else {
		err := ns.db.Insert(accountTokensDoc, ns.indexNamePrefix+"account_tokens")
		if err != nil {
			ns.log.Error().Err(err).Str("Id", accountTokensDoc.Id).Str("method", "insertAccountTokens").Msg("error while insert")
		}
	}
}

func (ns *Indexer) addAccountBalance(balanceDoc *doc.EsAccountBalance) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "account_balance",
		StringMatch: &db.StringMatchQuery{
			Field: "id",
			Value: balanceDoc.Id,
		},
	}, func() doc.DocType {
		balance := new(doc.EsAccountBalance)
		balance.BaseEsType = new(doc.BaseEsType)
		return balance
	})
	if err != nil {
		ns.log.Error().Err(err).Str("Id", balanceDoc.Id).Str("method", "insertAccountBalance").Msg("error while select")
	}

	if document != nil { // 기존에 존재하는 주소라면 잔고에 상관없이 update
		accountBalance := document.(*doc.EsAccountBalance)
		if balanceDoc.BlockNo < accountBalance.BlockNo { // blockNo, timeStamp 는 최신으로 저장
			balanceDoc.BlockNo = accountBalance.BlockNo
			balanceDoc.Timestamp = accountBalance.Timestamp
		}
		err = ns.db.Update(balanceDoc, ns.indexNamePrefix+"account_balance", balanceDoc.Id)
	} else if balanceDoc.BalanceFloat > 0 { // 처음 발견된 주소라면 잔고 > 0 일 때만 insert
		err = ns.db.Insert(balanceDoc, ns.indexNamePrefix+"account_balance")
	}
	if err != nil {
		ns.log.Error().Err(err).Str("Id", balanceDoc.Id).Str("method", "insertAccountBalance").Msg("error while insert or update")
	}

	// stake 주소는 whitelist 에 추가
	if balanceDoc.StakingFloat > 0 {
		ns.cache.storeWhiteList(balanceDoc.Id)
	}
}

func (ns *Indexer) addTokenTransfer(blockType BlockType, tokenTransferDoc *doc.EsTokenTransfer) {
	if blockType == BlockType_Bulk {
		ns.bulk.BChannel.TokenTransfer <- ChanInfo{ChanType_Add, tokenTransferDoc}
	} else {
		err := ns.db.Insert(tokenTransferDoc, ns.indexNamePrefix+"token_transfer")
		if err != nil {
			ns.log.Error().Err(err).Str("Id", tokenTransferDoc.Id).Str("method", "insertTokenTransfer").Msg("error while insert")
		}
	}
}

func (ns *Indexer) addNFT(nftDoc *doc.EsNFT) {
	document, err := ns.getNFT(nftDoc.Id)
	if err != nil {
		return
	}
	if document != nil { // 기존에 존재한다면 blockno 가 최신일 때만 update
		if nftDoc.BlockNo > document.BlockNo {
			err = ns.db.Update(nftDoc, ns.indexNamePrefix+"nft", nftDoc.Id)
		}
	} else {
		err = ns.db.Insert(nftDoc, ns.indexNamePrefix+"nft")
	}
	if err != nil {
		ns.log.Error().Err(err).Str("Id", nftDoc.Id).Str("method", "insertNFT").Msg("error while insert or update")
	}
}

func (ns *Indexer) updateToken(tokenDoc *doc.EsTokenUp) {
	err := ns.db.Update(tokenDoc, ns.indexNamePrefix+"token", tokenDoc.Id)
	if err != nil {
		ns.log.Error().Str("Id", tokenDoc.Id).Err(err).Str("method", "updateToken").Msg("error while update")
	}
}

func (ns *Indexer) getNFT(id string) (nftDoc *doc.EsNFT, err error) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "nft",
		StringMatch: &db.StringMatchQuery{
			Field: "id",
			Value: id,
		},
	}, func() doc.DocType {
		nft := new(doc.EsNFT)
		nft.BaseEsType = new(doc.BaseEsType)
		return nft
	})
	if err != nil {
		ns.log.Error().Err(err).Str("Id", id).Str("method", "getNFT").Msg("error while select")
		return nil, err
	}
	if document == nil {
		return nil, nil
	}
	return document.(*doc.EsNFT), nil
}

func (ns *Indexer) getToken(id string) (tokenDoc *doc.EsToken, err error) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "token",
		StringMatch: &db.StringMatchQuery{
			Field: "_id",
			Value: id,
		},
	}, func() doc.DocType {
		token := new(doc.EsToken)
		token.BaseEsType = new(doc.BaseEsType)
		return token
	})
	if err != nil {
		ns.log.Error().Err(err).Str("Id", id).Str("method", "getToken").Msg("error while select")
		return nil, err
	} else if document == nil {
		return nil, nil
	}
	return document.(*doc.EsToken), nil
}

func (ns *Indexer) cntTokenTransfer(id string) (ttCnt uint64, err error) {
	cnt, err := ns.db.Count(db.QueryParams{
		IndexName: ns.indexNamePrefix + "token_transfer",
		StringMatch: &db.StringMatchQuery{
			Field: "address",
			Value: id,
		},
	})
	if err != nil {
		ns.log.Error().Err(err).Str("Id", id).Str("method", "countTokenTransfer").Msg("error while count")
		return 0, err
	}
	return uint64(cnt), nil
}

func (ns *Indexer) ValidChainInfo() error {
	chainInfoFromNode, err := ns.grpcClient.GetChainInfo() // get chain info from node
	if err != nil {
		return err
	}

	document, err := ns.db.SelectOne(db.QueryParams{ // get chain info from db
		IndexName: ns.indexNamePrefix + "chain_info",
		SortField: "version",
		SortAsc:   true,
		From:      0,
	}, func() doc.DocType {
		chainInfo := new(doc.EsChainInfo)
		chainInfo.BaseEsType = new(doc.BaseEsType)
		return chainInfo
	})
	if err != nil {
		ns.log.Info().Err(err).Msg("Could not query chain info, add new one.")
	}
	if document == nil { // if empty in db, put new chain info
		chainInfo := doc.EsChainInfo{
			BaseEsType: &doc.BaseEsType{
				Id: chainInfoFromNode.Id.Magic,
			},
			Mainnet:   chainInfoFromNode.Id.Mainnet,
			Public:    chainInfoFromNode.Id.Public,
			Consensus: chainInfoFromNode.Id.Consensus,
			Version:   uint64(chainInfoFromNode.Id.Version),
		}
		err = ns.db.Insert(&chainInfo, ns.indexNamePrefix+"chain_info")
		if err != nil {
			return err
		}
	} else {
		chainInfoFromDb := document.(*doc.EsChainInfo)
		if chainInfoFromDb.Id != chainInfoFromNode.Id.Magic ||
			chainInfoFromDb.Consensus != chainInfoFromNode.Id.Consensus ||
			chainInfoFromDb.Public != chainInfoFromNode.Id.Public ||
			chainInfoFromDb.Mainnet != chainInfoFromNode.Id.Mainnet ||
			chainInfoFromDb.Version != uint64(chainInfoFromNode.Id.Version) { // valid chain info
			return errors.New("chain info is not matched")
		}
	}
	return nil
}

// UpdateAliasForType updates aliases
func (ns *Indexer) UpdateAliasForType(documentType string) {
	aliasName := ns.aliasNamePrefix + documentType
	indexName := ns.indexNamePrefix + documentType
	err := ns.db.UpdateAlias(aliasName, indexName)
	if err != nil {
		ns.log.Warn().Err(err).Str("aliasName", aliasName).Str("indexName", indexName).Msg("Error when updating alias")
	} else {
		ns.log.Info().Err(err).Str("aliasName", aliasName).Str("indexName", indexName).Msg("Updated alias")
	}
}

// CreateIndexIfNotExists creates the indices and aliases in ES
func (ns *Indexer) CreateIndexIfNotExists(documentType string) error {
	aliasName := ns.aliasNamePrefix + documentType

	// Check for existing index to find out current indexNamePrefix
	exists, indexNamePrefix, err := ns.db.GetExistingIndexPrefix(aliasName, documentType)
	if err != nil {
		ns.log.Error().Err(err).Msg("Error when checking for alias")
		return err
	}

	if exists {
		ns.log.Info().Str("aliasName", aliasName).Str("indexNamePrefix", indexNamePrefix).Msg("Alias found")
		ns.indexNamePrefix = indexNamePrefix
		return nil
	}

	// Create new index
	indexName := ns.indexNamePrefix + documentType
	err = ns.db.CreateIndex(indexName, documentType)
	if err != nil {
		ns.log.Error().Err(err).Str("indexName", indexName).Msg("Error when creating index")
		return err
	} else {
		ns.log.Info().Str("indexName", indexName).Msg("Created index")
	}

	// Update alias
	err = ns.db.UpdateAlias(aliasName, indexName)
	if err != nil {
		ns.log.Error().Err(err).Str("aliasName", aliasName).Str("indexName", indexName).Msg("Error when updating alias")
		return err
	} else {
		ns.log.Info().Str("aliasName", aliasName).Str("indexName", indexName).Msg("Updated alias")
	}
	return nil
}

// GetBestBlockFromDb retrieves the current best block from the db
func (ns *Indexer) GetBestBlockFromDb() (uint64, error) {
	block, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "block",
		SortField: "no",
		SortAsc:   false,
	}, func() doc.DocType {
		block := new(doc.EsBlock)
		block.BaseEsType = new(doc.BaseEsType)
		return block
	})
	if err != nil {
		return 0, err
	}
	if block == nil {
		return 0, errors.New("best block not found")
	}
	return block.(*doc.EsBlock).BlockNo, nil
}

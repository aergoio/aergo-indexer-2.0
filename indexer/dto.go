package indexer

import (
	"io"

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

func (ns *Indexer) addEvent(blockType BlockType, eventDoc *doc.EsEvent) {
	if blockType == BlockType_Bulk {
		ns.bulk.BChannel.Event <- ChanInfo{ChanType_Add, eventDoc}
	} else {
		err := ns.db.Insert(eventDoc, ns.indexNamePrefix+"event")
		if err != nil {
			ns.log.Error().Err(err).Str("Id", eventDoc.Id).Str("method", "insertEvent").Msg("error while insert")
		}
	}
}

func (ns *Indexer) addContract(blockType BlockType, contractDoc *doc.EsContract) {
	if blockType == BlockType_Bulk {
		ns.bulk.BChannel.Contract <- ChanInfo{ChanType_Add, contractDoc}
	} else {
		err := ns.db.Insert(contractDoc, ns.indexNamePrefix+"contract")
		if err != nil {
			ns.log.Error().Err(err).Str("Id", contractDoc.Id).Str("method", "insertContract").Msg("error while insert")
		}
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
	document, err := ns.getAccountBalance(balanceDoc.Id)
	if err != nil {
		ns.log.Error().Err(err).Str("Id", balanceDoc.Id).Str("method", "insertAccountBalance").Msg("error while select")
	}

	if document != nil { // 기존에 존재하는 주소라면 잔고에 상관없이 update
		if balanceDoc.BlockNo < document.BlockNo { // blockNo, timeStamp 는 최신으로 저장
			balanceDoc.BlockNo = document.BlockNo
			balanceDoc.Timestamp = document.Timestamp
		}
		err = ns.db.Update(balanceDoc, ns.indexNamePrefix+"account_balance", balanceDoc.Id)
	} else if balanceDoc.BalanceFloat > 0 { // 처음 발견된 주소라면 잔고 > 0 일 때만 insert
		err = ns.db.Insert(balanceDoc, ns.indexNamePrefix+"account_balance")
	}
	if err != nil {
		ns.log.Error().Err(err).Str("Id", balanceDoc.Id).Str("method", "insertAccountBalance").Msg("error while insert or update")
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

func (ns *Indexer) addWhitelist(whitelistDoc *doc.EsWhitelist) {
	err := ns.db.Insert(whitelistDoc, ns.indexNamePrefix+"whitelist")
	if err != nil {
		ns.log.Error().Err(err).Str("Id", whitelistDoc.Id).Str("method", "insertWhitelist").Msg("error while insert")
	}
}

func (ns *Indexer) updateToken(tokenDoc *doc.EsTokenUpSupply) {
	err := ns.db.Update(tokenDoc, ns.indexNamePrefix+"token", tokenDoc.Id)
	if err != nil {
		ns.log.Error().Str("Id", tokenDoc.Id).Err(err).Str("method", "updateToken").Msg("error while update")
	}
}

func (ns *Indexer) updateTokenVerified(tokenDoc *doc.EsTokenUpVerified) {
	err := ns.db.Update(tokenDoc, ns.indexNamePrefix+"token", tokenDoc.Id)
	if err != nil {
		ns.log.Error().Str("Id", tokenDoc.Id).Err(err).Str("method", "updateToken").Msg("error while update")
	}
}

func (ns *Indexer) updateContractSource(contractDoc *doc.EsContractSource) {
	err := ns.db.Update(contractDoc, ns.indexNamePrefix+"contract", contractDoc.Id)
	if err != nil {
		ns.log.Error().Str("Id", contractDoc.Id).Err(err).Str("method", "updateContractSource").Msg("error while update")
	}
}

func (ns *Indexer) updateContractToken(contractDoc *doc.EsContractToken) {
	err := ns.db.Update(contractDoc, ns.indexNamePrefix+"contract", contractDoc.Id)
	if err != nil {
		ns.log.Error().Str("Id", contractDoc.Id).Err(err).Str("method", "updateContractToken").Msg("error while update")
	}
}

func (ns *Indexer) getContract(id string) (contractDoc *doc.EsContract, err error) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "contract",
		StringMatch: &db.StringMatchQuery{
			Field: "_id",
			Value: id,
		},
	}, func() doc.DocType {
		contract := new(doc.EsContract)
		contract.BaseEsType = new(doc.BaseEsType)
		return contract
	})
	if err != nil {
		ns.log.Error().Err(err).Str("Id", id).Str("method", "getContract").Msg("error while select")
		return nil, err
	} else if document == nil {
		return nil, nil
	}
	return document.(*doc.EsContract), nil
}

func (ns *Indexer) getToken(contractAddr string) (tokenDoc *doc.EsToken, err error) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "token",
		StringMatch: &db.StringMatchQuery{
			Field: "_id",
			Value: contractAddr,
		},
	}, func() doc.DocType {
		token := new(doc.EsToken)
		token.BaseEsType = new(doc.BaseEsType)
		return token
	})
	if err != nil {
		ns.log.Error().Err(err).Str("contract", contractAddr).Str("method", "getToken").Msg("error while select")
		return nil, err
	} else if document == nil {
		return nil, nil
	}
	return document.(*doc.EsToken), nil
}

func (ns *Indexer) getNFT(id string) (nftDoc *doc.EsNFT, err error) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "nft",
		StringMatch: &db.StringMatchQuery{
			Field: "_id",
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

func (ns *Indexer) getAccountBalance(id string) (contractDoc *doc.EsAccountBalance, err error) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "account_balance",
		StringMatch: &db.StringMatchQuery{
			Field: "_id",
			Value: id,
		},
	}, func() doc.DocType {
		balance := new(doc.EsAccountBalance)
		balance.BaseEsType = new(doc.BaseEsType)
		return balance
	})
	if err != nil {
		ns.log.Error().Err(err).Str("Id", id).Str("method", "getAccountBalance").Msg("error while select")
	}
	if document == nil {
		return nil, nil
	}

	return document.(*doc.EsAccountBalance), nil
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

func (ns *Indexer) ScrollToken(fn func(*doc.EsToken)) error {
	scroll := ns.db.Scroll(db.QueryParams{
		IndexName: ns.indexNamePrefix + "token",
		SortField: "blockno",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		token := new(doc.EsToken)
		token.BaseEsType = new(doc.BaseEsType)
		return token
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if token, ok := document.(*doc.EsToken); ok {
			fn(token)
		}
	}
	return nil
}

func (ns *Indexer) ScrollContract(fn func(*doc.EsContract)) error {
	scroll := ns.db.Scroll(db.QueryParams{
		IndexName: ns.indexNamePrefix + "contract",
		SortField: "blockno",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		contract := new(doc.EsContract)
		contract.BaseEsType = new(doc.BaseEsType)
		return contract
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if contract, ok := document.(*doc.EsContract); ok {
			fn(contract)
		}
	}
	return nil
}

func (ns *Indexer) ScrollTokenTransfer(fn func(*doc.EsTokenTransfer)) error {
	scroll := ns.db.Scroll(db.QueryParams{
		IndexName: ns.indexNamePrefix + "token_transfer",
		SortField: "blockno",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		tokenTransfer := new(doc.EsTokenTransfer)
		tokenTransfer.BaseEsType = new(doc.BaseEsType)
		return tokenTransfer
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if tokenTransfer, ok := document.(*doc.EsTokenTransfer); ok {
			fn(tokenTransfer)
		}
	}
	return nil
}

func (ns *Indexer) ScrollAccountTokens(fn func(*doc.EsAccountTokens)) error {
	scroll := ns.db.Scroll(db.QueryParams{
		IndexName: ns.indexNamePrefix + "account_tokens",
		SortField: "ts",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		accountTokens := new(doc.EsAccountTokens)
		accountTokens.BaseEsType = new(doc.BaseEsType)
		return accountTokens
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if accountTokens, ok := document.(*doc.EsAccountTokens); ok {
			fn(accountTokens)
		}
	}
	return nil
}

func (ns *Indexer) ScrollBalance(fn func(*doc.EsAccountBalance)) error {
	scroll := ns.db.Scroll(db.QueryParams{
		IndexName: ns.indexNamePrefix + "account_balance",
		SortField: "staking_float",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		balance := new(doc.EsAccountBalance)
		balance.BaseEsType = new(doc.BaseEsType)
		return balance
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if balance, ok := document.(*doc.EsAccountBalance); ok {
			fn(balance)
		}
	}
	return nil
}

func (ns *Indexer) ScrollWhitelist(fn func(*doc.EsWhitelist)) error {
	scroll := ns.db.Scroll(db.QueryParams{
		IndexName: ns.indexNamePrefix + "whitelist",
		SortField: "type",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		whitelist := new(doc.EsWhitelist)
		whitelist.BaseEsType = new(doc.BaseEsType)
		return whitelist
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if whitelist, ok := document.(*doc.EsWhitelist); ok {
			fn(whitelist)
		}
	}
	return nil
}

package indexer

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

// IndexTxs indexes a list of transactions in bulk
func (ns *Indexer) Miner(RChannel chan BlockInfo, MinerGRPC *client.AergoClientController) {
	var block *types.Block
	blockQuery := make([]byte, 8)

	var err error
	for info := range RChannel {
		// stop miner
		if info.Type == BlockType_StopMiner {
			ns.log.Debug().Msg("stop miner")
			break
		}

		blockHeight := info.Height
		binary.LittleEndian.PutUint64(blockQuery, uint64(blockHeight))

		for {
			block, err = MinerGRPC.GetBlock(blockQuery)
			if err != nil {
				ns.log.Warn().Uint64("blockHeight", blockHeight).Err(err).Msg("Failed to get block")
				time.Sleep(100 * time.Millisecond)
			} else {
				break
			}
		}
		// Get Block doc
		blockDoc := doc.ConvBlock(block, ns.makePeerId(block.Header.PubKey))
		for _, tx := range block.Body.Txs {
			ns.MinerTx(info, blockDoc, tx, MinerGRPC)
		}

		// Add block doc
		ns.insertBlock(info.Type, blockDoc)

		// indexing whitelist balance
		if info.Type == BlockType_Sync && blockHeight%1000 == 0 { // onsync only
			ns.whiteListAddresses.Range(func(k, v interface{}) bool {
				if addr, ok := k.(string); ok {
					if addr, err := types.DecodeAddress(addr); err == nil {
						ns.MinerBalance(info, blockDoc, addr, MinerGRPC)
					}
				}
				return true
			})
		}
	}
}

func (ns *Indexer) MinerTx(info BlockInfo, blockDoc doc.EsBlock, tx *types.Tx, MinerGRPC *client.AergoClientController) {
	// Get Tx doc
	txDoc := doc.ConvTx(tx, blockDoc)

	// add tx doc ( defer )
	defer ns.insertTx(info.Type, txDoc)

	// set tx status
	receipt, err := MinerGRPC.GetReceipt(tx.GetHash())
	if err != nil {
		txDoc.Status = "NO_RECEIPT"
		ns.log.Warn().Str("tx", txDoc.Id).Err(err).Msg("Failed to get tx receipt")
		return
	}
	txDoc.Status = receipt.Status
	if receipt.Status == "ERROR" {
		return
	}

	// Process name transactions
	if tx.GetBody().GetType() == types.TxType_GOVERNANCE && string(tx.GetBody().GetRecipient()) == "aergo.name" {
		nameDoc := doc.ConvName(tx, txDoc.BlockNo)
		ns.insertName(info.Type, nameDoc)
		return
	}

	// Balance from, to
	ns.MinerBalance(info, blockDoc, tx.Body.Account, MinerGRPC)
	if bytes.Equal(tx.Body.Account, tx.Body.Recipient) != true {
		ns.MinerBalance(info, blockDoc, tx.Body.Recipient, MinerGRPC)
	}

	// Process Token and TokenTransfer
	switch txDoc.Category {
	case transaction.TxCall:
	case transaction.TxDeploy:
	case transaction.TxPayload:
	case transaction.TxMultiCall:
	default:
		return
	}

	// Contract Deploy
	if txDoc.Category == transaction.TxDeploy {
		contractDoc := doc.ConvContract(txDoc, receipt.ContractAddress)
		ns.insertContract(info.Type, contractDoc)
	}

	// Process Events
	events := receipt.GetEvents()
	for idx, event := range events {
		ns.MinerEvent(info, blockDoc, txDoc, idx, event, MinerGRPC)
	}

	// POLICY 2 Token
	tType := transaction.MaybeTokenCreation(tx)
	switch tType {
	case transaction.TokenARC1, transaction.TokenARC2:
		name, symbol, decimals := MinerGRPC.QueryTokenInfo(receipt.ContractAddress)
		if name == "" {
			return
		}

		// Add Token doc
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(receipt.ContractAddress, ns.isCccvNft(receipt.ContractAddress))
		tokenDoc := doc.ConvToken(txDoc, receipt.ContractAddress, tType, name, symbol, decimals, supply, supplyFloat)
		ns.insertToken(info.Type, tokenDoc)

		// Add Contract doc
		contractDoc := doc.ConvContract(txDoc, receipt.ContractAddress)
		ns.insertContract(info.Type, contractDoc)

		ns.log.Info().Str("contract", transaction.EncodeAccount(receipt.ContractAddress)).Msg("Token created ( Policy 2 )")
	}

	return
}

func (ns *Indexer) MinerBalance(info BlockInfo, block doc.EsBlock, address []byte, MinerGRPC *client.AergoClientController) {
	if transaction.IsBalanceNotResolved(string(address)) {
		return
	}
	balance, balanceFloat, staking, stakingFloat := MinerGRPC.BalanceOf(address)
	balanceFromDoc := doc.ConvAccountBalance(info.Height, address, block.Timestamp, balance, balanceFloat, staking, stakingFloat)
	ns.insertAccountBalance(info.Type, balanceFromDoc)
}

func (ns *Indexer) MinerEvent(info BlockInfo, blockDoc doc.EsBlock, txDoc doc.EsTx, idx int, event *types.Event, MinerGRPC *client.AergoClientController) {
	switch event.EventName {
	case "new_arc1_token", "new_arc2_token":
		tokenType, contractAddress, err := transaction.UnmarshalEventNewArcToken(event)
		if err != nil {
			ns.log.Error().Err(err).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add Token Doc
		name, symbol, decimals := MinerGRPC.QueryTokenInfo(contractAddress)
		if name == "" {
			return
		}
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.isCccvNft(contractAddress))
		tokenDoc := doc.ConvToken(txDoc, contractAddress, tokenType, name, symbol, decimals, supply, supplyFloat)
		ns.insertToken(info.Type, tokenDoc)

		// Add AccountTokens Doc
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, txDoc.Account, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo), txDoc.Timestamp, txDoc.Account, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add Contract Doc
		contractDoc := doc.ConvContract(txDoc, contractAddress)
		ns.insertContract(info.Type, contractDoc)

		ns.log.Info().Str("contract", transaction.EncodeAccount(contractAddress)).Msg("Token created ( Policy 1 )")
	case "mint":
		contractAddress, accountFrom, accountTo, amountOrId, err := transaction.UnmarshalEventMint(event)
		if err != nil {
			ns.log.Error().Err(err).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add TokenTransfer Doc
		tokenType, tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(contractAddress, amountOrId, ns.isCccvNft(event.ContractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(contractAddress, txDoc, idx, accountFrom, accountTo, tokenId, amount, amountFloat)
		txDoc.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferDoc)

		// Update Token Doc
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.isCccvNft(contractAddress))
		tokenUpDoc := doc.ConvTokenUp(txDoc, contractAddress, supply, supplyFloat)
		ns.updateToken(tokenUpDoc)

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.To, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.To, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc
		if tokenType == transaction.TokenARC2 {
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(contractAddress, tokenTransferDoc.TokenId)
			nftDoc := doc.ConvNFT(tokenTransferDoc, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftDoc)
		}
		ns.log.Info().Str("contract", transaction.EncodeAccount(contractAddress)).Msg("Token minted")
	case "transfer":
		contractAddress, accountFrom, accountTo, amountOrId, err := transaction.UnmarshalEventMint(event)
		if err != nil {
			ns.log.Error().Err(err).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add TokenTransfer Doc
		tokenType, tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(contractAddress, amountOrId, ns.isCccvNft(contractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(contractAddress, txDoc, idx, accountFrom, accountTo, tokenId, amount, amountFloat)
		if tokenTransferDoc.Amount == "" {
			return
		}
		txDoc.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferDoc)

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.To, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.To, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat = MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.From, ns.isCccvNft(contractAddress))
		accountTokensDoc = doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.From, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc ( update NFT )
		if tokenType == transaction.TokenARC2 {
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(contractAddress, tokenId)
			nftDoc := doc.ConvNFT(tokenTransferDoc, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftDoc)
		}
		ns.log.Info().Str("contract", transaction.EncodeAccount(contractAddress)).Msg("Token transfered")
	case "burn":
		contractAddress, accountFrom, accountTo, amountOrId, err := transaction.UnmarshalEventBurn(event)
		if err != nil {
			ns.log.Error().Err(err).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add TokenTransfer Doc
		tokenType, tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(contractAddress, amountOrId, ns.isCccvNft(contractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(contractAddress, txDoc, idx, accountFrom, accountTo, tokenId, amount, amountFloat)
		if tokenTransferDoc.Amount == "" {
			return
		}
		txDoc.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferDoc)

		// Update TokenUp Doc
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.isCccvNft(contractAddress))
		tokenUpDoc := doc.ConvTokenUp(txDoc, contractAddress, supply, supplyFloat)
		ns.updateToken(tokenUpDoc)

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.From, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.From, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc
		if tokenType == transaction.TokenARC2 {
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(contractAddress, tokenId)
			nftDoc := doc.ConvNFT(tokenTransferDoc, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftDoc)
		}
		ns.log.Info().Str("contract", transaction.EncodeAccount(contractAddress)).Msg("Token burned")
	default:
		return
	}
}

func (ns *Indexer) makePeerId(pubKey []byte) string {
	if peerId, is_ok := ns.peerId.Load(string(pubKey)); is_ok == true {
		return peerId.(string)
	}
	cryptoPubKey, err := crypto.UnmarshalPublicKey(pubKey)
	if err != nil {
		return ""
	}
	Id, err := peer.IDFromPublicKey(cryptoPubKey)
	if err != nil {
		return ""
	}
	peer := peer.IDB58Encode(Id)
	ns.peerId.Store(string(pubKey), peer)
	return peer
}

func (ns *Indexer) insertBlock(blockType BlockType, blockDoc doc.EsBlock) {
	if blockType == BlockType_Bulk {
		ns.BChannel.Block <- ChanInfo{ChanType_Add, blockDoc}
	} else {
		err := ns.db.Insert(blockDoc, ns.indexNamePrefix+"block")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertBlock")
		}
	}
}

func (ns *Indexer) insertTx(blockType BlockType, txDoc doc.EsTx) {
	if blockType == BlockType_Bulk {
		ns.BChannel.Tx <- ChanInfo{ChanType_Add, txDoc}
	} else {
		err := ns.db.Insert(txDoc, ns.indexNamePrefix+"tx")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertTx")
		}
	}
}

func (ns *Indexer) insertContract(blockType BlockType, contractDoc doc.EsContract) {
	err := ns.db.Insert(contractDoc, ns.indexNamePrefix+"contract")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertContract")
	}
}

func (ns *Indexer) insertName(blockType BlockType, nameDoc doc.EsName) {
	err := ns.db.Insert(nameDoc, ns.indexNamePrefix+"name")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertName")
	}
}

func (ns *Indexer) insertToken(blockType BlockType, tokenDoc doc.EsToken) {
	err := ns.db.Insert(tokenDoc, ns.indexNamePrefix+"token")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertToken")
	}
}

func (ns *Indexer) insertAccountTokens(blockType BlockType, accountTokensDoc doc.EsAccountTokens) {
	if blockType == BlockType_Bulk {
		if _, ok := ns.accToken.Load(accountTokensDoc.Id); ok {
			return
		} else {
			ns.BChannel.AccTokens <- ChanInfo{ChanType_Add, accountTokensDoc}
			ns.accToken.Store(accountTokensDoc.Id, true)
		}
	} else {
		err := ns.db.Insert(accountTokensDoc, ns.indexNamePrefix+"account_tokens")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertAccountTokens")
		}
	}
}

func (ns *Indexer) insertAccountBalance(blockType BlockType, balanceDoc doc.EsAccountBalance) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "account_balance",
		StringMatch: &db.StringMatchQuery{
			Field: "_id",
			Value: balanceDoc.Id,
		},
	}, func() doc.DocType {
		balance := new(doc.EsAccountBalance)
		balance.BaseEsType = new(doc.BaseEsType)
		return balance
	})
	if err != nil {
		ns.log.Error().Err(err).Msg("insertAccountBalance")
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
		ns.log.Error().Err(err).Msg("insertAccountBalance")
	}

	// stake 주소는 whitelist 에 추가
	if balanceDoc.StakingFloat > 0 {
		ns.whiteListAddresses.Store(balanceDoc.Id, true)
	}
}

func (ns *Indexer) insertTokenTransfer(blockType BlockType, tokenTransferDoc doc.EsTokenTransfer) {
	if blockType == BlockType_Bulk {
		ns.BChannel.TokenTransfer <- ChanInfo{ChanType_Add, tokenTransferDoc}
	} else {
		err := ns.db.Insert(tokenTransferDoc, ns.indexNamePrefix+"token_transfer")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertTokenTransfer")
		}
	}
}

func (ns *Indexer) insertNFT(blockType BlockType, nftDoc doc.EsNFT) {
	document, err := ns.db.SelectOne(db.QueryParams{
		IndexName: ns.indexNamePrefix + "nft",
		StringMatch: &db.StringMatchQuery{
			Field: "_id",
			Value: nftDoc.Id,
		},
	}, func() doc.DocType {
		balance := new(doc.EsNFT)
		balance.BaseEsType = new(doc.BaseEsType)
		return balance
	})
	if err != nil {
		ns.log.Error().Err(err).Msg("insertNft")
	}

	if document != nil { // 기존에 존재한다면 blockno 가 최신일 때만 update
		if nftDoc.BlockNo > document.(*doc.EsNFT).BlockNo {
			err = ns.db.Update(nftDoc, ns.indexNamePrefix+"nft", nftDoc.Id)
		}
	} else {
		err = ns.db.Insert(nftDoc, ns.indexNamePrefix+"nft")
	}
	if err != nil {
		ns.log.Error().Err(err).Msg("insertNft")
	}
}

func (ns *Indexer) updateToken(tokenDoc doc.EsTokenUp) {
	/*
		if blockType == BlockType_Bulk {
			ns.BChannel.Token <- ChanInfo{ChanType_Add, tokenD}
		} else {
			ns.db.Insert(tokenD, ns.indexNamePrefix+"token")
		}
	*/
	err := ns.db.Update(tokenDoc, ns.indexNamePrefix+"token", tokenDoc.Id)
	if err != nil {
		ns.log.Error().Err(err).Msg("updateToken")
	}
}

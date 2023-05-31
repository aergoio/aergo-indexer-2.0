package indexer

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/category"
	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
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

	// set tx status
	receipt, err := MinerGRPC.GetReceipt(tx.GetHash())
	if err != nil {
		txDoc.Status = "NO_RECEIPT"
		ns.log.Warn().Str("tx", txDoc.Id).Err(err).Msg("Failed to get tx receipt")
		ns.insertTx(info.Type, txDoc)
		return
	}
	txDoc.Status = receipt.Status
	if receipt.Status == "ERROR" {
		ns.insertTx(info.Type, txDoc)
		return
	}

	// Process name transactions
	if tx.GetBody().GetType() == types.TxType_GOVERNANCE && string(tx.GetBody().GetRecipient()) == "aergo.name" {
		nameDoc := doc.ConvName(tx, txDoc.BlockNo)
		ns.insertName(info.Type, nameDoc)
		ns.insertTx(info.Type, txDoc)
		return
	}

	// Balance from, to
	ns.MinerBalance(info, blockDoc, tx.Body.Account, MinerGRPC)
	if bytes.Equal(tx.Body.Account, tx.Body.Recipient) != true {
		ns.MinerBalance(info, blockDoc, tx.Body.Recipient, MinerGRPC)
	}

	// Process Token and TokenTransfer
	switch txDoc.Category {
	case category.Call:
	case category.Deploy:
	case category.Payload:
	case category.MultiCall:
	default:
		ns.insertTx(info.Type, txDoc)
		return
	}

	// Contract Deploy
	if txDoc.Category == category.Deploy {
		contractDoc := doc.ConvContract(txDoc, receipt.ContractAddress)
		ns.insertContract(info.Type, contractDoc)
	}

	// Process Events
	events := receipt.GetEvents()
	for idx, event := range events {
		ns.MinerEvent(info, blockDoc, txDoc, idx, event, MinerGRPC)
	}

	// POLICY 2 Token
	tType := category.MaybeTokenCreation(tx)
	switch tType {
	case category.TokenARC1, category.TokenARC2:
		// Add Token doc
		name, symbol, decimals := MinerGRPC.QueryTokenInfo(receipt.ContractAddress)
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(receipt.ContractAddress, ns.isCccvNft(receipt.ContractAddress))
		tokenDoc := doc.ConvToken(txDoc, receipt.ContractAddress, tType, name, symbol, decimals, supply, supplyFloat)
		if tokenDoc.Name != "" {
			// Add Token doc
			ns.insertToken(info.Type, tokenDoc)

			// Add Contract doc
			contractDoc := doc.ConvContract(txDoc, receipt.ContractAddress)
			ns.insertContract(info.Type, contractDoc)

			// TODO : Policy 2 에서는 NFT 처리 없음? - ARC2 토큰 들어오는지 확인 필요

			fmt.Println(">>>>>>>>>>> POLICY 2 :", doc.EncodeAccount(receipt.ContractAddress))
		}
	default:
	}
	ns.insertTx(info.Type, txDoc)
	return
}

func (ns *Indexer) MinerBalance(info BlockInfo, block doc.EsBlock, address []byte, MinerGRPC *client.AergoClientController) {
	if doc.IsBalanceNotResolved(string(address)) {
		return
	}
	balance, balanceFloat, staking, stakingFloat := MinerGRPC.BalanceOf(address)
	balanceFromDoc := doc.ConvAccountBalance(info.Height, address, block.Timestamp, balance, balanceFloat, staking, stakingFloat)
	ns.insertAccountBalance(info.Type, balanceFromDoc)
}

func (ns *Indexer) MinerEvent(info BlockInfo, blockDoc doc.EsBlock, txDoc doc.EsTx, idx int, event *types.Event, MinerGRPC *client.AergoClientController) {
	var args []interface{}
	switch event.EventName {
	case "new_arc1_token", "new_arc2_token":
		// 2022.04.20 FIX
		// 배포된 컨트랙트 주소 값이 return 값으로 출력하던 스펙 변경
		// contractAddress, err := types.DecodeAddress(receipt.Ret[1:len(receipt.Ret)-1])
		err := json.Unmarshal([]byte(event.JsonArgs), &args)
		if err != nil {
			return
		}
		// TODO : 컨트랙트 인자값 바뀔 시 fatal 가능성 존재 - 예외처리 필요
		contractAddress, err := types.DecodeAddress(args[0].(string))
		if err != nil {
			return
		}

		// Add Token Doc
		var tokenType category.TokenType
		if event.EventName == "new_arc1_token" {
			tokenType = category.TokenARC1
		} else {
			tokenType = category.TokenARC2
		}
		name, symbol, decimals := MinerGRPC.QueryTokenInfo(contractAddress)
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.isCccvNft(contractAddress))
		tokenDoc := doc.ConvToken(txDoc, contractAddress, tokenType, name, symbol, decimals, supply, supplyFloat)
		if tokenDoc.Name == "" {
			return
		}
		ns.insertToken(info.Type, tokenDoc)

		// Add AccountTokens Doc ( update Amount )
		tokenTransferDoc := doc.EsTokenTransfer{
			Timestamp:    txDoc.Timestamp,
			TokenAddress: doc.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo),
		}
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, txDoc.Account, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenTransferDoc, txDoc.Account, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add Contract Doc
		contractDoc := doc.ConvContract(txDoc, contractAddress)
		ns.insertContract(info.Type, contractDoc)

		// TODO : NFT 추가가 없음

		fmt.Println(">>>>>>>>>>> POLICY 1 :", doc.EncodeAccount(contractAddress))
	case "mint":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 2 {
			return
		}

		// Add TokenTransfer Doc ( mint )
		tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(event.ContractAddress, args[1], ns.isCccvNft(event.ContractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(event.ContractAddress, txDoc, idx, "MINT", args[0].(string), tokenId, amount, amountFloat)
		if tokenTransferDoc.Amount == "" {
			return
		}
		txDoc.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferDoc) // TODO : TokenTransfer 에서 amount 에 NFT 이름이 들어있는지 디버깅 필요

		// Update TokenUp Doc
		if info.Type == BlockType_Sync {
			supply, supplyFloat := MinerGRPC.QueryTotalSupply(event.ContractAddress, ns.isCccvNft(event.ContractAddress))
			tokenUpDoc := doc.ConvTokenUp(txDoc, event.ContractAddress, supply, supplyFloat)
			ns.updateToken(tokenUpDoc)
		}

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferDoc.To, ns.isCccvNft(event.ContractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenTransferDoc, tokenTransferDoc.To, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc
		if tokenTransferDoc.TokenId != "" { // ARC2
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(event.ContractAddress, tokenTransferDoc.TokenId)
			// ARC2.tokenTransfer.Amount --> nftDoc.Account (ownerOf)
			nftDoc := doc.ConvNFT(event.ContractAddress, tokenTransferDoc, tokenTransferDoc.Amount, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftDoc)
		}
	case "transfer":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 3 {
			return
		}

		// Add TokenTransfer Doc ( transfer )
		tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(event.ContractAddress, args[2], ns.isCccvNft(event.ContractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(event.ContractAddress, txDoc, idx, args[0].(string), args[1].(string), tokenId, amount, amountFloat)
		if tokenTransferDoc.Amount == "" {
			return
		}
		if strings.Contains(tokenTransferDoc.From, "1111111111111111111111111") {
			tokenTransferDoc.From = "MINT"
		} else if strings.Contains(tokenTransferDoc.To, "1111111111111111111111111") {
			tokenTransferDoc.To = "BURN"
		}
		txDoc.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferDoc)

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferDoc.To, ns.isCccvNft(event.ContractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenTransferDoc, tokenTransferDoc.To, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat = MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferDoc.From, ns.isCccvNft(event.ContractAddress))
		accountTokensDoc = doc.ConvAccountTokens(tokenTransferDoc, tokenTransferDoc.From, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc ( update NFT )
		if tokenTransferDoc.TokenId != "" { // ARC2
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(event.ContractAddress, tokenTransferDoc.TokenId)
			// ARC2.tokenTransfer.Amount --> nftDoc.Account (ownerOf)
			nftDoc := doc.ConvNFT(event.ContractAddress, tokenTransferDoc, tokenTransferDoc.Amount, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftDoc)
		}
	case "burn":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 2 {
			return
		}

		// Add TokenTransfer Doc ( burn )
		tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(event.ContractAddress, args[1], ns.isCccvNft(event.ContractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(event.ContractAddress, txDoc, idx, args[0].(string), "BURN", tokenId, amount, amountFloat)
		if tokenTransferDoc.Amount == "" {
			return
		}
		txDoc.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferDoc)

		// Update TokenUp Doc
		if info.Type == BlockType_Sync { // TODO : 성능 문제인듯.. 아마 지우는게 맞음 ( 캐싱을 하던가 )
			supply, supplyFloat := MinerGRPC.QueryTotalSupply(event.ContractAddress, ns.isCccvNft(event.ContractAddress))
			tokenUpDoc := doc.ConvTokenUp(txDoc, event.ContractAddress, supply, supplyFloat)
			ns.updateToken(tokenUpDoc)
		}

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferDoc.From, ns.isCccvNft(event.ContractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenTransferDoc, tokenTransferDoc.From, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc ( delete NFT on Sync only )
		if tokenTransferDoc.TokenId != "" && info.Type == BlockType_Sync { // ARC2. TODO.. Sync 지우기
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(event.ContractAddress, tokenTransferDoc.TokenId)
			// ARC2.tokenTransfer.Amount --> nftDoc.Account (ownerOf)
			nftDoc := doc.ConvNFT(event.ContractAddress, tokenTransferDoc, tokenTransferDoc.Amount, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftDoc)
		}
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
	/*
		if blockType == BlockType_Bulk {
			// to bulk
			ns.BChannel.Contract <- ChanInfo{ChanType_Add, contractD}
		} else {
			// to es
			ns.db.Insert(contractD, ns.indexNamePrefix+"contract")
		}
	*/
	err := ns.db.Insert(contractDoc, ns.indexNamePrefix+"contract")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertContract")
	}
}

func (ns *Indexer) insertName(blockType BlockType, nameDoc doc.EsName) {
	/*
		if blockType == BlockType_Bulk {
			// to bulk
			ns.BChannel.Name <- ChanInfo{ChanType_Add, nameD}
		} else {
			// to es
			ns.db.Insert(nameD, ns.indexNamePrefix+"name")
		}
	*/
	err := ns.db.Insert(nameDoc, ns.indexNamePrefix+"name")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertName")
	}
}

func (ns *Indexer) insertToken(blockType BlockType, tokenDoc doc.EsToken) {
	/*
		if blockType == BlockType_Bulk {
			ns.BChannel.Token <- ChanInfo{ChanType_Add, tokenD}
		} else {
			ns.db.Insert(tokenD, ns.indexNamePrefix+"token")
		}
	*/
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
			ns.BChannel.AccTokens <- ChanInfo{1, accountTokensDoc}
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
	/*
		if blockType == BlockType_Bulk {
			ns.BChannel.NFT <- ChanInfo{ChanType_Add, nftD}
		} else {
			ns.db.Insert(nftD, ns.indexNamePrefix+"nft")
		}
	*/

	// TODO : 과거 nft가 추가될 경우 갱신하지 않는 로직 추가
	err := ns.db.Insert(nftDoc, ns.indexNamePrefix+"nft")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertNFT")
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

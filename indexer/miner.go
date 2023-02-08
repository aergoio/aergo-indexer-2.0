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
			fmt.Println(":::::::::::::::::::::: STOP Miner")
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
		blockD := doc.ConvBlock(block, ns.makePeerId(block.Header.PubKey))
		for _, tx := range block.Body.Txs {
			ns.MinerTx(info, blockD, tx, MinerGRPC)
		}

		// Add block doc
		ns.insertBlock(info.Type, blockD)

		// indexing whitelist balance
		if ns.runMode == "onsync" && blockHeight%ns.whiteListBlockInterval == 0 {
			for _, w := range ns.whiteListAddresses {
				if addr, err := types.DecodeAddress(w); err == nil {
					ns.MinerBalance(info, addr, MinerGRPC)
				}
			}
		}
	}
}

func (ns *Indexer) MinerTx(info BlockInfo, blockD doc.EsBlock, tx *types.Tx, MinerGRPC *client.AergoClientController) {
	// Get Tx doc
	txD := doc.ConvTx(tx, blockD)

	receipt, err := MinerGRPC.GetReceipt(tx.GetHash())
	if err != nil {
		txD.Status = "NO_RECEIPT"
		ns.log.Warn().Str("tx", txD.Id).Err(err).Msg("Failed to get tx receipt")
		ns.insertTx(info.Type, txD)
		return
	}
	txD.Status = receipt.Status
	if receipt.Status == "ERROR" {
		ns.insertTx(info.Type, txD)
		return
	}

	// Process name transactions
	if tx.GetBody().GetType() == types.TxType_GOVERNANCE && string(tx.GetBody().GetRecipient()) == "aergo.name" {
		nameD := doc.ConvName(tx, txD.BlockNo)
		ns.insertName(info.Type, nameD)
		ns.insertTx(info.Type, txD)
		return
	}

	// Balance from, to
	ns.MinerBalance(info, tx.Body.Account, MinerGRPC)
	if bytes.Equal(tx.Body.Account, tx.Body.Recipient) != true {
		ns.MinerBalance(info, tx.Body.Recipient, MinerGRPC)
	}

	// Process Token and TokenTransfer
	switch txD.Category {
	case category.Call:
	case category.Deploy:
	case category.Payload:
	case category.MultiCall:
	default:
		ns.insertTx(info.Type, txD)
		return
	}

	// Contract Deploy
	if txD.Category == category.Deploy {
		contractD := doc.ConvContract(txD, receipt.ContractAddress)
		ns.insertContract(info.Type, contractD)
	}

	// Process Events
	events := receipt.GetEvents()
	for idx, event := range events {
		ns.MinerEvent(info, blockD, txD, idx, event, MinerGRPC)
	}

	// POLICY 2 Token
	tType := MaybeTokenCreation(tx)
	switch tType {
	case TokenCreationType_ARC1, TokenCreationType_ARC2:
		// Add Token doc
		var tokenType category.TokenType
		if tType == TokenCreationType_ARC1 {
			tokenType = category.ARC1
		} else {
			tokenType = category.ARC2
		}
		name, symbol, decimals := MinerGRPC.QueryTokenInfo(receipt.ContractAddress)
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(receipt.ContractAddress, ns.is_cccv_nft(receipt.ContractAddress))
		tokenD := doc.ConvToken(txD, receipt.ContractAddress, tokenType, name, symbol, decimals, supply, supplyFloat)
		if tokenD.Name != "" {
			// Add Token doc
			ns.insertToken(info.Type, tokenD)

			// Add Contract doc
			contractD := doc.ConvContract(txD, receipt.ContractAddress)
			ns.insertContract(info.Type, contractD)

			fmt.Println(">>>>>>>>>>> POLICY 2 :", doc.EncodeAccount(receipt.ContractAddress))
		}
	default:
	}
	ns.insertTx(info.Type, txD)
	return
}

func (ns *Indexer) MinerBalance(info BlockInfo, address []byte, MinerGRPC *client.AergoClientController) {
	balance, balanceFloat, staking, stakingFloat := MinerGRPC.BalanceOf(address)
	balanceFromD := doc.ConvAccountBalance(info.Height, address, balance, balanceFloat, staking, stakingFloat)
	ns.insertAccountBalance(info.Type, balanceFromD)
}

func (ns *Indexer) MinerEvent(info BlockInfo, blockD doc.EsBlock, txD doc.EsTx, idx int, event *types.Event, MinerGRPC *client.AergoClientController) {
	var args []interface{}
	switch event.EventName {
	case "new_arc1_token", "new_arc2_token":
		// 2022.04.20 FIX
		// 배포된 컨트랙트 주소 값이 return 값으로 출력하던 스펙 변경
		// contractAddress, err := types.DecodeAddress(receipt.Ret[1:len(receipt.Ret)-1])
		json.Unmarshal([]byte(event.JsonArgs), &args)
		contractAddress, err := types.DecodeAddress(args[0].(string))
		if err != nil {
			return
		}

		// Add Token Doc
		var tokenType category.TokenType
		if event.EventName == "new_arc1_token" {
			tokenType = category.ARC1
		} else {
			tokenType = category.ARC2
		}
		name, symbol, decimals := MinerGRPC.QueryTokenInfo(contractAddress)
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.is_cccv_nft(contractAddress))
		tokenD := doc.ConvToken(txD, contractAddress, tokenType, name, symbol, decimals, supply, supplyFloat)
		if tokenD.Name == "" {
			return
		}
		ns.insertToken(info.Type, tokenD)

		// Add AccountTokens Doc ( update Amount )
		tokenTransferD := doc.EsTokenTransfer{
			Timestamp:    txD.Timestamp,
			TokenAddress: doc.EncodeAndResolveAccount(contractAddress, txD.BlockNo),
		}
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, txD.Account, ns.is_cccv_nft(contractAddress))
		accountTokensD := doc.ConvAccountTokens(tokenTransferD, txD.Account, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensD)

		// Add Contract Doc
		contractD := doc.ConvContract(txD, contractAddress)
		ns.insertContract(info.Type, contractD)

		fmt.Println(">>>>>>>>>>> POLICY 1 :", doc.EncodeAccount(contractAddress))
	case "mint":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 2 {
			return
		}

		// Add TokenTransfer Doc ( mint )
		tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(event.ContractAddress, args[1], ns.is_cccv_nft(event.ContractAddress))
		tokenTransferD := doc.ConvTokenTransfer(event.ContractAddress, txD, idx, "MINT", args[0].(string), tokenId, amount, amountFloat)
		if tokenTransferD.Amount == "" {
			return
		}
		txD.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferD)

		// Update TokenUp Doc
		if info.Type == BlockType_Sync {
			supply, supplyFloat := MinerGRPC.QueryTotalSupply(event.ContractAddress, ns.is_cccv_nft(event.ContractAddress))
			tokenUpD := doc.ConvTokenUp(txD, event.ContractAddress, supply, supplyFloat)
			ns.updateToken(tokenUpD)
		}

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferD.To, ns.is_cccv_nft(event.ContractAddress))
		accountTokensD := doc.ConvAccountTokens(tokenTransferD, tokenTransferD.To, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensD)

		// Add NFT Doc
		if tokenTransferD.TokenId != "" { // ARC2
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(event.ContractAddress, tokenTransferD.TokenId)
			// ARC2.tokenTransfer.Amount --> nftD.Account (ownerOf)
			nftD := doc.ConvNFT(event.ContractAddress, tokenTransferD, tokenTransferD.Amount, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftD)
		}
	case "transfer":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 3 {
			return
		}

		// Add TokenTransfer Doc
		tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(event.ContractAddress, args[2], ns.is_cccv_nft(event.ContractAddress))
		tokenTransferD := doc.ConvTokenTransfer(event.ContractAddress, txD, idx, args[0].(string), args[1].(string), tokenId, amount, amountFloat)
		if tokenTransferD.Amount == "" {
			return
		}
		if strings.Contains(tokenTransferD.From, "1111111111111111111111111") {
			tokenTransferD.From = "MINT"
		} else if strings.Contains(tokenTransferD.To, "1111111111111111111111111") {
			tokenTransferD.To = "BURN"
		}
		txD.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferD)

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferD.To, ns.is_cccv_nft(event.ContractAddress))
		accountTokensD := doc.ConvAccountTokens(tokenTransferD, tokenTransferD.To, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensD)

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat = MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferD.From, ns.is_cccv_nft(event.ContractAddress))
		accountTokensD = doc.ConvAccountTokens(tokenTransferD, tokenTransferD.From, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensD)

		// Add NFT Doc ( update NFT on Sync only )
		if tokenTransferD.TokenId != "" && info.Type == BlockType_Sync { // ARC2
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(event.ContractAddress, tokenTransferD.TokenId)
			// ARC2.tokenTransfer.Amount --> nftD.Account (ownerOf)
			nftD := doc.ConvNFT(event.ContractAddress, tokenTransferD, tokenTransferD.Amount, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftD)
		}
	case "burn":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 2 {
			return
		}

		// Add TokenTransfer Doc ( burn )
		tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(event.ContractAddress, args[1], ns.is_cccv_nft(event.ContractAddress))
		tokenTransferD := doc.ConvTokenTransfer(event.ContractAddress, txD, idx, args[0].(string), "BURN", tokenId, amount, amountFloat)
		if tokenTransferD.Amount == "" {
			return
		}
		txD.TokenTransfers++
		ns.insertTokenTransfer(info.Type, tokenTransferD)

		// Update TokenUp Doc
		if info.Type == BlockType_Sync {
			supply, supplyFloat := MinerGRPC.QueryTotalSupply(event.ContractAddress, ns.is_cccv_nft(event.ContractAddress))
			tokenUpD := doc.ConvTokenUp(txD, event.ContractAddress, supply, supplyFloat)
			ns.updateToken(tokenUpD)
		}

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(event.ContractAddress, tokenTransferD.From, ns.is_cccv_nft(event.ContractAddress))
		accountTokensD := doc.ConvAccountTokens(tokenTransferD, tokenTransferD.From, balance, balanceFloat)
		ns.insertAccountTokens(info.Type, accountTokensD)

		// Add NFT Doc ( delete NFT on Sync only )
		if tokenTransferD.TokenId != "" && info.Type == BlockType_Sync { // ARC2
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(event.ContractAddress, tokenTransferD.TokenId)
			// ARC2.tokenTransfer.Amount --> nftD.Account (ownerOf)
			nftD := doc.ConvNFT(event.ContractAddress, tokenTransferD, tokenTransferD.Amount, tokenUri, imageUrl)
			ns.insertNFT(info.Type, nftD)
		}
	default:
		return
	}
}

// MaybeTokenCreation runs a heuristic to determine if tx might be creating a token
func MaybeTokenCreation(tx *types.Tx) TokenCreationType {
	txBody := tx.GetBody()

	// We treat the payload (which is part bytecode, part ABI) as text
	// and check that ALL the ARC1/2 keywords are included
	if !(txBody.GetType() == types.TxType_DEPLOY && len(txBody.Payload) > 30) {
		return TokenCreationType_None
	}

	payload := string(txBody.GetPayload())

	keywords := [...]string{"name", "balanceOf", "transfer", "symbol", "totalSupply"}
	for _, keyword := range keywords {
		if !strings.Contains(payload, keyword) {
			return TokenCreationType_None
		}
	}

	suc := true
	keywords1 := [...]string{"decimals"}
	for _, keyword := range keywords1 {
		if !strings.Contains(payload, keyword) {
			suc = false
			break
		}
	}
	if suc {
		return TokenCreationType_ARC1
	}

	suc = true
	keywords2 := [...]string{"ownerOf"}
	for _, keyword := range keywords2 {
		if !strings.Contains(payload, keyword) {
			suc = false
			break
		}
	}
	if suc {
		return TokenCreationType_ARC2
	}
	return TokenCreationType_None
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

func (ns *Indexer) insertBlock(blockType BlockType, blockD doc.EsBlock) {
	if blockType == BlockType_Bulk {
		ns.BChannel.Block <- ChanInfo{ChanType_Add, blockD}
	} else {
		err := ns.db.Insert(blockD, ns.indexNamePrefix+"block")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertBlock")
		}
	}
}

func (ns *Indexer) insertTx(blockType BlockType, txD doc.EsTx) {
	if blockType == BlockType_Bulk {
		ns.BChannel.Tx <- ChanInfo{ChanType_Add, txD}
	} else {
		err := ns.db.Insert(txD, ns.indexNamePrefix+"tx")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertTx")
		}
	}
}

func (ns *Indexer) insertContract(blockType BlockType, contractD doc.EsContract) {
	/*
		if blockType == BlockType_Bulk {
			// to bulk
			ns.BChannel.Contract <- ChanInfo{ChanType_Add, contractD}
		} else {
			// to es
			ns.db.Insert(contractD, ns.indexNamePrefix+"contract")
		}
	*/
	err := ns.db.Insert(contractD, ns.indexNamePrefix+"contract")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertContract")
	}
}

func (ns *Indexer) insertName(blockType BlockType, nameD doc.EsName) {
	/*
		if blockType == BlockType_Bulk {
			// to bulk
			ns.BChannel.Name <- ChanInfo{ChanType_Add, nameD}
		} else {
			// to es
			ns.db.Insert(nameD, ns.indexNamePrefix+"name")
		}
	*/
	err := ns.db.Insert(nameD, ns.indexNamePrefix+"name")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertName")
	}
}

func (ns *Indexer) insertToken(blockType BlockType, tokenD doc.EsToken) {
	/*
		if blockType == BlockType_Bulk {
			ns.BChannel.Token <- ChanInfo{ChanType_Add, tokenD}
		} else {
			ns.db.Insert(tokenD, ns.indexNamePrefix+"token")
		}
	*/
	err := ns.db.Insert(tokenD, ns.indexNamePrefix+"token")
	if err != nil {
		ns.log.Error().Err(err).Msg("insertToken")
	}
}

func (ns *Indexer) insertAccountTokens(blockType BlockType, accountTokensD doc.EsAccountTokens) {
	if blockType == BlockType_Bulk {
		if _, ok := ns.accToken.Load(accountTokensD.Id); ok {
			// fmt.Println("succs", fmt.Sprintf("%s-%s", account, tokenTransfer.TokenAddress))
			return
		} else {
			// fmt.Println("fail", fmt.Sprintf("%s-%s", account, tokenTransfer.TokenAddress))
			ns.BChannel.AccTokens <- ChanInfo{1, accountTokensD}
			ns.accToken.Store(accountTokensD.Id, true)
		}
	} else {
		err := ns.db.Insert(accountTokensD, ns.indexNamePrefix+"account_tokens")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertAccountTokens")
		}
	}
}

func (ns *Indexer) insertAccountBalance(blockType BlockType, balanceD doc.EsAccountBalance) {
	var err error
	exist := ns.db.Exists(ns.indexNamePrefix+"account_balance", balanceD.Id)
	if exist { // 기존에 존재하는 주소라면 잔고에 상관없이 update
		err = ns.db.Update(balanceD, ns.indexNamePrefix+"account_balance", balanceD.Id)
	} else if balanceD.BalanceFloat > 0 { // 처음 발견된 주소라면 잔고 > 0 일 때만 insert
		err = ns.db.Insert(balanceD, ns.indexNamePrefix+"account_balance")
	}
	if err != nil {
		ns.log.Error().Err(err).Msg("insertAccountBalance")
	}
}

func (ns *Indexer) insertTokenTransfer(blockType BlockType, tokenTransferD doc.EsTokenTransfer) {
	if blockType == BlockType_Bulk {
		ns.BChannel.TokenTransfer <- ChanInfo{ChanType_Add, tokenTransferD}
	} else {
		err := ns.db.Insert(tokenTransferD, ns.indexNamePrefix+"token_transfer")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertTokenTransfer")
		}
	}
}

func (ns *Indexer) insertNFT(blockType BlockType, nftD doc.EsNFT) {
	if blockType == BlockType_Bulk {
		ns.BChannel.NFT <- ChanInfo{ChanType_Add, nftD}
	} else {
		err := ns.db.Insert(nftD, ns.indexNamePrefix+"nft")
		if err != nil {
			ns.log.Error().Err(err).Msg("insertNFT")
		}
	}
}

func (ns *Indexer) updateToken(tokenD doc.EsTokenUp) {
	/*
		if blockType == BlockType_Bulk {
			ns.BChannel.Token <- ChanInfo{ChanType_Add, tokenD}
		} else {
			ns.db.Insert(tokenD, ns.indexNamePrefix+"token")
		}
	*/
	err := ns.db.Update(tokenD, ns.indexNamePrefix+"token", tokenD.Id)
	if err != nil {
		ns.log.Error().Err(err).Msg("updateToken")
	}
}

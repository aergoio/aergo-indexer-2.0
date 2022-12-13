package indexer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/category"
	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/types"
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
			block, err = MinerGRPC.GetBlock(context.Background(), &types.SingleBytes{Value: blockQuery})
			if err != nil {
				ns.log.Warn().Uint64("blockHeight", blockHeight).Err(err).Msg("Failed to get block")
				time.Sleep(100 * time.Millisecond)
			} else {
				break
			}
		}
		// Get Block doc
		blockD := ns.ConvBlock(block)
		for _, tx := range block.Body.Txs {
			ns.MinerTx(info, blockD, tx, MinerGRPC)
		}

		// Add block doc
		ns.rec_Block(info.Type, blockD)
	}
}

func (ns *Indexer) MinerTx(info BlockInfo, blockD doc.EsBlock, tx *types.Tx, MinerGRPC *client.AergoClientController) {
	// Get Tx doc
	txD := ns.ConvTx(tx, blockD)

	receipt, err := MinerGRPC.GetReceipt(context.Background(), &types.SingleBytes{Value: tx.GetHash()})
	if err != nil {
		txD.Status = "NO_RECEIPT"
		ns.log.Warn().Str("tx", txD.Id).Err(err).Msg("Failed to get tx receipt")
		ns.rec_Tx(info.Type, txD)
		return
	}
	txD.Status = receipt.Status
	if receipt.Status == "ERROR" {
		ns.rec_Tx(info.Type, txD)
		return
	}

	// Process name transactions
	if tx.GetBody().GetType() == types.TxType_GOVERNANCE && string(tx.GetBody().GetRecipient()) == "aergo.name" {
		nameD := ns.ConvName(tx, txD.BlockNo)
		ns.rec_Name(info.Type, nameD)
		ns.rec_Tx(info.Type, txD)
		return
	}

	// Process Token and TokenTransfer
	switch txD.Category {
	case category.Call:
	case category.Deploy:
	case category.Payload:
	case category.MultiCall:
	default:
		ns.rec_Tx(info.Type, txD)
		return
	}

	// Contract Deploy
	if txD.Category == category.Deploy {
		contractD := ns.ConvContract(txD, receipt.ContractAddress)
		ns.rec_Contract(info.Type, contractD)
	}

	// Process Events
	events := receipt.GetEvents()
	for idx, event := range events {
		ns.MinerEvent(info, blockD, txD, idx, event, MinerGRPC)
	}

	tType := MaybeTokenCreation(tx)
	if tType == TokenCreationType_None {
		return
	}

	// POLICY 2 Token
	tokenD := ns.ConvToken(txD, receipt.ContractAddress, MinerGRPC) // Get ARC Token doc
	if tokenD.Name == "" {
		ns.rec_Tx(info.Type, txD)
		return
	}
	if tType == TokenCreationType_ARC1 {
		tokenD.Type = category.ARC1
	} else {
		tokenD.Type = category.ARC2
	}
	ns.rec_Token(info.Type, tokenD) // Add Token doc

	contractD := ns.ConvContract(txD, receipt.ContractAddress)
	ns.rec_Contract(info.Type, contractD) // Add Contract

	fmt.Println(">>>>>>>>>>> POLICY 2 :", encodeAccount(receipt.ContractAddress))
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

		// Token Doc
		tokenD := ns.ConvToken(txD, contractAddress, MinerGRPC)
		if tokenD.Name == "" {
			return
		}
		if event.EventName == "new_arc1_token" {
			tokenD.Type = category.ARC1
		} else {
			tokenD.Type = category.ARC2
		}
		ns.rec_Token(info.Type, tokenD)

		// TokenTransfer Doc
		tokenTransferD := doc.EsTokenTransfer{
			Timestamp:    txD.Timestamp,
			TokenAddress: ns.encodeAndResolveAccount(contractAddress, txD.BlockNo),
		}
		ns.UpdateAccountTokens(info.Type, contractAddress, tokenTransferD, txD.Account, MinerGRPC)

		// Contract Doc
		contractD := ns.ConvContract(txD, contractAddress)
		ns.rec_Contract(info.Type, contractD)

		fmt.Println(">>>>>>>>>>> POLICY 1 :", encodeAccount(contractAddress))
	case "mint":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 2 {
			return
		}

		// TokenTransfer Doc
		tokenTransferD := ns.ConvTokenTransfer(event.ContractAddress, txD, idx, "MINT", args[0].(string), args[1], MinerGRPC)
		if tokenTransferD.Amount == "" {
			return
		}
		txD.TokenTransfers++
		ns.rec_TokenTransfer(info.Type, tokenTransferD) // Add tokenTransfer doc

		// update Token
		if info.Type == BlockType_Sync {
			ns.UpdateToken(event.ContractAddress, MinerGRPC)
		}

		// update TO-Account
		ns.UpdateAccountTokens(info.Type, event.ContractAddress, tokenTransferD, tokenTransferD.To, MinerGRPC)
		// NEW NFT
		if tokenTransferD.TokenId != "" { // ARC2
			ns.UpdateNFT(info.Type, event.ContractAddress, tokenTransferD, MinerGRPC)
		}
	case "transfer":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 3 {
			return
		}

		tokenTransferD := ns.ConvTokenTransfer(event.ContractAddress, txD, idx, args[0].(string), args[1].(string), args[2], MinerGRPC)
		if tokenTransferD.Amount == "" {
			return
		}

		if strings.Contains(tokenTransferD.From, "1111111111111111111111111") {
			tokenTransferD.From = "MINT"
		} else if strings.Contains(tokenTransferD.To, "1111111111111111111111111") {
			tokenTransferD.To = "BURN"
		}

		txD.TokenTransfers++

		// Add tokenTransfer doc
		ns.rec_TokenTransfer(info.Type, tokenTransferD)

		// update TO-Account
		ns.UpdateAccountTokens(info.Type, event.ContractAddress, tokenTransferD, tokenTransferD.To, MinerGRPC)

		// update FROM-Account
		ns.UpdateAccountTokens(info.Type, event.ContractAddress, tokenTransferD, tokenTransferD.From, MinerGRPC)

		// update NFT on Sync
		if tokenTransferD.TokenId != "" && info.Type == 2 { // ARC2
			ns.UpdateNFT(info.Type, event.ContractAddress, tokenTransferD, MinerGRPC)
		}
	case "burn":
		json.Unmarshal([]byte(event.JsonArgs), &args)
		if args[0] == nil || len(args) < 2 {
			return
		}

		tokenTransferD := ns.ConvTokenTransfer(event.ContractAddress, txD, idx, args[0].(string), "BURN", args[1], MinerGRPC)
		if tokenTransferD.Amount == "" {
			return
		}

		txD.TokenTransfers++

		// Add tokenTransfer doc
		ns.rec_TokenTransfer(info.Type, tokenTransferD)

		// update Token
		if info.Type == BlockType_Sync {
			ns.UpdateToken(event.ContractAddress, MinerGRPC)
		}

		// update FROM-Account
		ns.UpdateAccountTokens(info.Type, event.ContractAddress, tokenTransferD, tokenTransferD.From, MinerGRPC)

		// Delete NFT on Sync
		if tokenTransferD.TokenId != "" && info.Type == 2 { // ARC2
			ns.UpdateNFT(info.Type, event.ContractAddress, tokenTransferD, MinerGRPC)
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

func (ns *Indexer) rec_Block(blockType BlockType, blockD doc.EsBlock) {
	if blockType == BlockType_Bulk {
		ns.BChannel.Block <- ChanInfo{ChanType_Add, blockD}
	} else {
		ns.db.Insert(blockD, ns.indexNamePrefix+"block")
	}
}

func (ns *Indexer) rec_Tx(blockType BlockType, txD doc.EsTx) {
	if blockType == BlockType_Bulk {
		ns.BChannel.Tx <- ChanInfo{ChanType_Add, txD}
	} else {
		ns.db.Insert(txD, ns.indexNamePrefix+"tx")
	}
}

func (ns *Indexer) rec_Contract(blockType BlockType, contractD doc.EsContract) {
	/*
		if blockType == BlockType_Bulk {
			// to bulk
			ns.BChannel.Contract <- ChanInfo{ChanType_Add, contractD}
		} else {
			// to es
			ns.db.Insert(contractD, ns.indexNamePrefix+"contract")
		}
	*/
	ns.db.Insert(contractD, ns.indexNamePrefix+"contract")
}

func (ns *Indexer) rec_Name(blockType BlockType, nameD doc.EsName) {
	/*
		if blockType == BlockType_Bulk {
			// to bulk
			ns.BChannel.Name <- ChanInfo{ChanType_Add, nameD}
		} else {
			// to es
			ns.db.Insert(nameD, ns.indexNamePrefix+"name")
		}
	*/
	ns.db.Insert(nameD, ns.indexNamePrefix+"name")
}

func (ns *Indexer) rec_Token(blockType BlockType, tokenD doc.EsToken) {
	/*
		if blockType == BlockType_Bulk {
			ns.BChannel.Token <- ChanInfo{ChanType_Add, tokenD}
		} else {
			ns.db.Insert(tokenD, ns.indexNamePrefix+"token")
		}
	*/
	ns.db.Insert(tokenD, ns.indexNamePrefix+"token")
}

func (ns *Indexer) rec_TokenTransfer(blockType BlockType, tokenTransferD doc.EsTokenTransfer) {
	if blockType == BlockType_Bulk {
		ns.BChannel.TokenTransfer <- ChanInfo{ChanType_Add, tokenTransferD}
	} else {
		ns.db.Insert(tokenTransferD, ns.indexNamePrefix+"token_transfer")
	}
}

func (ns *Indexer) rec_NFT(blockType BlockType, nftD doc.EsNFT) {
	if blockType == BlockType_Bulk {
		ns.BChannel.TokenTransfer <- ChanInfo{ChanType_Add, nftD}
	} else {
		ns.db.Insert(nftD, ns.indexNamePrefix+"nft")
	}
}

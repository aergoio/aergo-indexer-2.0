package indexer

import (
	"context"
	"time"
	"fmt"
//	"os"
	"encoding/binary"
	"encoding/json"
	"strings"
//	"strconv"
//	"bytes"
	"github.com/kjunblk/aergo-indexer/indexer/category"
	doc "github.com/kjunblk/aergo-indexer/indexer/documents"
	"github.com/kjunblk/aergo-indexer/types"
//	"github.com/mr-tron/base58/base58"
)

// IndexTxs indexes a list of transactions in bulk
func (ns *Indexer) Miner(RChannel chan BlockInfo) error {

	var block *types.Block
	blockQuery := make([]byte, 8)

	var err error
	var tokenTx doc.EsTokenTransfer
	var receipt *types.Receipt
	var args []interface{}
	var events []*types.Event
	var tType int

	for  info := range RChannel {

		// stop miner
		if info.Type == 0 {
			fmt.Println(":::::::::::::::::::::: STOP Minier")
			break
		}

		blockHeight :=  info.Height
		binary.LittleEndian.PutUint64(blockQuery, uint64(blockHeight))

		for {
			block, err = ns.grpcClient.GetBlock(context.Background(), &types.SingleBytes{Value: blockQuery})

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

			// Get Tx doc
			txD := ns.ConvTx(tx, blockD)

			// Process name transactions
			if ns.rec_Name(tx, txD, info.Type) { goto ADD_TX }

			// Process Token and TokenTX
			switch txD.Category {
			case category.Call :
			case category.Deploy :
			case category.Payload :
			default : goto ADD_TX
                        }

//			fmt.Println("category :", txD.Category)
			receipt, err = ns.grpcClient.GetReceipt(context.Background(), &types.SingleBytes{Value: tx.GetHash()})

			if err != nil {
				ns.log.Warn().Str("tx", txD.Id).Err(err).Msg("Failed to get tx receipt")
				goto ADD_TX
			}

			if receipt.Status == "ERROR" { goto ADD_TX }

			// Process Events 
			events = receipt.GetEvents()
			for idx, event := range events {

				switch event.EventName {

				case "new_arc1_token", "new_arc2_token" :

					contractAddress, err := types.DecodeAddress(receipt.Ret[1:len(receipt.Ret)-1])
					if err != nil { continue }

					// Get Token doc
					token := ns.ConvToken(txD, contractAddress)

					// Add Token doc
					if token.Name == "" { continue }

					if event.EventName == "new_arc1_token" {
						token.Type = category.ARC1
					} else {
						token.Type = category.ARC2
					}

					if info.Type == 1 { ns.BChannel.Token <- ChanInfo{1, token} } else { ns.db.Insert(token,ns.indexNamePrefix+"token") }
					fmt.Println(">>>>>>>>>>> POLICY 1 :", encodeAccount(contractAddress))

				case "mint" : 

					json.Unmarshal([]byte(event.JsonArgs), &args)
					if (args[0] == nil) { continue }

					tokenTx = ns.ConvTokenTx(event.ContractAddress, txD, idx, "MINT", args[0].(string), args[1])
					if tokenTx.Amount == "" { continue }

					txD.TokenTransfers ++

					// Add tokenTx doc
					if info.Type == 1 {
						ns.BChannel.TokenTx <- ChanInfo{1, tokenTx}
					} else {
						ns.db.Insert(tokenTx,ns.indexNamePrefix+"token_transfer")
					}

					// update Token
					 if info.Type != 1 { ns.UpdateToken(event.ContractAddress) }

					// update TO-Account
					ns.UpdateAccountTokens(info.Type,event.ContractAddress,tokenTx,tokenTx.To)

					// update NFT
					if (tokenTx.TokenId != "") { // ARC2
						nft := ns.ConvNFT(event.ContractAddress,tokenTx,tokenTx.To)

						if info.Type == 1 {
							 ns.BChannel.NFT <- ChanInfo{1, nft}
						} else {
							ns.db.Insert(nft,ns.indexNamePrefix+"nft")
						}
					}

				case "transfer" : 

					// if strings.Contains(tokenTx.From,"1111111111111111111111111") { tokenTx.From = "MINT" } 
					// if strings.Contains(tokenTx.To,"1111111111111111111111111") { tokenTx.To = "BURN" }
					json.Unmarshal([]byte(event.JsonArgs), &args)
					if (args[0] == nil) { continue }

					tokenTx = ns.ConvTokenTx(event.ContractAddress, txD, idx, args[0].(string), args[1].(string), args[2])
					if tokenTx.Amount == "" { continue }

					txD.TokenTransfers ++

					// Add tokenTx doc
					if info.Type == 1 {
						ns.BChannel.TokenTx <- ChanInfo{1, tokenTx}
					} else {
						ns.db.Insert(tokenTx,ns.indexNamePrefix+"token_transfer")
					}

					// update TO-Account
					ns.UpdateAccountTokens(info.Type,event.ContractAddress,tokenTx,tokenTx.To)

					// update FROM-Account
					ns.UpdateAccountTokens(info.Type,event.ContractAddress,tokenTx,tokenTx.From)

					// update NFT
					if (tokenTx.TokenId != "") { // ARC2
						nft := ns.ConvNFT(event.ContractAddress,tokenTx,tokenTx.To)

						if info.Type == 1 {
							 ns.BChannel.NFT <- ChanInfo{1, nft}
						} else {
							ns.db.Insert(nft,ns.indexNamePrefix+"nft")
						}
					}

				case "burn" : 

					json.Unmarshal([]byte(event.JsonArgs), &args)
					if (args[0] == nil) { continue }

					tokenTx = ns.ConvTokenTx(event.ContractAddress, txD, idx, args[0].(string), "BURN", args[1])
					if tokenTx.Amount == "" { continue }

					txD.TokenTransfers ++

					// Add tokenTx doc
					if info.Type == 1 {
						ns.BChannel.TokenTx <- ChanInfo{1, tokenTx}
					} else {
						ns.db.Insert(tokenTx,ns.indexNamePrefix+"token_transfer")
					}

					// update Token
					if info.Type != 1 { ns.UpdateToken(event.ContractAddress) }

					// update FROM-Account
					ns.UpdateAccountTokens(info.Type,event.ContractAddress,tokenTx,tokenTx.From)

					// Delete NFT --> Burn
					if (tokenTx.TokenId != "") { // ARC2
						nft := ns.ConvNFT(event.ContractAddress,tokenTx,"BURN")

						if info.Type == 1 {
							 ns.BChannel.NFT <- ChanInfo{1, nft}
						} else {
							ns.db.Insert(nft,ns.indexNamePrefix+"nft")
						}
					}

				default : continue
				}
			}

			// POLICY 2 Token
			tType = ns.MaybeTokenCreation(tx)
			switch  tType {
			case 1, 2 :
				// Get ARC Token doc
				token := ns.ConvToken(txD, receipt.ContractAddress)

				if token.Name == "" { goto ADD_TX }

				if tType == 1 {
					token.Type = category.ARC1
				} else {
					token.Type = category.ARC2
				}

				// Add Token doc
				if info.Type == 1 { ns.BChannel.Token <- ChanInfo{1, token} } else { ns.db.Insert(token,ns.indexNamePrefix+"token") }
				fmt.Println(">>>>>>>>>>> POLICY 2 :", encodeAccount(receipt.ContractAddress))

			default :
			}

			// Add Tx doc
	ADD_TX :	if info.Type == 1 { ns.BChannel.Tx <- ChanInfo{1, txD} } else { ns.db.Insert(txD, ns.indexNamePrefix+"tx") }
//			fmt.Println("--> Tx:", d)
		}

		// Add block doc

		if info.Type == 1 { ns.BChannel.Block <- ChanInfo{1, blockD} } else { ns.db.Insert(blockD, ns.indexNamePrefix+"block") }

//		fmt.Println("--> done:", blockHeight)
	}

	return nil
}


func (ns *Indexer) rec_Name(tx *types.Tx, txD doc.EsTx, Type uint) bool {

	if tx.GetBody().GetType() == types.TxType_GOVERNANCE && string(tx.GetBody().GetRecipient()) == "aergo.name" {
		nameDoc := ns.ConvNameTx(tx, txD.BlockNo)

		if Type == 1 {
			// to bulk
			ns.BChannel.Name <- ChanInfo{1, nameDoc}
		} else {
			// to es
			ns.db.Insert(nameDoc, ns.indexNamePrefix+"name")
		}

		return true

	} else {
		return false
	}
}

// MaybeTokenCreation runs a heuristic to determine if tx might be creating a token
func (ns *Indexer) MaybeTokenCreation(tx *types.Tx) int {

	txBody := tx.GetBody()

	// We treat the payload (which is part bytecode, part ABI) as text
	// and check that ALL the ARC1/2 keywords are included

	if !(txBody.GetType() == types.TxType_DEPLOY && len(txBody.Payload)  > 30) {
		return 0
	}

	payload := string(txBody.GetPayload())

	keywords := [...]string{"name", "balanceOf", "transfer", "symbol", "totalSupply"}
	for _, keyword := range keywords {
		if !strings.Contains(payload, keyword) {
			return 0
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
	if suc { return 1 }

	suc = true
	keywords2 := [...]string{"ownerOf"}
	for _, keyword := range keywords2 {
		if !strings.Contains(payload, keyword) {
			suc = false
			break
		}
	}

	if suc { return 2 }

	return 0
}

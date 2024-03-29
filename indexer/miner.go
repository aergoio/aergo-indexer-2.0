package indexer

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/lua_compiler"
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
		blockDoc := doc.ConvBlock(block, ns.cache.getPeerId(block.Header.PubKey))
		for i, tx := range block.Body.Txs {
			txIdx := uint64(i)
			ns.MinerTx(txIdx, info, blockDoc, tx, MinerGRPC)
		}

		// Add block doc
		ns.addBlock(info.Type, blockDoc)

		// update variables per 300 blocks
		if info.Type == BlockType_Sync && blockHeight%300 == 0 {
			ns.cache.refreshVariables(info, blockDoc, MinerGRPC)
		}
	}
}

func (ns *Indexer) MinerTx(txIdx uint64, info BlockInfo, blockDoc *doc.EsBlock, tx *types.Tx, MinerGRPC *client.AergoClientController) {
	// get receipt
	receipt, err := MinerGRPC.GetReceipt(tx.GetHash())
	if err != nil {
		receipt = nil
	}

	// get Tx doc
	txDoc := doc.ConvTx(txIdx, tx, receipt, blockDoc)

	// add tx doc ( defer )
	defer ns.addTx(info.Type, txDoc)

	// Process governance and name transactions
	if tx.GetBody().GetType() == types.TxType_GOVERNANCE && string(tx.GetBody().GetRecipient()) == "aergo.name" {
		nameDoc := doc.ConvName(tx, txDoc.BlockNo)
		ns.addName(nameDoc)
		return
	}

	// Balance from, to
	ns.cache.storeBalance(transaction.EncodeAndResolveAccount(tx.Body.Account, txDoc.BlockNo))
	ns.cache.storeBalance(transaction.EncodeAndResolveAccount(tx.Body.Recipient, txDoc.BlockNo))

	// Process Token and TokenTransfer
	switch txDoc.Category {
	case transaction.TxCall:
	case transaction.TxDeploy:
	case transaction.TxPayload:
	case transaction.TxMultiCall:
	default:
		return
	}

	// Process Contract Deploy
	if txDoc.Category == transaction.TxDeploy {
		contractDoc := doc.ConvContract(txDoc, receipt.ContractAddress)
		ns.addContract(info.Type, contractDoc)
	}

	// Process Events
	events := receipt.GetEvents()
	for _, event := range events {
		ns.MinerEvent(info, blockDoc, txDoc, event, txIdx, MinerGRPC)
	}

	// Process POLICY 2 Token
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
		ns.addToken(tokenDoc)

		// Add Contract Doc
		contractDoc := doc.ConvContract(txDoc, receipt.ContractAddress)
		ns.addContract(info.Type, contractDoc)

		ns.log.Info().Str("contract", transaction.EncodeAccount(receipt.ContractAddress)).Msg("Token created ( Policy 2 )")
	}

	return
}

func (ns *Indexer) MinerEvent(info BlockInfo, blockDoc *doc.EsBlock, txDoc *doc.EsTx, event *types.Event, txIdx uint64, MinerGRPC *client.AergoClientController) {
	// mine all events per contract
	eventDoc := doc.ConvEvent(event, blockDoc, txDoc, txIdx)
	ns.addEvent(info.Type, eventDoc)

	// parse event by contract address
	ns.MinerEventByAddr(blockDoc, txDoc, event, MinerGRPC)

	// parse event by event name
	ns.MinerEventByName(info, blockDoc, txDoc, event, MinerGRPC)
}

func (ns *Indexer) MinerEventByAddr(blockDoc *doc.EsBlock, txDoc *doc.EsTx, event *types.Event, MinerGRPC *client.AergoClientController) {
	if len(event.ContractAddress) != 0 && bytes.Equal(event.ContractAddress, ns.tokenVerifyAddr) == true {
		tokenAddr, err := transaction.UnmarshalEventVerifyToken(event)
		if err != nil {
			ns.log.Error().Err(err).Uint64("Block", blockDoc.BlockNo).Str("Tx", txDoc.Id).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}
		ns.addWhitelist(doc.ConvWhitelist(tokenAddr, "", "token"))
	}
	if len(event.ContractAddress) != 0 && bytes.Equal(event.ContractAddress, ns.contractVerifyAddr) == true {
		tokenAddr, err := transaction.UnmarshalEventVerifyContract(event)
		if err != nil {
			ns.log.Error().Err(err).Uint64("Block", blockDoc.BlockNo).Str("Tx", txDoc.Id).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}
		ns.addWhitelist(doc.ConvWhitelist(tokenAddr, "", "contract"))
	}
}

func (ns *Indexer) MinerEventByName(info BlockInfo, blockDoc *doc.EsBlock, txDoc *doc.EsTx, event *types.Event, MinerGRPC *client.AergoClientController) {
	switch transaction.EventName(event.EventName) {
	case transaction.EventNewArc1Token, transaction.EventNewArc2Token:
		tokenType, contractAddress, err := transaction.UnmarshalEventNewArcToken(event)
		if err != nil {
			ns.log.Error().Err(err).Uint64("Block", blockDoc.BlockNo).Str("Tx", txDoc.Id).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add Token Doc
		name, symbol, decimals := MinerGRPC.QueryTokenInfo(contractAddress)
		if name == "" {
			return
		}
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.isCccvNft(contractAddress))
		tokenDoc := doc.ConvToken(txDoc, contractAddress, tokenType, name, symbol, decimals, supply, supplyFloat)
		ns.addToken(tokenDoc)

		// Add AccountTokens Doc
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, txDoc.Account, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo), txDoc.Timestamp, txDoc.Account, balance, balanceFloat)
		ns.addAccountTokens(info.Type, accountTokensDoc)

		// Add Contract Doc
		contractDoc := doc.ConvContract(txDoc, contractAddress)
		ns.addContract(info.Type, contractDoc)

		ns.log.Info().Str("contract", transaction.EncodeAccount(contractAddress)).Msg("Token created ( Policy 1 )")
	case transaction.EventMint:
		contractAddress, accountFrom, accountTo, amountOrId, err := transaction.UnmarshalEventMint(event)
		if err != nil {
			ns.log.Error().Err(err).Uint64("Block", blockDoc.BlockNo).Str("Tx", txDoc.Id).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add TokenTransfer Doc
		tokenType, tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(contractAddress, amountOrId, ns.isCccvNft(event.ContractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(contractAddress, txDoc, int(event.EventIdx), accountFrom, accountTo, tokenId, amount, amountFloat)
		ns.addTokenTransfer(info.Type, tokenTransferDoc)

		// Update Token Doc
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.isCccvNft(contractAddress))
		tokenUpDoc := doc.ConvTokenUp(txDoc, contractAddress, supply, supplyFloat)
		ns.updateToken(tokenUpDoc)

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.To, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.To, balance, balanceFloat)
		ns.addAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc
		if tokenType == transaction.TokenARC2 {
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(contractAddress, tokenTransferDoc.TokenId)
			nftDoc := doc.ConvNFT(tokenTransferDoc, tokenUri, imageUrl)
			ns.addNFT(nftDoc)
		}
		ns.log.Debug().Str("contract", transaction.EncodeAccount(contractAddress)).Str("type", string(tokenType)).Msg("Event mint")
	case transaction.EventTransfer:
		contractAddress, accountFrom, accountTo, amountOrId, err := transaction.UnmarshalEventTransfer(event)
		if err != nil {
			ns.log.Error().Err(err).Uint64("Block", blockDoc.BlockNo).Str("Tx", txDoc.Id).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add TokenTransfer Doc
		tokenType, tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(contractAddress, amountOrId, ns.isCccvNft(contractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(contractAddress, txDoc, int(event.EventIdx), accountFrom, accountTo, tokenId, amount, amountFloat)
		if tokenTransferDoc.Amount == "" {
			return
		}
		ns.addTokenTransfer(info.Type, tokenTransferDoc)

		// Add AccountTokens Doc ( update TO-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.To, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.To, balance, balanceFloat)
		ns.addAccountTokens(info.Type, accountTokensDoc)

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat = MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.From, ns.isCccvNft(contractAddress))
		accountTokensDoc = doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.From, balance, balanceFloat)
		ns.addAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc ( update NFT )
		if tokenType == transaction.TokenARC2 {
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(contractAddress, tokenId)
			nftDoc := doc.ConvNFT(tokenTransferDoc, tokenUri, imageUrl)
			ns.addNFT(nftDoc)
		}
		ns.log.Debug().Str("contract", transaction.EncodeAccount(contractAddress)).Str("type", string(tokenType)).Msg("Event transfer")
	case transaction.EventBurn:
		contractAddress, accountFrom, accountTo, amountOrId, err := transaction.UnmarshalEventBurn(event)
		if err != nil {
			ns.log.Error().Err(err).Uint64("Block", blockDoc.BlockNo).Str("Tx", txDoc.Id).Str("eventName", event.EventName).Msg("Failed to unmarshal event args")
			return
		}

		// Add TokenTransfer Doc
		tokenType, tokenId, amount, amountFloat := MinerGRPC.QueryOwnerOf(contractAddress, amountOrId, ns.isCccvNft(contractAddress))
		tokenTransferDoc := doc.ConvTokenTransfer(contractAddress, txDoc, int(event.EventIdx), accountFrom, accountTo, tokenId, amount, amountFloat)
		if tokenTransferDoc.Amount == "" {
			return
		}
		ns.addTokenTransfer(info.Type, tokenTransferDoc)

		// Update TokenUp Doc
		supply, supplyFloat := MinerGRPC.QueryTotalSupply(contractAddress, ns.isCccvNft(contractAddress))
		tokenUpDoc := doc.ConvTokenUp(txDoc, contractAddress, supply, supplyFloat)
		ns.updateToken(tokenUpDoc)

		// Add AccountTokens Doc ( update FROM-Account )
		balance, balanceFloat := MinerGRPC.QueryBalanceOf(contractAddress, tokenTransferDoc.From, ns.isCccvNft(contractAddress))
		accountTokensDoc := doc.ConvAccountTokens(tokenType, tokenTransferDoc.TokenAddress, tokenTransferDoc.Timestamp, tokenTransferDoc.From, balance, balanceFloat)
		ns.addAccountTokens(info.Type, accountTokensDoc)

		// Add NFT Doc
		if tokenType == transaction.TokenARC2 {
			tokenUri, imageUrl := MinerGRPC.QueryNFTMetadata(contractAddress, tokenId)
			nftDoc := doc.ConvNFT(tokenTransferDoc, tokenUri, imageUrl)
			ns.addNFT(nftDoc)
		}
		ns.log.Debug().Str("contract", transaction.EncodeAccount(contractAddress)).Str("type", string(tokenType)).Msg("Event burn")
	default:
		return
	}
}

func (ns *Indexer) MinerBalance(block *doc.EsBlock, address string, MinerGRPC *client.AergoClientController) bool {
	addressRaw, err := types.DecodeAddress(address)
	if err != nil || transaction.IsBalanceNotResolved(string(addressRaw)) {
		return false
	}

	balance, balanceFloat, staking, stakingFloat := MinerGRPC.BalanceOf(addressRaw)
	balanceFromDoc := doc.ConvAccountBalance(block.BlockNo, address, block.Timestamp, balance, balanceFloat, staking, stakingFloat)
	ns.addAccountBalance(balanceFromDoc)

	// if staking balance >= 10000, keep track balance
	if stakingFloat >= 10000 {
		return true
	}
	return false
}

func (ns *Indexer) MinerTokenVerified(tokenAddr, contractAddr, metadata string, MinerGRPC *client.AergoClientController) (updateContractAddr string) {
	updateContractAddr, owner, comment, email, regDate, homepageUrl, imageUrl := transaction.UnmarshalMetadataVerifyToken(metadata)

	// remove exist token info
	if contractAddr != "" && updateContractAddr != contractAddr {
		tokenDoc, err := ns.getToken(contractAddr)
		if err != nil || tokenDoc == nil {
			ns.log.Error().Err(err).Str("addr", contractAddr).Msg("tokenDoc is not exist. wait until tokenDoc added")
			return contractAddr
		}

		totalTransfer, err := ns.cntTokenTransfer(contractAddr)
		if err != nil {
			totalTransfer = 0
		}
		tokenVerifiedDoc := doc.ConvTokenUpVerified(tokenDoc, string(NotVerified), "", "", "", "", "", "", "", totalTransfer)
		ns.updateTokenVerified(tokenVerifiedDoc)
		ns.log.Info().Str("contract", contractAddr).Str("token", tokenAddr).Msg("verified token removed")
	}

	// update token info
	if updateContractAddr != "" {
		tokenDoc, err := ns.getToken(updateContractAddr)
		if err != nil || tokenDoc == nil {
			ns.log.Error().Err(err).Str("addr", updateContractAddr).Msg("tokenDoc is not exist. wait until tokenDoc added")
			return contractAddr // 기존 contract address 반환
		}

		totalTransfer, err := ns.cntTokenTransfer(updateContractAddr)
		if err != nil {
			totalTransfer = 0
		}
		tokenVerifiedDoc := doc.ConvTokenUpVerified(tokenDoc, string(Verified), tokenAddr, owner, comment, email, regDate, homepageUrl, imageUrl, totalTransfer)
		ns.updateTokenVerified(tokenVerifiedDoc)
		ns.log.Info().Str("contract", updateContractAddr).Str("token", tokenAddr).Msg("verified token updated")
	}
	return updateContractAddr
}

func (ns *Indexer) MinerContractVerified(tokenAddr, contractAddr, metadata string, MinerGRPC *client.AergoClientController) (updateContractAddr string) {
	updateContractAddr, _, codeUrl := transaction.UnmarshalMetadataVerifyContract(metadata)

	// remove exist contract info
	if contractAddr != "" && contractAddr != updateContractAddr {
		contractDoc, err := ns.getContract(contractAddr)
		if err != nil || contractDoc == nil {
			ns.log.Error().Err(err).Str("addr", contractAddr).Msg("contractDoc is not exist. wait until contractDoc added")
			return contractAddr
		}
		contractUpDoc := doc.ConvContractUp(contractDoc.Id, string(NotVerified), "", "", "")
		ns.updateContract(contractUpDoc)
		ns.log.Info().Str("contract", contractAddr).Str("token", tokenAddr).Msg("verified contract removed")
	}

	// update contract info
	if updateContractAddr != "" {
		contractDoc, err := ns.getContract(updateContractAddr)
		if err != nil || contractDoc == nil {
			ns.log.Error().Err(err).Msg("contractDoc is not exist. wait until contractDoc added")
			return contractAddr // 기존 contract address 반환
		}

		// skip if codeUrl not changed
		var code string
		var status string = string(NotVerified)
		if codeUrl != "" && contractDoc.CodeUrl == codeUrl {
			ns.log.Debug().Str("method", "verifyContract").Str("tokenAddr", tokenAddr).Msg("codeUrl is not changed, skip")
			return updateContractAddr
		}
		code, err = lua_compiler.GetCode(codeUrl)
		if err != nil {
			ns.log.Error().Err(err).Str("method", "verifyContract").Msg("Failed to get code")
		} else if len(code) > 0 {
			status = string(Verified)
		}

		// TODO : valid bytecode
		/*
			bytecode, err := lua_compiler.CompileCode(code)
			if err != nil {
				ns.log.Error().Err(err).Str("method", "verifyContract").Msg("Failed to compile code")
			}

			// compare bytecode and payload
			var status string
			if bytes.Contains([]byte(contractDoc.Payload), bytecode) == true {
				status = string(Verified)
			} else {
				ns.log.Error().Str("method", "verifyContract").Str("tokenAddr", tokenAddr).Msg("Failed to verify contract")
				fmt.Println([]byte(contractDoc.Payload))
				var i interface{}
				json.Unmarshal([]byte(contractDoc.Payload), i)
				fmt.Println(i)
				fmt.Println(bytecode)
				status = string(NotVerified)
			}
		*/

		contractUpDoc := doc.ConvContractUp(updateContractAddr, status, tokenAddr, codeUrl, code)
		ns.updateContract(contractUpDoc)
		ns.log.Info().Str("contract", updateContractAddr).Str("token", tokenAddr).Msg("verified contract updated")
	}
	return updateContractAddr
}

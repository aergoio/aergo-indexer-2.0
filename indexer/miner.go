package indexer

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/aergoio/aergo-lib/log"
	"github.com/mr-tron/base58"
)

var opLog = log.NewLogger("internalOp")
var eventLog = log.NewLogger("event")

// Miner indexes a list of transactions in bulk
func (ns *Indexer) Miner(RChannel chan BlockInfo, MinerGRPC *client.AergoClientController) {
	var block *types.Block
	blockQuery := make([]byte, 8)

	var err error
	for info := range RChannel {
		// stop miner
		if info.Type == BlockType_StopMiner {
			ns.log.Info().Msg("stop miner")
			break
		}

		blockHeight := uint64(info.Height)
		binary.LittleEndian.PutUint64(blockQuery, blockHeight)

		for {
			block, err = MinerGRPC.GetBlock(blockQuery)
			if err != nil {
				ns.log.Warn().Uint64("blockHeight", blockHeight).Err(err).Msg("Failed to get block")
				time.Sleep(100 * time.Millisecond)
			} else {
				break
			}
		}

		// Get Internal Operations
		var txsInternalOps []InternalOperations
		if len(block.Body.Txs) > 0 {
			// request the list of internal operations for this block
			jsonInternalOps, err := MinerGRPC.GetInternalOperations(blockHeight)
			if err != nil {
				ns.log.Warn().Uint64("blockHeight", blockHeight).Err(err).Msg("Failed to get internal operations")
			}
			if len(jsonInternalOps) > 0 {
				// decode the JSON array into objects
				err := json.Unmarshal(jsonInternalOps, &txsInternalOps)
				if err != nil {
					ns.log.Error().Err(err).Uint64("blockHeight", blockHeight).Msg("Failed to unmarshal internal operations tree")
				}
			}
		}

		// Add block doc
		blockDoc := doc.ConvBlock(block, ns.cache.getPeerId(block.Header.PubKey))
		ns.addBlock(info.Type, blockDoc)

		// Process each transaction in the block
		for i, tx := range block.Body.Txs {
			txIdx := uint64(i)
			// find the internal operations for this transaction
			var internalOps *InternalOperations
			for _, ops := range txsInternalOps {
				if ops.TxHash == base58.Encode(tx.GetHash()) {
					internalOps = &ops
					break
				}
			}
			// process the transaction
			ns.MinerTx(txIdx, info, blockDoc, tx, internalOps, MinerGRPC)
		}

		// update variables per 300 blocks
		if info.Type == BlockType_Sync && blockHeight%300 == 0 {
			ns.cache.refreshVariables(info, blockDoc, MinerGRPC)
		}
	}
}

type CallInfo struct {
	BlockHeight uint64
	Timestamp   time.Time
	TxHash      string
	TxIdx       uint64
	TxDoc       *doc.EsTx
	CallIdx     uint64
	SendIdx     uint64
}

func (ns *Indexer) MinerTx(txIdx uint64, info BlockInfo, blockDoc *doc.EsBlock, tx *types.Tx, internalOps *InternalOperations, MinerGRPC *client.AergoClientController) {
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

	sender := transaction.EncodeAndResolveAccount(tx.Body.Account, txDoc.BlockNo)
	recipient := transaction.EncodeAndResolveAccount(tx.Body.Recipient, txDoc.BlockNo)

	// Balance from, to
	ns.cache.storeBalance(sender)
	ns.cache.storeBalance(recipient)

	// Process Contract Deploy
	if txDoc.Category == transaction.TxDeploy {
		contractDoc := doc.ConvContractFromTx(txDoc, receipt.ContractAddress)
		if contractDoc != nil {
			ns.addContract(info.Type, contractDoc)
		}
	}

	// Process the internal operations for this transaction
	if internalOps != nil {
		callInfo := CallInfo{
			BlockHeight: blockDoc.BlockNo,
			Timestamp:   blockDoc.Timestamp,
			TxHash:      internalOps.TxHash,
			TxIdx:       txIdx,
			TxDoc:       txDoc,
			CallIdx:     1,
			SendIdx:     0,
		}
		// register external call
		txCall := internalOps.Call
		txCallDoc := doc.ConvContractCall(callInfo.BlockHeight, callInfo.Timestamp, callInfo.TxHash, callInfo.TxIdx, callInfo.CallIdx, sender, txCall.Contract, txCall.Function, txCall.Args, txCall.Amount, txDoc.Status == "ERROR")
		ns.addContractCall(txCallDoc)
		// Process internal operations
		if len(txCall.Operations) > 0 {
			ns.MinerTxInternalOps(&callInfo, &txCall)
		}
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

		ns.log.Info().Str("contract", transaction.EncodeAccount(receipt.ContractAddress)).Msg("Token created ( Policy 2 )")
	}

	return
}

type InternalOperation struct {
	Operation string        `json:"op"`
	Amount    string        `json:"amount,omitempty"`
	Args      []string      `json:"args"`
	Result    string        `json:"result,omitempty"`
	Call      *InternalCall `json:"call,omitempty"`
	Reverted  bool          `json:"reverted,omitempty"`
}

type InternalCall struct {
	Contract   string              `json:"contract,omitempty"`
	Function   string              `json:"function,omitempty"`
	Args       []interface{}       `json:"args,omitempty"`
	Amount     string              `json:"amount,omitempty"`
	Operations []InternalOperation `json:"operations,omitempty"`
}

type InternalOperations struct {
	TxHash string       `json:"txhash"`
	Call   InternalCall `json:"call"`
}

func (ns *Indexer) MinerTxInternalOps(callInfo *CallInfo, outerCall *InternalCall) {
	// save the entire tree of internal operations for the transaction
	// re-encode operations to json
	jsonOperations, err := json.Marshal(outerCall)
	if err != nil {
		ns.log.Error().Err(err).Str("txHash", callInfo.TxHash).Str("contract", outerCall.Contract).Msg("Failed to marshal internal operations")
		return
	}
	opLog.Debug().Str("txHash", callInfo.TxHash).Str("contract", outerCall.Contract).
		Str("operations", string(jsonOperations)).Msg("Processing internal operations")
	// save to db
	internalOpsDoc := doc.ConvInternalOperations(callInfo.TxHash, string(jsonOperations))
	ns.addInternalOperations(internalOpsDoc)

	// process each operation from this contract
	for _, operation := range outerCall.Operations {
		ns.MinerContractInternalOp(callInfo, outerCall.Contract, operation, operation.Reverted)
	}
}

func (ns *Indexer) MinerContractInternalOp(callInfo *CallInfo, contract string, operation InternalOperation, reverted bool) {
	opLog.Debug().Str("txHash", callInfo.TxHash).Str("contract", contract).
		Str("operation", operation.Operation).Msg("Processing internal operation")

	internalOpReverted := callInfo.TxDoc.Status == "ERROR" || reverted

	// if the transaction didn't fail and the operation was not reverted...
	if !internalOpReverted {
		// register transfers, deploy, etc.
		ns.MinerInternalOp(callInfo, contract, operation)
	}

	// if it has a call to another contract
	if operation.Call != nil {
		internalCall := operation.Call
		// increment call index
		callInfo.CallIdx++

		// register internal call
		internalCallDoc := doc.ConvContractCall(callInfo.BlockHeight, callInfo.Timestamp, callInfo.TxHash, callInfo.TxIdx, callInfo.CallIdx, contract, internalCall.Contract, internalCall.Function, internalCall.Args, internalCall.Amount, internalOpReverted)
		ns.addContractCall(internalCallDoc)

		// process each operation from this internal call
		for _, nestedOperation := range internalCall.Operations {
			is_reverted := reverted || nestedOperation.Reverted
			ns.MinerContractInternalOp(callInfo, internalCall.Contract, nestedOperation, is_reverted)
		}
	}
}

func (ns *Indexer) MinerInternalOp(callInfo *CallInfo, contract string, operation InternalOperation) {

	// register individual internal operation - not needed
	//internalOpDoc := doc.ConvInternalOperation(txHash, contract, operation.Operation, operation.Amount, operation.Args, operation.Result)
	//ns.addInternalOperation(internalOpDoc)

	// if it's a send operation
	if operation.Operation == "send" ||
	  (operation.Operation == "call" && operation.Amount != "") ||
	  (operation.Operation == "deploy" && operation.Amount != "") {
		// register the internal transfer of aergo tokens
		sender := contract
		var recipient string
		if operation.Operation == "send" || operation.Operation == "call" {
			recipient = operation.Args[0]
		} else { // deploy
			// get the address of the new contract from the result
			recipient = operation.Result
		}
		amount := operation.Amount

		callInfo.SendIdx++
		aergoTransferDoc := doc.ConvAergoTransfer(callInfo.TxDoc, callInfo.SendIdx, sender, recipient, amount)
		ns.addTokenTransfer(BlockType_Sync, aergoTransferDoc)

		// check the new balance of the sender and recipient
		ns.cache.storeBalance(sender)
		ns.cache.storeBalance(recipient)
	}

	// if it's a stake operation
	if operation.Operation == "stake" {
		// TODO: register staking of aergo tokens
	} else if operation.Operation == "unstake" {
		// TODO: register unstaking of aergo tokens
	}

	// if it's an internal contract deployment
	if operation.Operation == "deploy" {
		creator := contract
		// extract source code and deploy args
		sourceCode := operation.Args[0]
		deployArgs := operation.Args[1:]
		// get the address of the new contract from the result
		contractAddr := operation.Result
		// TODO: register new contract
		contractDoc := doc.ConvContractFromCall(callInfo.BlockHeight, callInfo.Timestamp, callInfo.TxHash, contractAddr, creator, sourceCode, deployArgs)
		if contractDoc != nil {
			ns.addContract(BlockType_Sync, contractDoc)
		}
	}

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
		contractDoc := doc.ConvInternalContract(txDoc, contractAddress)
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
		eventLog.Debug().Str("contract", transaction.EncodeAccount(contractAddress)).Str("type", string(tokenType)).Msg("Event mint")
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
		eventLog.Debug().Str("contract", transaction.EncodeAccount(contractAddress)).Str("type", string(tokenType)).Msg("Event transfer")
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
		eventLog.Debug().Str("contract", transaction.EncodeAccount(contractAddress)).Str("type", string(tokenType)).Msg("Event burn")
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
			ns.log.Error().Err(err).Str("addr", contractAddr).Msg("tokenDoc does not exist. wait until tokenDoc added")
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
			ns.log.Error().Err(err).Str("addr", updateContractAddr).Msg("tokenDoc does not exist. wait until tokenDoc added")
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

// it appears that this function is used for 2 different cases:
// 1. verifying and updating the contract source code
// 2. updating the verified token status
// TODO: separate the logic into two different functions
func (ns *Indexer) MinerContractVerified(tokenSymbol, contractAddr, metadata string, MinerGRPC *client.AergoClientController) (updateContractAddr string) {
	updateContractAddr, _, _ = transaction.UnmarshalMetadataVerifyContract(metadata)

	// remove existing contract info (verified token)
	if contractAddr != "" && contractAddr != updateContractAddr {
		contractDoc, err := ns.getContract(contractAddr)
		if err != nil || contractDoc == nil {
			ns.log.Error().Err(err).Str("addr", contractAddr).Msg("contractDoc does not exist. wait until contractDoc is added")
			return contractAddr
		}
		contractUpDoc := doc.ConvContractToken(contractDoc.Id, string(NotVerified), "")
		ns.updateContractToken(contractUpDoc)
		ns.log.Info().Str("contract", contractAddr).Str("token", tokenSymbol).Msg("verified contract removed")
	}

	// update contract info
	if updateContractAddr != "" {
		contractDoc, err := ns.getContract(updateContractAddr)
		if err != nil || contractDoc == nil {
			ns.log.Error().Err(err).Msg("contractDoc does not exist. wait until contractDoc is added")
			return contractAddr // 기존 contract address 반환
		}

		/*
			// skip if codeUrl not changed
			var code string
			if codeUrl != "" && contractDoc.CodeUrl == codeUrl {
				ns.log.Debug().Str("method", "verifyContract").Str("token", tokenSymbol).Msg("codeUrl is not changed, skip")
				return updateContractAddr
			}
			code, err = lua_compiler.GetCode(codeUrl)
			if err != nil {
				ns.log.Error().Err(err).Str("method", "verifyContract").Msg("Failed to get code")
			} else if len(code) > 0 {
				...
			}
		*/

		// TODO : valid bytecode
		/*
			bytecode, err := lua_compiler.CompileCode(code)
			if err != nil {
				ns.log.Error().Err(err).Str("method", "verifyContract").Msg("Failed to compile code")
			}

			// compare bytecode and payload
			if bytes.Equal([]byte(contractDoc.ByteCode), bytecode) == true {
				...
			} else {
				ns.log.Error().Str("method", "verifyContract").Str("token", tokenSymbol).Msg("Failed to verify contract")
				fmt.Println([]byte(contractDoc.Payload))
				var i interface{}
				json.Unmarshal([]byte(contractDoc.Payload), i)
				fmt.Println(i)
				fmt.Println(bytecode)
			}
		*/

		contractUpDoc := doc.ConvContractToken(updateContractAddr, string(Verified), tokenSymbol)
		ns.updateContractToken(contractUpDoc)
		ns.log.Info().Str("contract", updateContractAddr).Str("token", tokenSymbol).Msg("verified contract updated")
	}
	return updateContractAddr
}

// TODO: use this function in the backend
func (ns *Indexer) checkContractSourceCode(contractAddress, sourceCode string) (status string) {
	contractDoc, err := ns.getContract(contractAddress)
	if err != nil || contractDoc == nil {
		ns.log.Error().Err(err).Str("addr", contractAddress).Msg("not found")
		return "this contract is not yet added to the index. wait until it is indexed or check the contract address"
	}

	// compile the source code
	bytecode, _, err := doc.CompileSourceCode(sourceCode)
	if err != nil {
		ns.log.Error().Err(err).Str("addr", contractAddress).Msg("failed to compile source code")
		return "compile error"
	}

	// compare the generated bytecode with the contract bytecode
	isCorrect := bytes.Equal(bytecode, []byte(contractDoc.ByteCode))

	if isCorrect {
		// store the source code in the contract doc
		contractUpDoc := doc.ConvContractSource(contractDoc.Id, sourceCode)
		ns.updateContractSource(contractUpDoc)
		ns.log.Info().Str("contract", contractAddress).Msg("contract source code updated")
		return "OK"
	} else {
		ns.log.Error().Str("contract", contractAddress).Msg("invalid source code")
		return "invalid source code"
	}
}

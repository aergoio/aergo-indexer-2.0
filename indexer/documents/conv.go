package documents

import (
	"fmt"
	"math/big"
	"strings"
	"time"
	"bytes"
	"encoding/binary"

	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/aergoio/aergo-indexer-2.0/lua_compiler"
	"github.com/mr-tron/base58"
	"google.golang.org/protobuf/proto"
)

// ConvBlock converts Block from RPC into Elasticsearch type - 1.0
func ConvBlock(block *types.Block, blockProducer string) *EsBlock {
	rewardAmount := ""
	if len(block.Header.Consensus) > 0 {
		rewardAmount = "160000000000000000"
	}
	return &EsBlock{
		BaseEsType:    &BaseEsType{Id: base58.Encode(block.Hash)},
		Timestamp:     time.Unix(0, block.Header.Timestamp),
		BlockNo:       block.Header.BlockNo,
		PreviousBlock: base58.Encode(block.Header.PrevBlockHash),
		TxCount:       uint64(len(block.Body.Txs)),
		Size:          uint64(proto.Size(block)),
		Coinbase:      transaction.EncodeAndResolveAccount(block.Header.CoinbaseAccount, block.Header.BlockNo),
		BlockProducer: blockProducer,
		RewardAccount: transaction.EncodeAndResolveAccount(block.Header.Consensus, block.Header.BlockNo),
		RewardAmount:  rewardAmount,
	}
}

// ConvTx converts Tx from RPC into Elasticsearch type
func ConvTx(txIdx uint64, tx *types.Tx, receipt *types.Receipt, blockDoc *EsBlock) *EsTx {
	var status string = "NO_RECEIPT"
	var result string
	var gasUsed uint64
	var feeDelegation bool
	var feeUsed string
	var gasPrice string
	var contract string
	if receipt != nil {
		status = receipt.Status
		gasUsed = receipt.GasUsed
		contract = transaction.EncodeAndResolveAccount(receipt.ContractAddress, blockDoc.BlockNo)
		feeDelegation = receipt.FeeDelegation
		result = receipt.Ret
		feeUsed = big.NewInt(0).SetBytes(receipt.FeeUsed).String()
	}
	// TODO: currently, gas price always zero in tx. so, use default price
	gasPrice = "50000000000"
	// gasPrice := big.NewInt(0).SetBytes(tx.GetBody().GasPrice)
	amount := big.NewInt(0).SetBytes(tx.GetBody().Amount)
	category, method := transaction.DetectTxCategory(tx)
	if len(method) > 50 {
		method = method[:50]
	}
	nonce := tx.Body.Nonce

	return &EsTx{
		BaseEsType:    &BaseEsType{Id: base58.Encode(tx.Hash)},
		BlockNo:       blockDoc.BlockNo,
		BlockId:       blockDoc.Id,
		Timestamp:     blockDoc.Timestamp,
		TxIdx:         txIdx,
		Payload:       tx.GetBody().GetPayload(),
		Account:       transaction.EncodeAndResolveAccount(tx.Body.Account, blockDoc.BlockNo),
		Recipient:     transaction.EncodeAndResolveAccount(tx.Body.Recipient, blockDoc.BlockNo),
		Amount:        amount.String(),
		AmountFloat:   bigIntToFloat(amount, 18),
		Type:          uint64(tx.Body.Type),
		Category:      category,
		Method:        method,
		Status:        status,
		Result:        result,
		Contract:      contract,
		Nonce:         nonce,
		FeeDelegation: feeDelegation,
		GasPrice:      gasPrice,
		GasUsed:       gasUsed,
		GasLimit:      tx.Body.GasLimit,
		FeeUsed:       feeUsed,
	}
}

// ConvContractFromTx creates document for contract creation
func ConvContractFromTx(txDoc *EsTx, contractAddressByte []byte) *EsContract {
	byteCode, sourceCode, abi, deployArgs := extractContractCode(txDoc.Payload)
	contractAddress := transaction.EncodeAndResolveAccount(contractAddressByte, txDoc.BlockNo)
	return ConvContract(txDoc.BlockNo, txDoc.Timestamp, txDoc.GetID(), contractAddress, txDoc.Account, byteCode, abi, sourceCode, deployArgs)
}

func ConvContractFromCall(blockHeight uint64, timestamp time.Time, txHash, contractAddress, creator, sourceCode string, deployArgs []string) *EsContract {
	byteCode, abi, err := CompileSourceCode(sourceCode)
	if err != nil {
		panic(err)
	}
	var deployArgsStr string
	if len(deployArgs) > 0 {
		deployArgsStr = "[" + strings.Join(deployArgs, ",") + "]"
	}
	return ConvContract(blockHeight, timestamp, txHash, contractAddress, creator, byteCode, abi, sourceCode, deployArgsStr)
}

func ConvContract(blockHeight uint64, timestamp time.Time, txHash, contractAddress, creator string, byteCode []byte, abi, sourceCode, deployArgs string) *EsContract {
	return &EsContract{
		BaseEsType: &BaseEsType{Id: contractAddress},
		BlockNo:    blockHeight,
		Timestamp:  timestamp,
		TxId:       txHash,
		Creator:    creator,
		ABI:        abi,
		ByteCode:   byteCode,
		SourceCode: sourceCode,
		DeployArgs: deployArgs,
	}
}

func ConvInternalContract(txDoc *EsTx, contractAddress []byte) *EsContract {
	return &EsContract{
		BaseEsType: &BaseEsType{Id: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		Creator:    txDoc.Account,
		TxId:       txDoc.GetID(),
		BlockNo:    txDoc.BlockNo,
		Timestamp:  txDoc.Timestamp,
	}
}

func ConvContractSource(contractAddress string, sourceCode string) *EsContractSource {
	return &EsContractSource{
		BaseEsType:     &BaseEsType{Id: contractAddress},
		SourceCode:     sourceCode,
	}
}

func ConvContractToken(contractAddress string, status, token string) *EsContractToken {
	return &EsContractToken{
		BaseEsType:     &BaseEsType{Id: contractAddress},
		VerifiedToken:  token,
		VerifiedStatus: status,
	}
}

// returns: bytecode, sourceCode, abi, deployArgs
func extractContractCode(payload []byte) ([]byte, string, string, string) {
	if len(payload) <= 12 {
		return nil, "", "", ""
	}
	// check for LuaJIT bytecode signature at position 8
	if bytes.HasPrefix(payload[8:], []byte{0x1b, 0x4c, 0x4a}) {
		// before hardfork 4, the deploy contains the contract bytecode, abi and deploy args
		bytecode, abi, deployArgs := extractByteCode(payload)
		return bytecode, "", abi, deployArgs
	}
	// on hardfork 4, the deploy contains the contract source code and deploy args
	sourceCode, deployArgs := extractSourceCode(payload)
	bytecode, abi, err := CompileSourceCode(sourceCode)
	if err != nil {
		panic(err)
	}
	return bytecode, sourceCode, abi, deployArgs
}

func extractByteCode(payload []byte) ([]byte, string, string) {
	// read the length of the first section
	codeAbiLength := binary.BigEndian.Uint32(payload[:4])
	// read the bytecode length
	bytecodeLength := binary.BigEndian.Uint32(payload[4:8])
	// check if the lengths are valid
	if codeAbiLength > uint32(len(payload)) || bytecodeLength > codeAbiLength {
		return nil, "", ""
	}
	// extract the code+abi and deploy args
	codeAbi := payload[4:codeAbiLength]
	deployArgs := payload[4+codeAbiLength:]
	// extract the bytecode and abi
	bytecode := codeAbi[4:bytecodeLength]
	abi := codeAbi[4+bytecodeLength:]
	return bytecode, string(abi), string(deployArgs)
}

func extractSourceCode(payload []byte) (string, string) {
	// read the code length
	codeLength := binary.BigEndian.Uint32(payload[:4])
	// extract the source code and deploy args
	sourceCode := payload[4:codeLength]
	deployArgs := payload[4+codeLength:]
	return string(sourceCode), string(deployArgs)
}

// CompileSourceCode compiles the source code and returns the bytecode and abi
func CompileSourceCode(sourceCode string) ([]byte, string, error) {
	bytecodeABI, err := lua_compiler.CompileCode(sourceCode)
	if err != nil {
		return nil, "", err
	}
	// read the bytecode length
	bytecodeLength := binary.BigEndian.Uint32(bytecodeABI[:4])
	// extract the bytecode and abi
	bytecode := bytecodeABI[4:bytecodeLength]
	abi := bytecodeABI[4+bytecodeLength:]
	return bytecode, string(abi), nil
}

// stores all the internal operations (as a json string) from a transaction
func ConvInternalOperations(txHash string, contract string, jsonOperations string) *EsInternalOperations {
	return &EsInternalOperations{
		BaseEsType: &BaseEsType{Id: txHash},
		Contract:   contract,
		Operations: jsonOperations,
	}
}

// stores each call (internal or external) to a contract
func ConvContractCall(blockNo uint64, timestamp time.Time, txHash string, txIdx uint64, callIdx uint64, caller string, contract string, function string, args string, amount string) *EsContractCall {
	// Create a unique ID using block number, tx index and call index
	id := fmt.Sprintf("%020d-%05d-%04d", blockNo, txIdx, callIdx)

	return &EsContractCall{
		BaseEsType: &BaseEsType{Id: id},
		BlockNo:    blockNo,
		Timestamp:  timestamp,
		TxHash:     txHash,
		IsInternal: callIdx > 0,
		Caller:     caller,
		Contract:   contract,
		Function:   function,
		Args:       args,
		Amount:     amount,
	}
}

// ConvEvent converts Event from RPC into Elasticsearch type
func ConvEvent(event *types.Event, blockDoc *EsBlock, txDoc *EsTx, txIdx uint64) *EsEvent {
	id := fmt.Sprintf("%d-%d-%d", blockDoc.BlockNo, txDoc.TxIdx, event.EventIdx)
	return &EsEvent{
		BaseEsType: &BaseEsType{Id: id},
		Contract:   transaction.EncodeAndResolveAccount(event.ContractAddress, txDoc.BlockNo),
		BlockNo:    blockDoc.BlockNo,
		TxId:       txDoc.Id,
		TxIdx:      txIdx,
		EventIdx:   uint64(event.EventIdx),
		EventName:  event.EventName,
		EventArgs:  event.JsonArgs,
	}
}

func ConvTokenUp(txDoc *EsTx, contractAddress []byte, supply string, supplyFloat float32) *EsTokenUpSupply {
	return &EsTokenUpSupply{
		BaseEsType:  &BaseEsType{Id: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		Supply:      supply,
		SupplyFloat: supplyFloat,
	}
}

func ConvTokenUpVerified(tokenDoc *EsToken, status, tokenAddress, owner, comment, email, regDate, homepageUrl, imageUrl string, totalTransfer uint64) *EsTokenUpVerified {
	return &EsTokenUpVerified{
		BaseEsType:     &BaseEsType{Id: tokenDoc.Id},
		VerifiedStatus: status,
		TokenAddress:   tokenAddress,
		Owner:          owner,
		Comment:        comment,
		Email:          email,
		RegDate:        regDate,
		ImageUrl:       imageUrl,
		HomepageUrl:    homepageUrl,
		TotalTransfer:  totalTransfer,
	}
}

func ConvToken(txDoc *EsTx, contractAddress []byte, tokenType transaction.TokenType, name string, symbol string, decimals uint8, supply string, supplyFloat float32) *EsToken {
	return &EsToken{
		BaseEsType:   &BaseEsType{Id: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		TxId:         txDoc.GetID(),
		BlockNo:      txDoc.BlockNo,
		Creator:      txDoc.Account, // tx account --> token creator
		Type:         tokenType,
		Name:         name,
		Name_lower:   strings.ToLower(name),
		Symbol:       symbol,
		Symbol_lower: strings.ToLower(symbol),
		Decimals:     decimals,
		Supply:       supply,
		SupplyFloat:  supplyFloat,
	}
}

// ConvName parses a name transaction into Elasticsearch type
func ConvName(tx *types.Tx, blockNo uint64) *EsName {
	var name = "error"
	var address string

	payload, err := transaction.UnmarshalPayloadWithArgs(tx)
	if err == nil {
		name = payload.Args[0]
		if payload.Name == "v1createName" {
			address = transaction.EncodeAndResolveAccount(tx.Body.Account, blockNo)
		}
		if payload.Name == "v1updateName" {
			address = payload.Args[1]
		}
	}
	hash := base58.Encode(tx.Hash)

	return &EsName{
		BaseEsType: &BaseEsType{Id: fmt.Sprintf("%s-%s", name, hash)},
		Name:       name,
		Address:    address,
		UpdateTx:   hash,
		BlockNo:    blockNo,
	}
}

func ConvNFT(ttDoc *EsTokenTransfer, tokenUri string, imageUrl string) *EsNFT {
	return &EsNFT{
		BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%s", ttDoc.TokenAddress, ttDoc.TokenId)},
		TokenAddress: ttDoc.TokenAddress,
		TokenId:      ttDoc.TokenId,
		Timestamp:    ttDoc.Timestamp,
		BlockNo:      ttDoc.BlockNo,
		Account:      ttDoc.Amount, // ARC2.tokenTransfer.Amount --> nftDoc.Account (ownerOf)
		TokenUri:     tokenUri,
		ImageUrl:     imageUrl,
	}
}

func ConvTokenTransfer(contractAddress []byte, txDoc *EsTx, idx int, from string, to string, tokenId string, amount string, amountFloat float32) *EsTokenTransfer {
	return &EsTokenTransfer{
		BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%d", txDoc.Id, idx)},
		TxId:         txDoc.GetID(),
		BlockNo:      txDoc.BlockNo,
		Timestamp:    txDoc.Timestamp,
		TokenAddress: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo),
		Sender:       txDoc.Account,
		From:         from,
		To:           to,
		TokenId:      tokenId,
		Amount:       amount,
		AmountFloat:  amountFloat,
	}
}

func ConvAccountTokens(tokenType transaction.TokenType, tokenAddress string, timestamp time.Time, account string, balance string, balanceFloat float32) *EsAccountTokens {
	return &EsAccountTokens{
		BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%s", account, tokenAddress)},
		Account:      account,
		TokenAddress: tokenAddress,
		Timestamp:    timestamp,
		Type:         tokenType,
		Balance:      balance,
		BalanceFloat: balanceFloat,
	}
}

func ConvAccountBalance(blockNo uint64, address string, ts time.Time, balance string, balanceFloat float32, staking string, stakingFloat float32) *EsAccountBalance {
	return &EsAccountBalance{
		BaseEsType:   &BaseEsType{Id: address},
		Timestamp:    ts,
		BlockNo:      blockNo,
		Balance:      balance,
		BalanceFloat: balanceFloat,
		Staking:      staking,
		StakingFloat: stakingFloat,
	}
}

func ConvChainInfo(chainInfo *types.ChainInfo) *EsChainInfo {
	return &EsChainInfo{
		BaseEsType: &BaseEsType{Id: chainInfo.Id.Magic},
		Public:     chainInfo.Id.Public,
		Mainnet:    chainInfo.Id.Mainnet,
		Consensus:  chainInfo.Id.Consensus,
		Version:    uint64(chainInfo.Id.Version),
	}
}

func ConvWhitelist(token string, contract string, whitelistType string) *EsWhitelist {
	return &EsWhitelist{
		BaseEsType: &BaseEsType{Id: token},
		Contract:   contract,
		Type:       whitelistType,
	}
}

// bigIntToFloat takes a big.Int, divides it by 10^exp and returns the resulting float
// Note that this float is not precise. It can be used for sorting purposes
func bigIntToFloat(a *big.Int, exp int64) float32 {
	var y, e = big.NewInt(10), big.NewInt(exp)
	y.Exp(y, e, nil)
	z := new(big.Float).Quo(
		new(big.Float).SetInt(a),
		new(big.Float).SetInt(y),
	)
	f, _ := z.Float32()
	return f
}


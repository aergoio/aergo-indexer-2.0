package documents

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
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
	if receipt != nil {
		status = receipt.Status
		gasUsed = receipt.GasUsed
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
		Payload:       string(tx.GetBody().GetPayload()),
		Account:       transaction.EncodeAndResolveAccount(tx.Body.Account, blockDoc.BlockNo),
		Recipient:     transaction.EncodeAndResolveAccount(tx.Body.Recipient, blockDoc.BlockNo),
		Amount:        amount.String(),
		AmountFloat:   bigIntToFloat(amount, 18),
		Type:          uint64(tx.Body.Type),
		Category:      category,
		Method:        method,
		Status:        status,
		Result:        result,
		Nonce:         nonce,
		FeeDelegation: feeDelegation,
		GasPrice:      gasPrice,
		GasUsed:       gasUsed,
		GasLimit:      tx.Body.GasLimit,
		FeeUsed:       feeUsed,
	}
}

// ConvContractCreateTx creates document for token creation
func ConvContract(txDoc *EsTx, contractAddress []byte) *EsContract {
	return &EsContract{
		BaseEsType: &BaseEsType{Id: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		Creator:    txDoc.Account,
		TxId:       txDoc.GetID(),
		BlockNo:    txDoc.BlockNo,
		Timestamp:  txDoc.Timestamp,
		Payload:    txDoc.Payload,
	}
}

func ConvContractUp(contractAddress string, status, payload, token, codeUrl, code string) *EsContractUp {
	return &EsContractUp{
		BaseEsType:     &BaseEsType{Id: contractAddress},
		Payload:        payload,
		VerifiedToken:  token,
		VerifiedStatus: status,
		CodeUrl:        codeUrl,
		Code:           code,
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

func ConvAccountBalance(blockNo uint64, address []byte, ts time.Time, balance string, balanceFloat float32, staking string, stakingFloat float32) *EsAccountBalance {
	return &EsAccountBalance{
		BaseEsType:   &BaseEsType{Id: transaction.EncodeAndResolveAccount(address, blockNo)},
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

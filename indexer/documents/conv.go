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
func ConvBlock(block *types.Block, blockProducer string) EsBlock {
	rewardAmount := ""
	if len(block.Header.Consensus) > 0 {
		rewardAmount = "160000000000000000"
	}
	return EsBlock{
		BaseEsType:    &BaseEsType{Id: base58.Encode(block.Hash)},
		Timestamp:     time.Unix(0, block.Header.Timestamp),
		BlockNo:       block.Header.BlockNo,
		TxCount:       uint(len(block.Body.Txs)),
		Size:          uint64(proto.Size(block)),
		BlockProducer: blockProducer,
		RewardAccount: transaction.EncodeAndResolveAccount(block.Header.Consensus, block.Header.BlockNo),
		RewardAmount:  rewardAmount,
	}
}

// ConvTx converts Tx from RPC into Elasticsearch type
func ConvTx(tx *types.Tx, blockDoc EsBlock) EsTx {
	amount := big.NewInt(0).SetBytes(tx.GetBody().Amount)
	category, method := transaction.DetectTxCategory(tx)
	if len(method) > 50 {
		method = method[:50]
	}
	return EsTx{
		BaseEsType:     &BaseEsType{Id: base58.Encode(tx.Hash)},
		Account:        transaction.EncodeAndResolveAccount(tx.Body.Account, blockDoc.BlockNo),
		Recipient:      transaction.EncodeAndResolveAccount(tx.Body.Recipient, blockDoc.BlockNo),
		Amount:         amount.String(),
		AmountFloat:    bigIntToFloat(amount, 18),
		Type:           fmt.Sprintf("%d", tx.Body.Type),
		Category:       category,
		Method:         method,
		Timestamp:      blockDoc.Timestamp,
		BlockNo:        blockDoc.BlockNo,
		TokenTransfers: 0,
	}
}

// ConvContractCreateTx creates document for token creation
func ConvContract(txDoc EsTx, contractAddress []byte) EsContract {
	return EsContract{
		BaseEsType: &BaseEsType{Id: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		Creator:    txDoc.Account,
		TxId:       txDoc.GetID(),
		BlockNo:    txDoc.BlockNo,
		Timestamp:  txDoc.Timestamp,
	}
}

// ConvContractCreateTx creates document for token creation
func ConvTokenUp(txDoc EsTx, contractAddress []byte, supply string, supplyFloat float32) EsTokenUp {
	return EsTokenUp{
		BaseEsType:  &BaseEsType{Id: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		Supply:      supply,
		SupplyFloat: supplyFloat,
	}
}

// ConvContractCreateTx creates document for token creation
func ConvToken(txDoc EsTx, contractAddress []byte, tokenType transaction.TokenType, name string, symbol string, decimals uint8, supply string, supplyFloat float32) EsToken {
	return EsToken{
		BaseEsType:     &BaseEsType{Id: transaction.EncodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		TxId:           txDoc.GetID(),
		BlockNo:        txDoc.BlockNo,
		TokenTransfers: uint64(0),
		Type:           tokenType,
		Name:           name,
		Name_lower:     strings.ToLower(name),
		Symbol:         symbol,
		Symbol_lower:   strings.ToLower(symbol),
		Decimals:       decimals,
		Supply:         supply,
		SupplyFloat:    supplyFloat,
	}
}

// ConvName parses a name transaction into Elasticsearch type
func ConvName(tx *types.Tx, blockNo uint64) EsName {
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

	return EsName{
		BaseEsType: &BaseEsType{Id: fmt.Sprintf("%s-%s", name, hash)},
		Name:       name,
		Address:    address,
		UpdateTx:   hash,
		BlockNo:    blockNo,
	}
}

func ConvNFT(contractAddress []byte, ttDoc EsTokenTransfer, account string, tokenUri string, imageUrl string) EsNFT {
	return EsNFT{
		BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%s", ttDoc.TokenAddress, ttDoc.TokenId)},
		TokenAddress: ttDoc.TokenAddress,
		TokenId:      ttDoc.TokenId,
		Timestamp:    ttDoc.Timestamp,
		BlockNo:      ttDoc.BlockNo,
		Account:      account,
		TokenUri:     tokenUri,
		ImageUrl:     imageUrl,
	}
}

func ConvTokenTransfer(contractAddress []byte, txDoc EsTx, idx int, from string, to string, tokenId string, amount string, amountFloat float32) EsTokenTransfer {
	return EsTokenTransfer{
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

func ConvAccountTokens(ttDoc EsTokenTransfer, account string, balance string, balanceFloat float32) EsAccountTokens {
	var tokenType transaction.TokenType // TODO : 외부에서 ARC 여부를 판단하도록 변경
	if ttDoc.TokenId == "" {
		tokenType = transaction.TokenARC1
	} else {
		tokenType = transaction.TokenARC2
	}

	return EsAccountTokens{
		BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%s", account, ttDoc.TokenAddress)},
		Account:      account,
		TokenAddress: ttDoc.TokenAddress,
		Timestamp:    ttDoc.Timestamp,
		Type:         tokenType,
		Balance:      balance,
		BalanceFloat: balanceFloat,
	}
}

func ConvAccountBalance(blockNo uint64, address []byte, ts time.Time, balance string, balanceFloat float32, staking string, stakingFloat float32) EsAccountBalance {
	return EsAccountBalance{
		BaseEsType:   &BaseEsType{Id: transaction.EncodeAndResolveAccount(address, blockNo)},
		Timestamp:    ts,
		BlockNo:      blockNo,
		Balance:      balance,
		BalanceFloat: balanceFloat,
		Staking:      staking,
		StakingFloat: stakingFloat,
	}
}

func ConvChainInfo(chainInfo *types.ChainInfo) EsChainInfo {
	return EsChainInfo{
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

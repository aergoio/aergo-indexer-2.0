package indexer

import (
	"context"
	"fmt"
	"math/big"
//	"strings"
	"strconv"
	"time"
	"encoding/json"
//	"os"

	"github.com/kjunblk/aergo-indexer/indexer/category"
	doc "github.com/kjunblk/aergo-indexer/indexer/documents"
	"github.com/kjunblk/aergo-indexer/indexer/transaction"
	"github.com/kjunblk/aergo-indexer/types"
	"github.com/golang/protobuf/proto"
	"github.com/mr-tron/base58/base58"
)

// ConvBlock converts Block from RPC into Elasticsearch type
func (ns *Indexer) ConvBlock(block *types.Block) doc.EsBlock {
	rewardAmount := ""
	if len(block.Header.Consensus) > 0 {
		rewardAmount = "160000000000000000"
	}
	return doc.EsBlock{
		BaseEsType:    &doc.BaseEsType{base58.Encode(block.Hash)},
		Timestamp:     time.Unix(0, block.Header.Timestamp),
		BlockNo:       block.Header.BlockNo,
		TxCount:       uint(len(block.Body.Txs)),
		Size:          int64(proto.Size(block)),
		RewardAccount: ns.encodeAndResolveAccount(block.Header.Consensus, block.Header.BlockNo),
		RewardAmount:  rewardAmount,
	}
}

// Internal names refer to special accounts that don't need to be resolved
func isInternalName(name string) bool {
	switch name {
	case
		"aergo.name",
		"aergo.system",
		"aergo.enterprise",
		"aergo.vault":
		return true
	}
	return false
}

func encodeAccount(account []byte) string {
	if account == nil {
		return ""
	}
	if len(account) <= 12 || isInternalName(string(account)) {
		return string(account)
	}

	return types.EncodeAddress(account)
}

func (ns *Indexer) encodeAndResolveAccount(account []byte, blockNo uint64) string {

	var encoded = encodeAccount(account)

	//FIXME:seo test
	return encoded

	if len(encoded) > 12 || isInternalName(encoded) || encoded == "" {
		return encoded
	}

	// Resolve name
	var nameRequest = &types.Name{
		Name:    encoded,
		BlockNo: blockNo,
	}

	ctx := context.Background()
	nameInfo, err := ns.grpcClient.GetNameInfo(ctx, nameRequest)

	if err != nil {
		return "UNRESOLVED: " + encoded
	}

	return encodeAccount(nameInfo.GetDestination())
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

// ConvTx converts Tx from RPC into Elasticsearch type
func (ns *Indexer) ConvTx(tx *types.Tx, blockNo uint64) doc.EsTx {

	account := ns.encodeAndResolveAccount(tx.Body.Account, blockNo)
	recipient := ns.encodeAndResolveAccount(tx.Body.Recipient, blockNo)

	amount := big.NewInt(0).SetBytes(tx.GetBody().Amount)
	category, method := category.DetectTxCategory(tx)
	if len(method) > 50 {
		method = method[:50]
	}
	doc := doc.EsTx{
		BaseEsType:     &doc.BaseEsType{base58.Encode(tx.Hash)},
		Account:        account,
		Recipient:      recipient,
		Amount:         amount.String(),
		AmountFloat:    bigIntToFloat(amount, 18),
		Type:           fmt.Sprintf("%d", tx.Body.Type),
		Category:       category,
		Method:         method,
		TokenTransfers: 0,
	}
	return doc
}

// ConvNameTx parses a name transaction into Elasticsearch type
func (ns *Indexer) ConvNameTx(tx *types.Tx, blockNo uint64) doc.EsName {

	var name = "error"
	var address string

	payload, err := transaction.UnmarshalPayloadWithArgs(tx)

	if err == nil {
		name = payload.Args[0]
		if payload.Name == "v1createName" {
			address = ns.encodeAndResolveAccount(tx.Body.Account, blockNo)
		}
		if payload.Name == "v1updateName" {
			address = payload.Args[1]
		}
	}

	hash := base58.Encode(tx.Hash)
	return doc.EsName{
		BaseEsType: &doc.BaseEsType{fmt.Sprintf("%s-%s", name, hash)},
		Name:       name,
		Address:    address,
		UpdateTx:   hash,
	}
}


func (ns *Indexer) ConvAccountTokens(contractAddress []byte, ttDoc doc.EsTokenTransfer, account string) doc.EsAccountTokens {

	document := doc.EsAccountTokens {
		Account:	account,
		TokenAddress:	ttDoc.TokenAddress,
		TokenId:	ttDoc.TokenId,
		BlockNo:	ttDoc.BlockNo,
	}

	if document.TokenId == "" { // ARC1
		document.BaseEsType = &doc.BaseEsType{fmt.Sprintf("%s-%s", document.Account, document.TokenAddress)}
	} else { // ARC2
		document.BaseEsType = &doc.BaseEsType{fmt.Sprintf("%s-%s", document.TokenAddress, document.TokenId)}
	}

	if document.TokenId != "" {
		document.Balance = 0
		return document
	}

	Balance, err := ns.queryContract_Bignum(contractAddress, "balanceOf", document.Account)

	if err != nil {
		document.Balance = 0
		return document
	}

	if AmountFloat, err := strconv.ParseFloat(Balance, 32); err == nil {
		document.Balance = float32(AmountFloat)
	} else {
		document.Balance = 0
	}

//	fmt.Println("---- Balance :", Balance, document.Balance)
//	ns.Stop()

	return document
}


// ConvContractCreateTx creates document for token creation
func (ns *Indexer) ConvTokenTx_mint(contractAddress []byte, txDoc doc.EsTx, idx int, args []interface{}) doc.EsTokenTransfer {

	document := doc.EsTokenTransfer{
		BaseEsType:   &doc.BaseEsType{fmt.Sprintf("%s-%d", txDoc.Id, idx)},
		TxId:         txDoc.GetID(),
		BlockNo:      txDoc.BlockNo,
		Timestamp:    txDoc.Timestamp,
		TokenAddress: ns.encodeAndResolveAccount(contractAddress, txDoc.BlockNo),
		From:         "MINT",
		To:           args[0].(string),
	}

	switch args[1].(type) {
	case string :
		if AmountFloat, err := strconv.ParseFloat(args[1].(string),32); err == nil {
			document.AmountFloat = float32(AmountFloat)
			document.Amount  = args[1].(string)
			document.TokenId = ""
//			document.TokenId = args[1].(string) // for arc2 token have number id
		} else {
			document.TokenId  = args[1].(string)
			document.Amount = "1"
			document.AmountFloat = 1.0
		}
	default : document.Amount = ""
	}

	return document
}

func (ns *Indexer) ConvTokenTx_burn(contractAddress []byte, txDoc doc.EsTx, idx int, args []interface{}) doc.EsTokenTransfer {

	document := doc.EsTokenTransfer{
		BaseEsType:   &doc.BaseEsType{fmt.Sprintf("%s-%d", txDoc.Id, idx)},
		TxId:         txDoc.GetID(),
		BlockNo:      txDoc.BlockNo,
		Timestamp:    txDoc.Timestamp,
		TokenAddress: ns.encodeAndResolveAccount(contractAddress, txDoc.BlockNo),
		From:         args[0].(string),
		To:           "BURN",
	}

	switch args[1].(type) {
	case string :
		if AmountFloat, err := strconv.ParseFloat(args[1].(string),32); err == nil {
			document.AmountFloat = float32(AmountFloat)
			document.Amount  = args[1].(string)
			document.TokenId = ""
//			document.TokenId = args[1].(string) // for arc2 token have number id
		} else {
			document.TokenId  = args[1].(string)
			document.Amount = "1"
			document.AmountFloat = 1.0
		}
	default : document.Amount = ""
	}

	return document
}

func (ns *Indexer) ConvTokenTx_transfer(contractAddress []byte, txDoc doc.EsTx, idx int, args []interface{}) doc.EsTokenTransfer {

	document := doc.EsTokenTransfer{
		BaseEsType:   &doc.BaseEsType{fmt.Sprintf("%s-%d", txDoc.Id, idx)},
		TxId:         txDoc.GetID(),
		BlockNo:      txDoc.BlockNo,
		Timestamp:    txDoc.Timestamp,
		TokenAddress: ns.encodeAndResolveAccount(contractAddress, txDoc.BlockNo),
		From:         args[0].(string),
		To:           args[1].(string),
	}

	switch args[2].(type) {
	case string :
		if AmountFloat, err := strconv.ParseFloat(args[2].(string),32); err == nil {
			document.AmountFloat = float32(AmountFloat)
			document.Amount  = args[2].(string)
			document.TokenId = ""
//			document.TokenId = args[2].(string) // for arc2 token have number id
		} else {
			document.TokenId  = args[2].(string)
			document.Amount = "1"
			document.AmountFloat = 1.0
		}
	default : document.Amount = ""
	}

	return document
}

// ConvContractCreateTx creates document for token creation
func (ns *Indexer) ConvTokenCreateTx(txDoc doc.EsTx, ContractAddress []byte) doc.EsToken {

	document :=  doc.EsToken{
		BaseEsType:  &doc.BaseEsType{ns.encodeAndResolveAccount(ContractAddress, txDoc.BlockNo)},
		TxId:        txDoc.GetID(),
		UpdateBlock: txDoc.BlockNo,
	}

	var err error

	document.Name, err = ns.queryContract(ContractAddress, "name")
	if document.Name == "null" || err != nil {
		document.Name = ""
		return document
	}

	decimals, err := ns.queryContract(ContractAddress, "decimals")
        if err == nil {
                if d, err := strconv.Atoi(decimals); err == nil {
                        document.Decimals = uint8(d)
                }

	} else {
                document.Decimals = uint8(1)
	}

	document.Symbol, err = ns.queryContract(ContractAddress, "symbol")

	return document
}


func (ns *Indexer) queryContract_Bignum(address []byte, name string, args string) (string, error) {

	queryinfo := map[string]interface{}{"Name": name, "Args": []string{args}}

	queryinfoJson, err := json.Marshal(queryinfo)

	if err != nil { return "", err }

	result, err := ns.grpcClient.QueryContract(context.Background(), &types.Query{
		ContractAddress: address,
		Queryinfo:       queryinfoJson,
	})

//	fmt.Println("Query :", queryinfo, result)

	if err != nil { return "", err }

	var ret interface{}

	err = json.Unmarshal([]byte(result.Value), &ret)

	if err != nil {
		return "", err
	}

	switch c := ret.(type) {

	case string:
		return c, nil

	case map[string]interface{}:

		am, ok := convertBignumJson(c)
		if ok {
			return am.String(), nil
		}

	case int:
		return fmt.Sprint(c), nil
	}

	return string(result.Value), nil
}


func (ns *Indexer) queryContract(address []byte, name string) (string, error) {

	queryinfo := map[string]interface{}{"Name": name}

	queryinfoJson, err := json.Marshal(queryinfo)

	if err != nil { return "", err }

	result, err := ns.grpcClient.QueryContract(context.Background(), &types.Query{
		ContractAddress: address,
		Queryinfo:       queryinfoJson,
	})

	if err != nil { return "", err }

	var ret interface{}

	err = json.Unmarshal([]byte(result.Value), &ret)

	if err != nil {
		return "", err
	}

	switch c := ret.(type) {

	case string:
		return c, nil

	case map[string]interface{}:

		am, ok := convertBignumJson(c)
		if ok {
			return am.String(), nil
		}

	case int:
		return fmt.Sprint(c), nil
	}

	return string(result.Value), nil
}

func convertBignumJson(in map[string]interface{}) (*big.Int, bool) {

	bignum, ok := in["_bignum"].(string)
	if ok {
		n := new(big.Int)
		n, ok := n.SetString(bignum, 10)
		if ok {
			return n, true
		}
	}
	return nil, false
}


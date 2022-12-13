package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/types"
	"google.golang.org/grpc"
)

type AergoClientController struct {
	types.AergoRPCServiceClient
}

func NewAergoClient(serverAddr string, ctx context.Context) (*AergoClientController, error) {
	var conn *grpc.ClientConn
	var err error
	maxMsgSize := 1024 * 1024 * 10 // 10mb

	conn, err = grpc.DialContext(ctx, serverAddr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)),
	)
	if err != nil {
		return nil, err
	}

	return &AergoClientController{types.NewAergoRPCServiceClient(conn)}, nil
}

func (t *AergoClientController) QueryBalanceOf(contractAddress []byte, account string, isCccvNft bool) (balance string, balanceFloat float32) {
	var err error
	if isCccvNft == true {
		balance, err = t.queryContract(contractAddress, "query", []string{"balanceOf", account})
	} else {
		balance, err = t.queryContract(contractAddress, "balanceOf", []string{account})
	}
	if err != nil {
		return "0", 0
	}

	if AmountFloat, err := strconv.ParseFloat(balance, 32); err == nil {
		balanceFloat = float32(AmountFloat)
	} else {
		balanceFloat = 0
		balance = "0"
	}
	return balance, balanceFloat
}

func (t *AergoClientController) QueryOwnerOf(contractAddress []byte, args string, isCccvNft bool) (tokenId, amount string, amountFloat float32) {
	var err error
	var owner string
	// 2022/06/05 숫자인 token ID 허용
	if isCccvNft == true {
		owner, err = t.queryContract(contractAddress, "query", []string{"ownerOf", args})
	} else {
		owner, err = t.queryContract(contractAddress, "ownerOf", []string{args})
	}

	if err == nil { // ARC 2
		tokenId = args
		amountFloat = 1.0
		// ARC2.tokenTransfer.Amount --> nft.Account (ownerOf)
		if owner != "" {
			amount = owner
		} else {
			amount = "BURN"
		}
	} else { // ARC 1
		if AmountFloat, err := strconv.ParseFloat(args, 32); err == nil {
			amountFloat = float32(AmountFloat)
			amount = args
			tokenId = ""
		} else {
			amount = ""
		}
	}
	return tokenId, amount, amountFloat
}

func (t *AergoClientController) QueryTotalSupply(contractAddress []byte, isCccvNft bool) (supply string, supplyFloat float32) {
	var err error
	if isCccvNft {
		supply, err = t.queryContract(contractAddress, "query", []string{"totalSupply"})
	} else {
		supply, err = t.queryContract(contractAddress, "totalSupply", nil)
	}
	if err != nil {
		return "0", 0
	}

	if AmountFloat, err := strconv.ParseFloat(supply, 32); err == nil {
		supplyFloat = float32(AmountFloat)
	} else {
		return "0", 0
	}
	return supply, supplyFloat
}

func (t *AergoClientController) QueryTokenInfo(contractAddress []byte) (name, symbol string, decimals uint8) {
	var err error
	name, err = t.queryContract(contractAddress, "name", nil)
	if name == "null" || err != nil {
		return "", "", 0
	}

	symbol, err = t.queryContract(contractAddress, "symbol", nil)
	if symbol == "null" || err != nil {
		return "", "", 0
	}

	strDecimals, err := t.queryContract(contractAddress, "decimals", nil)
	if err == nil {
		if d, err := strconv.Atoi(strDecimals); err == nil {
			decimals = uint8(d)
		}
	} else {
		decimals = uint8(1)
	}

	return name, symbol, decimals
}
func (t *AergoClientController) QueryNFTMetadata(contractAddress []byte, tokenId string) (tokenUri, imageUrl string) {
	var err error
	tokenUri, err = t.queryContract(contractAddress, "get_metadata", []string{tokenId, "token_uri"})
	if tokenUri == "null" || err != nil {
		tokenUri = ""
	}
	imageUrl, err = t.queryContract(contractAddress, "get_metadata", []string{tokenId, "image_url"})
	if imageUrl == "null" || err != nil {
		imageUrl = ""
	}
	return tokenUri, imageUrl
}

func (t *AergoClientController) queryContract(address []byte, name string, args []string) (string, error) {
	queryinfo := map[string]interface{}{"Name": name}
	if args != nil {
		queryinfo["Args"] = args
	}

	queryinfoJson, err := json.Marshal(queryinfo)
	if err != nil {
		return "", err
	}

	result, err := t.QueryContract(context.Background(), &types.Query{
		ContractAddress: address,
		Queryinfo:       queryinfoJson,
	})
	if err != nil {
		return "", err
	}

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

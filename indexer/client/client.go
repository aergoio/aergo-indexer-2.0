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
	client types.AergoRPCServiceClient
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
	} else if conn == nil {
		return nil, fmt.Errorf("failed to connect to server: %s", serverAddr)
	}

	return &AergoClientController{types.NewAergoRPCServiceClient(conn)}, nil
}

func (t *AergoClientController) GetChainInfo() (*types.ChainInfo, error) {
	chaininfo, err := t.client.GetChainInfo(context.Background(), &types.Empty{})
	if err != nil {
		return nil, err
	}
	return chaininfo, nil
}

func (t *AergoClientController) GetBestBlock() (uint64, error) {
	blockchain, err := t.client.Blockchain(context.Background(), &types.Empty{})
	if err != nil {
		return 0, err
	}
	return blockchain.BestHeight, nil
}

func (t *AergoClientController) GetBlock(blockQuery []byte) (*types.Block, error) {
	block, err := t.client.GetBlock(context.Background(), &types.SingleBytes{Value: blockQuery})
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (t *AergoClientController) GetReceipt(txHash []byte) (*types.Receipt, error) {
	receipt, err := t.client.GetReceipt(context.Background(), &types.SingleBytes{Value: txHash})
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (t *AergoClientController) ListBlockStream() (types.AergoRPCService_ListBlockStreamClient, error) {
	stream, err := t.client.ListBlockStream(context.Background(), &types.Empty{})
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (t *AergoClientController) BalanceOf(address []byte) (balance string, balanceFloat float32, staking string, stakingFloat float32) {
	// get unstake balance
	unstakingInfo, err := t.client.GetState(context.Background(), &types.SingleBytes{Value: address})
	if err != nil {
		return "0", 0, "0", 0
	}
	bigUnstaking := big.NewInt(0).SetBytes(unstakingInfo.GetBalance())

	// get stake balance
	stakingInfo, err := t.client.GetStaking(context.Background(), &types.AccountAddress{Value: address})
	if err != nil {
		return "0", 0, "0", 0
	}
	bigStaking := big.NewInt(0).SetBytes(stakingInfo.GetAmount())
	staking = bigStaking.String()

	// make total balance
	bigTotal := big.NewInt(0).Add(bigUnstaking, bigStaking)
	balance = bigTotal.String()

	// make float
	if BalanceFloat, err := strconv.ParseFloat(balance, 32); err == nil {
		balanceFloat = float32(BalanceFloat)
	} else {
		balanceFloat = 0
		balance = "0"
	}
	if StakingFloat, err := strconv.ParseFloat(staking, 32); err == nil {
		stakingFloat = float32(StakingFloat)
	} else {
		stakingFloat = 0
		staking = "0"
	}
	return balance, balanceFloat, staking, stakingFloat
}

func (t *AergoClientController) QueryBalanceOf(contractAddress []byte, account string, isCccvNft bool) (balance string, balanceFloat float32) {
	var err error
	if isCccvNft == true {
		balance, err = t.queryContract(contractAddress, "query", "balanceOf", account)
	} else {
		balance, err = t.queryContract(contractAddress, "balanceOf", account)
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

func (t *AergoClientController) QueryOwnerOf(contractAddress []byte, arg interface{}, isCccvNft bool) (tokenId, amount string, amountFloat float32) {
	var err error
	var owner string

	if argStr, ok := arg.(string); ok {
		// 2022/06/05 숫자인 token ID 허용
		if isCccvNft == true {
			owner, err = t.queryContract(contractAddress, "query", "ownerOf", argStr)
		} else {
			owner, err = t.queryContract(contractAddress, "ownerOf", argStr)
		}
		if err == nil { // ARC 2
			tokenId = argStr
			amountFloat = 1.0
			// ARC2.tokenTransfer.Amount --> nft.Account (ownerOf)
			if owner != "" {
				amount = owner
			} else {
				amount = "BURN"
			}
		} else { // ARC 1
			if AmountFloat, err := strconv.ParseFloat(argStr, 32); err == nil {
				amountFloat = float32(AmountFloat)
				amount = argStr
				tokenId = ""
			} else {
				amount = ""
			}
		}
	} else {
		amount = ""
	}

	return tokenId, amount, amountFloat
}

func (t *AergoClientController) QueryTotalSupply(contractAddress []byte, isCccvNft bool) (supply string, supplyFloat float32) {
	var err error
	if isCccvNft == true {
		supply, err = t.queryContract(contractAddress, "query", "totalSupply")
	} else {
		supply, err = t.queryContract(contractAddress, "totalSupply")
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
	name, err = t.queryContract(contractAddress, "name")
	if name == "null" || err != nil {
		return "", "", 0
	}

	symbol, err = t.queryContract(contractAddress, "symbol")
	if symbol == "null" || err != nil {
		symbol = ""
	}

	strDecimals, err := t.queryContract(contractAddress, "decimals")
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
	tokenUri, err = t.queryContract(contractAddress, "get_metadata", tokenId, "token_uri")
	if tokenUri == "null" || err != nil {
		tokenUri = ""
	}
	imageUrl, err = t.queryContract(contractAddress, "get_metadata", tokenId, "image_url")
	if imageUrl == "null" || err != nil {
		imageUrl = ""
	}
	return tokenUri, imageUrl
}

func (t *AergoClientController) queryContract(address []byte, name string, args ...string) (string, error) {
	queryinfo := map[string]interface{}{"Name": name}
	if args != nil {
		queryinfo["Args"] = args
	}

	queryinfoJson, err := json.Marshal(queryinfo)
	if err != nil {
		return "", err
	}

	result, err := t.client.QueryContract(context.Background(), &types.Query{
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

package client

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/stretchr/testify/require"
)

const (
	// AergoServerAddress = "3.38.108.120:7845"
	AergoServerAddress = "testnet.api-aergo-io:7845" // testnet
)

func TestQuery_BalanceOf(t *testing.T) {
	ctx := context.Background()

	grpcClient, err := NewAergoClient(AergoServerAddress, ctx)
	require.NoError(t, err)
	blockchainStatus, err := grpcClient.client.Blockchain(ctx, &types.Empty{})
	require.NoError(t, err)
	fmt.Println(blockchainStatus.ChainInfo)
	fmt.Println("best chainid hash :", blockchainStatus.BestChainIdHash)

	blockNumber, err := grpcClient.GetBestBlock()
	require.NoError(t, err)

	var blockQuery []byte = make([]byte, 8)
	binary.LittleEndian.PutUint64(blockQuery, blockNumber)

	// fmt.Println("best block chain num or id :", blockInfo.Header.BlockNo, blockInfo.Header.ChainID)
	// rawAddr, _ := types.DecodeAddress("AmPpcKvToDCUkhT1FJjdbNvR4kNDhLFJGHkSqfjWe3QmHm96qv4R")
	// fmt.Println(grpcClient.BalanceOf(rawAddr))
}

func TestQuery_TokenInfo(t *testing.T) {
	grpcClient, err := NewAergoClient(AergoServerAddress, context.Background())
	require.NoError(t, err)

	fn_test := func(contractAddress, nameExpect, symbolExpect string, decimalExpect uint8) {
		contract, _ := types.DecodeAddress(contractAddress)
		name, symbol, decimal := grpcClient.QueryTokenInfo(contract)
		require.Equal(t, nameExpect, name)
		require.Equal(t, symbolExpect, symbol)
		require.Equal(t, decimalExpect, decimal)
	}

	// Blankazzang Point
	fn_test("AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", "Blankazzang Point", "PBLKA", 18)
	// Tname2
	fn_test("AmhnHbTejbuLNzDCKFcnXrTBhXr3eyrt2tqPb8Vn2QCeTEjYx9Xc", "Tname2", "tn2", 18)
	// BOOOST Your Life
	fn_test("AmhWqG43WoT7J7X7cQNChBBLfD4PPGavgGEhAjtEtTuZCUhAGYK7", "BOOOST Your Life", "BYL", 18)
	// Live In Value
	fn_test("AmgnriATUjtsFqsP9sLdxo6xVvYotYeE3Af7EJwc7LjNT6yb5Tzt", "Live In Value", "LIV", 18)
	// Bigdeal Point
	fn_test("Amg7VYQXznevyMjW2S1tdefEjSUCEB3bQvjEXme6rA75wai7L7YP", "Bigdeal Point", "PDEAL1", 18)
	// AERGO v2
	fn_test("Amhpi4LgVS74YJoZAWXsVgkJfEztYe5KkV3tY7sYtCgXchcKQeCQ", "AERGO v2", "ARG", 18)
	// TWILim
	fn_test("AmgV6Z9Pju5u9dShE4KUtFeBEoauq1n9bM96Mm42AsLut1ucGo5u", "TWlLim", "TWL", 18)
	// KSLee Token
	fn_test("Amhk6ZYDPrdx5nTRevvgznPYpH39LaGcPnaK7kqS3E6uSR5GXxBY", "KSLee Token", "PKSLEE", 9)
	// invalid Token ( cccv )
	fn_test("Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF", "", "", 0)
}

func TestQuery_NFTMetadata(t *testing.T) {
	grpcClient, err := NewAergoClient(AergoServerAddress, context.Background())
	require.NoError(t, err)

	fn_test := func(contractAddress, tokenId, tokenUriExpect, imageUrlExpect string) {
		contract, _ := types.DecodeAddress(contractAddress)
		tokenUri, imageUrl := grpcClient.QueryNFTMetadata(contract, tokenId)
		require.Equal(t, tokenUriExpect, tokenUri)
		require.Equal(t, imageUrlExpect, imageUrl)
	}

	// BOOST Vehicle NFT
	fn_test(
		"AmgjGhDQcKWLTtk2ux6ywLftRND1Qd9jD68j3NaFhynwwPcSPDUQ",
		"OSOMDJ0SR6",
		"https://api.booost.live/nft/vehicles/OSOMDJ0SR6",
		"https://booost-nft-prod.s3.ap-northeast-2.amazonaws.com/vehicle-cbt.png?v=2",
	)

	fn_test(
		"AmgjGhDQcKWLTtk2ux6ywLftRND1Qd9jD68j3NaFhynwwPcSPDUQ",
		"RECLZDDCZ3",
		"https://api.booost.live/nft/vehicles/RECLZDDCZ3",
		"https://booost-nft-prod.s3.ap-northeast-2.amazonaws.com/vehicle-cbt.png",
	)

	fn_test(
		"AmhShWWv2qnzSXHmYHrNxb4Eh51UrGsnAfcch3MrsSJ7Acmq3F2M",
		"FQOR48XJ9A",
		"https://dev-api.booost.live/nft/vehicles/FQOR48XJ9A",
		"https://booost-nft-prod.s3.ap-northeast-2.amazonaws.com/vehicle-cbt.png",
	)
}

// TODO: Not able to test Changing Value ( amount, balance, owner, etc. )
/*
	func TestQuery_AccountInfo(t *testing.T) {
		grpcClient, err := NewAergoClient(AergoServerAddress, context.Background())
		require.NoError(t, err)

		blockHeight, _ := grpcClient.GetBestBlock()

		blockQuery := make([]byte, 8)
		binary.LittleEndian.PutUint64(blockQuery, uint64(blockHeight))

		// balance, balanceFloat := grpcClient.client.QueryContract(nil, "AmPUAZw3omgiA8gQHFceFUusfj4M9eVwAtRXQShbQThvF5KDN4Ej", false)
		// fmt.Println(balance, balanceFloat)

		decodedAddr, _ := types.DecodeAddress("AmQLSEi7oeW9LztxGa8pXKQDrenzK2JdDrXsJoCAh6PXyzdBtnVJ")
		state, err := grpcClient.client.GetState(context.Background(), &types.SingleBytes{Value: decodedAddr})
		require.NoError(t, err)

		// grpcClient.client.GetBlockMetadata()
		fmt.Println(state.String())
		bigBalance := big.NewInt(0).SetBytes(state.Balance)
		fmt.Println("unstake :", bigBalance.String())

		staking, err := grpcClient.client.GetStaking(context.Background(), &types.AccountAddress{Value: decodedAddr})
		require.NoError(t, err)

		bigStake := big.NewInt(0).SetBytes(staking.Amount)
		fmt.Println("stake :", bigStake.String())

		fmt.Println("total :", big.NewInt(0).Add(bigBalance, bigStake).String())
	}

	func TestQuery_TotalSupply(t *testing.T) {
		grpcClient, err := NewAergoClient(AergoServerAddress, context.Background())
		require.NoError(t, err)

		fn_test := func(contractAddress string, isCCCVNft bool, supplyExpect string) {
			contract, _ := types.DecodeAddress(contractAddress)
			supply, supplyFloat := grpcClient.QueryTotalSupply(contract, isCCCVNft)
			require.Equal(t, supplyExpect, supply)
			supplyFloatExpect, _ := strconv.ParseFloat(supply, 32)
			require.Equal(t, float32(supplyFloatExpect), supplyFloat)
		}

		// PBLKA
		fn_test("AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", false, "100000000000000000000000000")
		// Tname2
		fn_test("AmhnHbTejbuLNzDCKFcnXrTBhXr3eyrt2tqPb8Vn2QCeTEjYx9Xc", false, "100000000199919999999995000")
		// BOOOST Your Life
		fn_test("AmhWqG43WoT7J7X7cQNChBBLfD4PPGavgGEhAjtEtTuZCUhAGYK7", false, "108012073894999999997342")
		// Live In Value
		fn_test("AmgnriATUjtsFqsP9sLdxo6xVvYotYeE3Af7EJwc7LjNT6yb5Tzt", false, "0")
		// Bigdeal Point
		fn_test("Amg7VYQXznevyMjW2S1tdefEjSUCEB3bQvjEXme6rA75wai7L7YP", false, "1000001000000000000000000000")
		// AERGO v2
		fn_test("Amhpi4LgVS74YJoZAWXsVgkJfEztYe5KkV3tY7sYtCgXchcKQeCQ", false, "1000019970000000000000000000")
		// TWILim
		fn_test("AmgV6Z9Pju5u9dShE4KUtFeBEoauq1n9bM96Mm42AsLut1ucGo5u", false, "100000000000000000000000000")
		// KSLee Token
		fn_test("Amhk6ZYDPrdx5nTRevvgznPYpH39LaGcPnaK7kqS3E6uSR5GXxBY", false, "1000000000000000")
		// BOOOST Vehicle NFT
		fn_test("AmgjGhDQcKWLTtk2ux6ywLftRND1Qd9jD68j3NaFhynwwPcSPDUQ", false, "87")
		// test_nft
		fn_test("AmgL1oYGgNWhrCc337xefNekr5qE7UfyzSNChrgWCobJCNkjWi2S", false, "3")
		// cccv_nft
		fn_test("Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF", true, "0")
		// NWlLim
		fn_test("AmfyF7jvn5t2JEaUJaLFrmrWYSiJWz4fXGYjVFbTZofSWFThj1UQ", false, "40")
	}

	func TestQuery_BalanceOf(t *testing.T) {
		grpcClient, err := NewAergoClient(AergoServerAddress, context.Background())
		require.NoError(t, err)

		fn_test := func(contractAddress string, account string, isCCCVNft bool, balanceExpect string) {
			contract, _ := types.DecodeAddress(contractAddress)
			balance, balanceFloat := grpcClient.QueryBalanceOf(contract, account, isCCCVNft)
			require.Equal(t, balanceExpect, balance)
			balanceFloatExpect, _ := strconv.ParseFloat(balance, 32)
			require.Equal(t, float32(balanceFloatExpect), balanceFloat)
		}

		// Blankazzang Point - holder 1
		fn_test("AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", "AmNBes1nksbz8VhbF6DiXfEqL1dx1YRHFpxZwZABQLqkctmCTFZU", false, "99999669999999777999700000")
		// Blankazzang Point - holder 2
		fn_test("AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", "AmhAn9t6t6m4L6MzZ2Ga8YBZ8qDSJjyqxAgGtxzb6rstZe7g2TVM", false, "329000000222000300020")
		// Blankazzang Point - holder 3
		fn_test("AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", "AmLXrXuyM5Q8FrS6dsyuQeeMNbjrMrDjvUF4YA1hDmK9xHT72Y6f", false, "999999999999999980")

		// BOOOST Your Life - holder 1
		fn_test("AmhWqG43WoT7J7X7cQNChBBLfD4PPGavgGEhAjtEtTuZCUhAGYK7", "AmhF6goUCP3i3ZpCve3kAQZhAB2eiFATSEdKgZxsSbKL8WQYQEpa", false, "69503789058099999997342")
		// BOOOST Your Life - holder 2
		fn_test("AmhWqG43WoT7J7X7cQNChBBLfD4PPGavgGEhAjtEtTuZCUhAGYK7", "AmNa8nfy8KqWyoYMD6cBV8n2Ze2yhpe4nEtP7HBeL8KQYcnFdPg8", false, "10000000000000000000000")
		// BOOOST Your Life - holder 3
		fn_test("AmhWqG43WoT7J7X7cQNChBBLfD4PPGavgGEhAjtEtTuZCUhAGYK7", "AmNGLJYFTk1Y1M4Q4FgmeothYxDvXv5iWLBMXwbvtjN9Xd8Tiyu9", false, "5500000000000000000000")

		// KSLee Token - holder 1
		fn_test("Amhk6ZYDPrdx5nTRevvgznPYpH39LaGcPnaK7kqS3E6uSR5GXxBY", "AmhAn9t6t6m4L6MzZ2Ga8YBZ8qDSJjyqxAgGtxzb6rstZe7g2TVM", false, "993000000000000")
		// KSLee Token - holder 2
		fn_test("Amhk6ZYDPrdx5nTRevvgznPYpH39LaGcPnaK7kqS3E6uSR5GXxBY", "AmNBes1nksbz8VhbF6DiXfEqL1dx1YRHFpxZwZABQLqkctmCTFZU", false, "7000000000000")

		// BOOOST Vehicle NFT - holder 1
		fn_test("AmgjGhDQcKWLTtk2ux6ywLftRND1Qd9jD68j3NaFhynwwPcSPDUQ", "AmPfbSb4UgExeThvYFwJyauTz9fF7qxYPuG1CMiNutky354LurgX", false, "2")
		// BOOOST Vehicle NFT - holder 2
		fn_test("AmgjGhDQcKWLTtk2ux6ywLftRND1Qd9jD68j3NaFhynwwPcSPDUQ", "AmMUfwZpt6BUTRmCkg9CJ1XZQ69eQoZrfpXDHc2gPRVHW26p9YfE", false, "2")
		// BOOOST Vehicle NFT - holder 3
		fn_test("AmgjGhDQcKWLTtk2ux6ywLftRND1Qd9jD68j3NaFhynwwPcSPDUQ", "AmPUuV5TmeS62nN5tF482BJEb8kT293R8fcEkqDKJWLqcv1DBm8h", false, "2")

		// cccv_nft - holder 1
		fn_test("Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF", "AmLhkzqY7Udpe8FoxGxjeJptVaihSQEeLER6r9g7h5xA5ApyDjBv", true, "172169")
		// cccv_nft - holder 2
		fn_test("Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF", "AmNKzmXKcQWRcZabHGToXKMorugoXsqFQVdYHpCySczQ1v7ZXzcU", true, "51547")
		// cccv_nft - holder 3
		fn_test("Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF", "AmPnfW43SdFjuMBuyP7sAePsk7Cnhw7SMz6dHBxBPvxEE2PeDPft", true, "16953")
	}

	func TestQuery_OwnerOf(t *testing.T) {
		grpcClient, err := NewAergoClient(AergoServerAddress, context.Background())
		require.NoError(t, err)

		// args => ??
		fn_test := func(contractAddress string, args string, isCCCVNft bool, tokenIdExpect, amountExpect string) {
			contract, _ := types.DecodeAddress(contractAddress)
			tokenId, amount, amountFloat := grpcClient.QueryOwnerOf(contract, args, isCCCVNft)
			require.Equal(t, tokenId, tokenIdExpect)
			require.Equal(t, amountExpect, amount)
			amountFloatExpect, _ := strconv.ParseFloat(amount, 32)
			require.Equal(t, float32(amountFloatExpect), amountFloat)
		}
		_ = fn_test
	}
*/

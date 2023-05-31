package documents

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/require"
)

func TestConvBlock(t *testing.T) {
	fn_test := func(aergoBlock *types.Block, blockProducer string, esBlockExpect EsBlock) {
		esBlockConv := ConvBlock(aergoBlock, blockProducer)
		require.Equal(t, esBlockExpect, esBlockConv)
	}

	fn_test(&types.Block{
		Hash: []byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu"),
		Header: &types.BlockHeader{
			Timestamp:       1668652376002288214,
			BlockNo:         104524962,
			Consensus:       []byte{48, 69, 2, 33, 0, 132, 143, 216, 185, 150, 194, 108, 165, 179, 18, 240},
			PrevBlockHash:   []byte("9CEiURiJbPpxg3JdsXVZAJLsvhMQfMVCytoPdmiJ1Tga"),
			CoinbaseAccount: []byte("AmPJRLHDKtzLpsaC8ubmPuRkxnMCyBSq5wBwYNDD6DJdgiRhAhYR"),
			PubKey:          []byte{8, 2, 18, 33, 3, 60, 71, 121, 135, 46, 248, 160, 86, 130, 38, 224, 220, 171, 89, 62, 26, 92, 212, 6, 20, 115, 142, 157, 231, 99, 245, 60, 28, 178, 140, 168, 4},
		},
		Body: &types.BlockBody{
			Txs: []*types.Tx{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
		},
	}, "16Uiu2HAmGiJ2QgVAWHMUtzLKKNM5eFUJ3Ds3FN7nYJq1mHN5ZPj9", EsBlock{
		BaseEsType:    &BaseEsType{Id: base58.Encode([]byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu"))},
		Timestamp:     time.Unix(0, 1668652376002288214),
		BlockNo:       104524962,
		Size:          244,
		TxCount:       11,
		BlockProducer: "16Uiu2HAmGiJ2QgVAWHMUtzLKKNM5eFUJ3Ds3FN7nYJq1mHN5ZPj9",
		RewardAccount: "554c66wDnfgGQ2XmBq7Q9jmHuTpNZ",
		RewardAmount:  "160000000000000000",
	})
}

func TestConvTx(t *testing.T) {
	fn_test := func(aergoTx *types.Tx, esBlock EsBlock, esTxExpect EsTx) {
		esTxConv := ConvTx(aergoTx, esBlock)
		require.Equal(t, esTxExpect, esTxConv)
	}

	fn_test(&types.Tx{
		Hash: []byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8"),
		Body: &types.TxBody{
			Account:   []byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"),
			Amount:    big.NewInt(100).Bytes(),
			Type:      types.TxType_TRANSFER,
			Recipient: []byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"),
		},
	}, EsBlock{
		BlockNo:   1,
		Timestamp: time.Unix(0, 1668652376002288214),
	}, EsTx{
		BaseEsType:  &BaseEsType{Id: base58.Encode([]byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8"))},
		Timestamp:   time.Unix(0, 1668652376002288214),
		BlockNo:     1,
		Account:     types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		Recipient:   types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		Amount:      "100",
		AmountFloat: bigIntToFloat(big.NewInt(100), 18),
		Type:        strconv.FormatInt(int64(types.TxType_TRANSFER), 10),
		Category:    transaction.TxTransfer,
	})
}

func TestConvContract(t *testing.T) {
	fn_test := func(esTx EsTx, contractAddress []byte, esContractExpect EsContract) {
		esContractConv := ConvContract(esTx, contractAddress)
		require.Equal(t, esContractExpect, esContractConv)
	}

	fn_test(EsTx{
		BaseEsType: &BaseEsType{Id: base58.Encode([]byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8"))},
		Timestamp:  time.Unix(0, 1668652376002288214),
		BlockNo:    1,
		Account:    types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		Type:       strconv.FormatInt(int64(types.TxType_DEPLOY), 10),
		Category:   transaction.TxDeploy,
	}, []byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"), EsContract{
		TxId:       base58.Encode([]byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8")),
		BaseEsType: &BaseEsType{Id: types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"))},
		Creator:    types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		BlockNo:    1,
		Timestamp:  time.Unix(0, 1668652376002288214),
	})
}

func TestConvTokenUp(t *testing.T) {
	fn_test := func(esTx EsTx, contractAddress string, supply string, supplyFloat float32, esTokenUpExpect EsTokenUp) {
		address, _ := types.DecodeAddress(contractAddress)
		esTokenUpConv := ConvTokenUp(esTx, address, supply, supplyFloat)
		require.Equal(t, esTokenUpExpect, esTokenUpConv)
	}

	fn_test(EsTx{
		BaseEsType: &BaseEsType{Id: base58.Encode([]byte("5Cd2ofFgwFQKSU9H4mDctKLCoQcrcAsY8XXcozCL6a2u"))},
		Timestamp:  time.Unix(0, 1668652376002288214),
		BlockNo:    95022525,
		Type:       strconv.FormatInt(int64(types.TxType_CALL), 10),
		Category:   transaction.TxCall,
	}, "AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", "1", 1, EsTokenUp{
		BaseEsType:  &BaseEsType{Id: "AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks"},
		Supply:      "1",
		SupplyFloat: 1,
	})
	fn_test(EsTx{
		BaseEsType: &BaseEsType{Id: base58.Encode([]byte("5Cd2ofFgwFQKSU9H4mDctKLCoQcrcAsY8XXcozCL6a2u"))},
		Timestamp:  time.Unix(0, 1668652376002288214),
		BlockNo:    95022525,
		Type:       strconv.FormatInt(int64(types.TxType_CALL), 10),
		Category:   transaction.TxCall,
	}, "AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", "100000000000000", 100000000000000, EsTokenUp{
		BaseEsType:  &BaseEsType{Id: "AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks"},
		Supply:      "100000000000000",
		SupplyFloat: 100000000000000,
	})
}

func TestConvToken(t *testing.T) {
	fn_test := func(esTx EsTx, contractAddress string, tokenType transaction.TokenType, name string, symbol string, decimals uint8, supply string, supplyFloat float32, esTokenExpect EsToken) {
		address, _ := types.DecodeAddress(contractAddress)
		esTokenConv := ConvToken(esTx, address, tokenType, name, symbol, decimals, supply, supplyFloat)
		require.Equal(t, esTokenExpect, esTokenConv)
	}

	fn_test(EsTx{
		BaseEsType: &BaseEsType{Id: base58.Encode([]byte("5Cd2ofFgwFQKSU9H4mDctKLCoQcrcAsY8XXcozCL6a2u"))},
		Timestamp:  time.Unix(0, 1668652376002288214),
		BlockNo:    95022525,
		Type:       strconv.FormatInt(int64(types.TxType_CALL), 10),
		Category:   transaction.TxCall,
	}, "AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks", transaction.TokenARC1, "Blankazzang Point", "PBLKA", 18, "100000000", 100000000, EsToken{
		BaseEsType:   &BaseEsType{Id: "AmhUUoFqF4GxjFxxUZrRUieUCRoWnBHT9ESekVAFbif3jU4Zo5ks"},
		TxId:         base58.Encode([]byte("5Cd2ofFgwFQKSU9H4mDctKLCoQcrcAsY8XXcozCL6a2u")),
		BlockNo:      95022525,
		Type:         "ARC1",
		Name:         "Blankazzang Point",
		Name_lower:   "blankazzang point",
		Symbol:       "PBLKA",
		Symbol_lower: "pblka",
		Decimals:     18,
		Supply:       "100000000",
		SupplyFloat:  100000000,
	})
}

func TestConvName(t *testing.T) {
	fn_test := func(aergoTx *types.Tx, blockNumber uint64, esNameExpect EsName) {
		esNameConv := ConvName(aergoTx, blockNumber)
		require.Equal(t, esNameExpect, esNameConv)
	}

	fn_test(&types.Tx{
		Hash: []byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu"),
		Body: &types.TxBody{
			Account: []byte("aergo-account"),
			Amount:  big.NewInt(100).Bytes(),
			Type:    types.TxType_TRANSFER,
			Payload: []byte(`{
				"Name": "v1createName",
				"Args": [
				  "koreanumber1"
				]
			  }`),
		},
	}, 1, EsName{
		BaseEsType: &BaseEsType{Id: fmt.Sprintf("%s-%s", "koreanumber1", base58.Encode([]byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu")))},
		BlockNo:    1,
		Name:       "koreanumber1",
		Address:    "3kgyku6nwyKqHvRQjrpb8Yinv",
		UpdateTx:   base58.Encode([]byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu")),
	})
}

func TestConvNFT(t *testing.T) {
	fn_test := func(contractAddress []byte, esTokenTransfer EsTokenTransfer, amount string, tokenUri string, imageUrl string, esNFTExpect EsNFT) {
		// ARC2.tokenTransfer.Amount --> nft.Account (ownerOf)
		esNFTConv := ConvNFT(contractAddress, esTokenTransfer, amount, tokenUri, imageUrl)
		require.Equal(t, esNFTExpect, esNFTConv)
	}

	fn_test(
		[]byte("Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF"),
		EsTokenTransfer{
			Timestamp:    time.Unix(0, 1668652376002288214),
			BlockNo:      1,
			TokenAddress: types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
			TokenId:      "cccv_nft",
		},
		"1000",
		"https://api.booost.live/nft/vehicles/OSOMDJ0SR6",
		"https://booost-nft-prod.s3.ap-northeast-2.amazonaws.com/vehicle-cbt.png?v=2",
		EsNFT{
			BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%s", types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")), "cccv_nft")},
			TokenAddress: types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
			TokenId:      "cccv_nft",
			Timestamp:    time.Unix(0, 1668652376002288214),
			BlockNo:      1,
			Account:      "1000",
			TokenUri:     "https://api.booost.live/nft/vehicles/OSOMDJ0SR6",
			ImageUrl:     "https://booost-nft-prod.s3.ap-northeast-2.amazonaws.com/vehicle-cbt.png?v=2",
		},
	)
}

func TestConvTokenTransfer(t *testing.T) {
	fn_test := func(contractAddress string, txDoc EsTx, idx int, from string, to string, tokenId string, amount string, amountFloat float32, esTokenTransferExpect EsTokenTransfer) {
		addr, _ := types.DecodeAddress(contractAddress)
		esTokenTransferConv := ConvTokenTransfer(addr, txDoc, idx, from, to, tokenId, amount, amountFloat)
		require.Equal(t, esTokenTransferExpect, esTokenTransferConv)
	}

	fn_test(
		"Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF", EsTx{
			BaseEsType: &BaseEsType{Id: "34yeCGMt2UxFqrztewP2qgJqATQVRdnsu71faJhaWdCA"},
			Timestamp:  time.Unix(0, 1668652376002288214),
			BlockNo:    105810874,
			Type:       strconv.FormatInt(int64(types.TxType_FEEDELEGATION), 10),
			Category:   transaction.TxCall,
			Account:    "AmPEHmsGApC19jtNsvuKrfcruxouAAmVDHg8VK32XamWdcGUmeFA",
		}, 27, "MINT", "AmPEHmsGApC19jtNsvuKrfcruxouAAmVDHg8VK32XamWdcGUmeFA", "a6d6d055488d443d29952c1ca276b34ca_28", "AmPEHmsGApC19jtNsvuKrfcruxouAAmVDHg8VK32XamWdcGUmeFA", 1, EsTokenTransfer{
			BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%d", "34yeCGMt2UxFqrztewP2qgJqATQVRdnsu71faJhaWdCA", 27)},
			Timestamp:    time.Unix(0, 1668652376002288214),
			BlockNo:      105810874,
			TxId:         "34yeCGMt2UxFqrztewP2qgJqATQVRdnsu71faJhaWdCA",
			TokenAddress: "Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF",
			From:         "MINT",
			To:           "AmPEHmsGApC19jtNsvuKrfcruxouAAmVDHg8VK32XamWdcGUmeFA",
			Sender:       "AmPEHmsGApC19jtNsvuKrfcruxouAAmVDHg8VK32XamWdcGUmeFA",
			Amount:       "AmPEHmsGApC19jtNsvuKrfcruxouAAmVDHg8VK32XamWdcGUmeFA",
			AmountFloat:  1,
			TokenId:      "a6d6d055488d443d29952c1ca276b34ca_28",
		},
	)
}

func TestConvAccountTokens(t *testing.T) {
	fn_test := func(tokenId string, tokenAddress string, timestamp time.Time, account string, balance string, balanceFloat float32, esAccountTokensExpect EsAccountTokens) {
		esAccountTokensConv := ConvAccountTokens(tokenId, tokenAddress, timestamp, account, balance, balanceFloat)
		require.Equal(t, esAccountTokensExpect, esAccountTokensConv)
	}

	fn_test(
		"a6d6d055488d443d29952c1ca276b34ca_203",
		"Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF",
		time.Unix(0, 1668652376002288214),
		"AmQLCGCaNqguH9CRuvBLUoYf2dSo77wXeCWyJh5p3mRYqY8o6vZD", "7364", 7364, EsAccountTokens{
			BaseEsType:   &BaseEsType{Id: fmt.Sprintf("%s-%s", "AmQLCGCaNqguH9CRuvBLUoYf2dSo77wXeCWyJh5p3mRYqY8o6vZD", "Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF")},
			Account:      "AmQLCGCaNqguH9CRuvBLUoYf2dSo77wXeCWyJh5p3mRYqY8o6vZD",
			TokenAddress: "Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF",
			Type:         transaction.TokenARC2,
			Timestamp:    time.Unix(0, 1668652376002288214),
			Balance:      "7364",
			BalanceFloat: 7364,
		},
	)
}

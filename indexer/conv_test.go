package indexer

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/category"
	"github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/aergoio/aergo-lib/log"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/require"
)

func TestConvBlock(t *testing.T) {
	indexer := new(Indexer)
	indexer.peerId = make(map[string]string)
	fn_diff := func(aergoBlock *types.Block, esBlockExpect *documents.EsBlock) {
		esBlockConv := indexer.ConvBlock(aergoBlock)
		require.Equal(t, *esBlockExpect, esBlockConv)
	}

	fn_diff(&types.Block{
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
	}, &documents.EsBlock{
		BaseEsType:    &documents.BaseEsType{Id: base58.Encode([]byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu"))},
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
	indexer := new(Indexer)
	fn_test := func(aergoTx *types.Tx, esBlock *documents.EsBlock, esTxExpect *documents.EsTx) {
		esTxConv := indexer.ConvTx(aergoTx, *esBlock)
		require.Equal(t, esTxExpect, &esTxConv)
	}

	fn_test(&types.Tx{
		Hash: []byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8"),
		Body: &types.TxBody{
			Account:   []byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"),
			Amount:    big.NewInt(100).Bytes(),
			Type:      types.TxType_TRANSFER,
			Recipient: []byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"),
		},
	}, &documents.EsBlock{
		BlockNo:   1,
		Timestamp: time.Unix(0, 1668652376002288214),
	}, &documents.EsTx{
		BaseEsType:  &documents.BaseEsType{Id: base58.Encode([]byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8"))},
		Timestamp:   time.Unix(0, 1668652376002288214),
		BlockNo:     1,
		Account:     types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		Recipient:   types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		Amount:      "100",
		AmountFloat: bigIntToFloat(big.NewInt(100), 18),
		Type:        strconv.FormatInt(int64(types.TxType_TRANSFER), 10),
		Category:    category.Transfer,
	})
}

func TestConvName(t *testing.T) {
	indexer := new(Indexer)
	fn_test := func(aergoTx *types.Tx, blockNumber uint64, esNameExpect *documents.EsName) {
		esNameConv := indexer.ConvName(aergoTx, blockNumber)
		require.Equal(t, esNameExpect, &esNameConv)
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
	}, 1, &documents.EsName{
		BaseEsType: &documents.BaseEsType{Id: fmt.Sprintf("%s-%s", "koreanumber1", base58.Encode([]byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu")))},
		BlockNo:    1,
		Name:       "koreanumber1",
		Address:    "3kgyku6nwyKqHvRQjrpb8Yinv",
		UpdateTx:   base58.Encode([]byte("AvtCKTqL3eQBCvkidbY7i4YkwbtbuResohfRKQhV5Bu")),
	})
}

func TestConvContract(t *testing.T) {
	indexer := new(Indexer)
	fn_test := func(esTx *documents.EsTx, contractAddress []byte, esContractExpect *documents.EsContract) {
		esContractConv := indexer.ConvContract(*esTx, contractAddress)
		require.Equal(t, esContractExpect, &esContractConv)
	}

	fn_test(&documents.EsTx{
		BaseEsType: &documents.BaseEsType{Id: base58.Encode([]byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8"))},
		Timestamp:  time.Unix(0, 1668652376002288214),
		BlockNo:    1,
		Account:    types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		Type:       strconv.FormatInt(int64(types.TxType_DEPLOY), 10),
		Category:   category.Deploy,
	}, []byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"), &documents.EsContract{
		TxId:       base58.Encode([]byte("8Zj68cFzrzUtwPe6kZF8qPgVp9LbsefjdTsi4C3hVY8")),
		BaseEsType: &documents.BaseEsType{Id: types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA"))},
		Creator:    types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
		BlockNo:    1,
		Timestamp:  time.Unix(0, 1668652376002288214),
	})
}

func TestConvNFT(t *testing.T) {
	indexer := new(Indexer)
	fn_test := func(contractAddress []byte, esTokenTransfer *documents.EsTokenTransfer, amount string, tokenUri string, esNFTExpect *documents.EsNFT) {
		// ARC2.tokenTransfer.Amount --> nft.Account (ownerOf)
		esNFTConv := indexer.ConvNFT(contractAddress, *esTokenTransfer, amount, tokenUri)
		require.Equal(t, esNFTExpect, &esNFTConv)
	}

	fn_test(
		[]byte("Amg5KQVkBcX1rR1nmKFPyZPnU8CeGWnZkqAiqp3v4fgSL6KmcCuF"),
		&documents.EsTokenTransfer{
			Timestamp:    time.Unix(0, 1668652376002288214),
			BlockNo:      1,
			TokenAddress: types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
			TokenId:      "cccv_nft",
		},
		"1000",
		"https://api.booost.live/nft/vehicles/OSOMDJ0SR6",
		&documents.EsNFT{
			BaseEsType:   &documents.BaseEsType{Id: fmt.Sprintf("%s-%s", types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")), "cccv_nft")},
			TokenAddress: types.EncodeAddress([]byte("AmLc7W3E9kGq9aFshbgBJdss1D8nwbMdjw3ErtJAXwjpBc69VkPA")),
			TokenId:      "cccv_nft",
			Timestamp:    time.Unix(0, 1668652376002288214),
			BlockNo:      1,
			Account:      "1000",
		},
	)
}

// TODO: ConvToken needs to refactor
func TestConvToken(t *testing.T) {

}

// TODO: ConvTokenTransfer needs to refactor
func TestConvTokenTransfer(t *testing.T) {

}

// TODO: ConvAccounTokens needs to refactor
func TestConvAccountTokens(t *testing.T) {

}

func TestQueryContract(t *testing.T) {
	indexer := new(Indexer)
	indexer.log = log.NewLogger("indexer")
	grpcClient := indexer.WaitForClient("testnet-api.aergo.io:7845")
	addr, _ := types.DecodeAddress("AmgjGhDQcKWLTtk2ux6ywLftRND1Qd9jD68j3NaFhynwwPcSPDUQ")
	res, err := indexer.queryContract(addr, "get_metadata", []string{"OSOMDJ0SR6", "token_uri"}, grpcClient)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

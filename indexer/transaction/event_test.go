package transaction

import (
	"testing"

	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalEventNewArcToken(t *testing.T) {
	fn_test := func(jsonArgs string, eventName EventName, expectTokenType TokenType, expectContract string) {
		tokenType, contractAddress, err := UnmarshalEventNewArcToken(&types.Event{
			EventName: string(eventName),
			JsonArgs:  jsonArgs,
		})
		require.NoError(t, err)
		require.Equal(t, expectTokenType, tokenType)
		expectContractDec, _ := types.DecodeAddress(string(expectContract))

		require.Equal(t, expectContractDec, contractAddress)
	}

	fn_test("[\"AmgP4yWTrna4cNsL4MyyMc363dNH6iwymeDKtTNywBSxqRx5j1KT\"]", "new_arc1_token",
		TokenARC1, "AmgP4yWTrna4cNsL4MyyMc363dNH6iwymeDKtTNywBSxqRx5j1KT")
}

func TestUnmarshalEventMint(t *testing.T) {
	fn_test := func(jsonArgs string, expectTo, expectAmountOrId string) {
		_, _, accountTo, amountOrId, err := UnmarshalEventMint(&types.Event{
			EventName: string(EventMint),
			JsonArgs:  jsonArgs,
		})
		require.NoError(t, err)
		require.Equal(t, expectTo, accountTo)
		require.Equal(t, expectAmountOrId, amountOrId)
	}
	fn_test("[\"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU\",\"7281500000000000000\"]",
		"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU", "7281500000000000000")

	fn_test("[null,\"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU\",\"7281500000000000000\"]",
		"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU", "7281500000000000000")

}

func TestUnmarshalEventTransfer(t *testing.T) {
	fn_test := func(jsonArgs string, expectFrom, expectTo, expectAmountOrId string) {
		_, accountFrom, accountTo, amountOrId, err := UnmarshalEventTransfer(&types.Event{
			EventName: string(EventTransfer),
			JsonArgs:  jsonArgs,
		})
		require.NoError(t, err)
		require.Equal(t, expectFrom, accountFrom)
		require.Equal(t, expectTo, accountTo)
		require.Equal(t, expectAmountOrId, amountOrId)
	}

	// normal case
	fn_test("[\"AmhC3FyqabqnYbjhUqwdKm9pNFCWcQZPWEJj6FSqTBZCk7A36ofn\",\"AmMvbMopNXWf78xTAA5BQgN9b2L1UGqqoukVn68xtkd9zPdYbpBV\",\"100000000000000000000000\"]",
		"AmhC3FyqabqnYbjhUqwdKm9pNFCWcQZPWEJj6FSqTBZCk7A36ofn", "AmMvbMopNXWf78xTAA5BQgN9b2L1UGqqoukVn68xtkd9zPdYbpBV", "100000000000000000000000")

	// MINT OR BURN
	fn_test("[\"1111111111111111111111111111111111111111111111111111\",\"1111111111111111111111111111111111111111111111111111\",\"0\"]",
		"MINT", "BURN", "0")

	// except case - big number
	fn_test("[\"1111111111111111111111111111111111111111111111111111\",\"1111111111111111111111111111111111111111111111111111\",{\"_bignum\":\"0\"}]",
		"MINT", "BURN", "0")

	fn_test("[\"AmhC3FyqabqnYbjhUqwdKm9pNFCWcQZPWEJj6FSqTBZCk7A36ofn\",\"AmMvbMopNXWf78xTAA5BQgN9b2L1UGqqoukVn68xtkd9zPdYbpBV\",{\"_bignum\":\"4\"}]",
		"AmhC3FyqabqnYbjhUqwdKm9pNFCWcQZPWEJj6FSqTBZCk7A36ofn", "AmMvbMopNXWf78xTAA5BQgN9b2L1UGqqoukVn68xtkd9zPdYbpBV", "4")

	// except case - null arg at first
	fn_test("[null,\"1111111111111111111111111111111111111111111111111111\",\"1111111111111111111111111111111111111111111111111111\",{\"_bignum\":\"0\"}]",
		"MINT", "BURN", "0")

	// arc2 nft
	fn_test("[\"1111111111111111111111111111111111111111111111111111\",\"AmMDb2KQzsCMUNMW3sQPCpYLgP2SbGrvhgk2wKtwUxETDAd5Hq7U\",\"a9b4e637bd3ba4f98acbb50f98bfd1cf5_1\",\"a9b4e637bd3ba4f98acbb50f98bfd1cf5\"]",
		"MINT", "AmMDb2KQzsCMUNMW3sQPCpYLgP2SbGrvhgk2wKtwUxETDAd5Hq7U", "a9b4e637bd3ba4f98acbb50f98bfd1cf5_1")

	fn_test("[\"AmLuL9c5LwR4QhbJLQZga38xoNKeYxsLJzS2i4neNqNVBNA7oMJY\",\"AmNjkN9fJJcRG94cvDhKFTQqC6S1AjqXzTLf6w5C3DHY3aBiFxC8\",\"abedf4c69f35d46fa8df016418bdb14ef_346\",\"abedf4c69f35d46fa8df016418bdb14ef\",{\"serviceType\":\"cccv\",\"type\":\"gift\",\"productId\":\"qev78oaPFfAk4PN5bq8S\",\"price\":null}]",
		"AmLuL9c5LwR4QhbJLQZga38xoNKeYxsLJzS2i4neNqNVBNA7oMJY", "AmNjkN9fJJcRG94cvDhKFTQqC6S1AjqXzTLf6w5C3DHY3aBiFxC8", "abedf4c69f35d46fa8df016418bdb14ef_346")

}

func TestUnmarshalEventBurn(t *testing.T) {
	fn_test := func(jsonArgs string, expectFrom, expectAmountOrId string) {
		_, accountFrom, _, amountOrId, err := UnmarshalEventBurn(&types.Event{
			EventName: string(EventBurn),
			JsonArgs:  jsonArgs,
		})
		require.NoError(t, err)
		require.Equal(t, expectFrom, accountFrom)
		require.Equal(t, expectAmountOrId, amountOrId)
	}
	fn_test("[\"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU\",\"7281500000000000000\"]",
		"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU", "7281500000000000000")

	fn_test("[null,\"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU\",\"7281500000000000000\"]",
		"AmgZBcmKuMJhSvN9DYi9KV7PF2DcezFvbv87PcCY3GbgpgphCXzU", "7281500000000000000")

}

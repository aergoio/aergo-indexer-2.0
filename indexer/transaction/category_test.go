package transaction

import (
	"math/big"
	"testing"

	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/stretchr/testify/require"
)

func TestDetectTxCategory(t *testing.T) {
	fn_diff := func(Tx *types.Tx, categoryExpect TxCategory, callNameExpect string) {
		category, callName := DetectTxCategory(Tx)
		require.Equal(t, categoryExpect, category)
		require.Equal(t, callNameExpect, callName)
	}

	// Redeploy
	fn_diff(&types.Tx{Body: &types.TxBody{Type: types.TxType_REDEPLOY}}, TxRedeploy, "")

	// MultiCall
	fn_diff(&types.Tx{Body: &types.TxBody{Type: types.TxType_MULTICALL}}, TxMultiCall, "")

	// Deploy
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_NORMAL,
			Recipient: nil,
			Payload:   []byte{1, 2, 3, 4}, // deploy contract bytecode
		},
	}, TxDeploy, "")

	// Cluster
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.enterprise"),
			Payload: []byte(`{
				"Name": "changeCluster",
				"Args": [
				  {
					"command": "remove",
					"id": "ee72676e83929233"
				  }
				]
			}`),
		},
	}, TxCluster, "changecluster")

	// Conf
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.enterprise"),
			Payload: []byte(`{
				"Name": "changeConf",
				"Args": [
				  {
					"command": "remove",
					"id": "ee72676e83929233"
				  }
				]
			}`),
		},
	}, TxConf, "changeconf")

	// Enterprise
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.enterprise"),
			Payload: []byte(`{
				"Name": "appendAdmin",
				"Args": [
				  "AmLjqiQiMiG44oc9aucS9XTxEdHfcMesLVRWS5VZtuK9Vo4ddecF"
				]
			}`),
		},
	}, TxEnterprise, "appendadmin")

	// NameCreate
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.name"),
			Payload: []byte(`{
				"Name": "v1createName",
				"Args": [
				"abcdefg12345"
				]
			}`),
		},
	}, TxNameCreate, "v1createname")

	// NameUpdate
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.name"),
			Payload: []byte(`{
				"Name": "v1updateName",
				"Args": [
				  "tokenlockerr",
				  "AmhXTtDUv7ZCJHB8Bz29S5F8pEj5F6gmb6wGLzUn7ADwergXNJAc"
				]
			}`),
		},
	}, TxNameUpdate, "v1updatename")

	// Name
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.name"),
			Payload: []byte(`{
				"Name": "v1setOwner",
				"Args": [
				  "AmhJX1FYQKqNuhwBACW7fkxcTHdMakAfMQobuD5QNXJwT1ZgriAc"
				]
			}`),
		},
	}, TxName, "v1setowner")

	// Staking
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.system"),
			Payload: []byte(`{
				"Name": "v1stake"
			}`),
		},
	}, TxStaking, "v1stake")

	// Voting
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_GOVERNANCE,
			Recipient: []byte("aergo.system"),
			Payload: []byte(`{
				"Name": "v1voteBP",
				"Args": [
				  "16Uiu2HAmGiJ2QgVAWHMUtzLKKNM5eFUJ3Ds3FN7nYJq1mHN5ZPj9"
				]
			}`),
		},
	}, TxVoting, "v1votebp")

	// System ( not exist now )

	// Governance
	fn_diff(&types.Tx{Body: &types.TxBody{Type: types.TxType_GOVERNANCE}}, TxGovernance, "")

	// Call
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_CALL,
			Recipient: []byte{12, 171, 69, 173, 86, 155, 29, 39, 247, 12, 159, 32, 253, 97, 50, 76, 129, 108, 21, 82, 227, 57, 171, 87, 153, 60, 50, 199, 126, 40, 150, 147, 124},
			Payload: []byte(`{
				"Name": "invoke",
				"Args": [
				"claimBylTokenReward",
				"AmPDbCi6D5EatLGco42x4zyqFV88f4EiV8Ge3TvEAN7yHYEUBeCW",
				"5859000000000000"
				]
			}`),
		},
	}, TxCall, "invoke")

	// Payload
	fn_diff(&types.Tx{
		Body: &types.TxBody{
			Type:      types.TxType_TRANSFER,
			Recipient: []byte{12, 171, 69, 173, 86, 155, 29, 39, 247, 12, 159, 32, 253, 97, 50, 76, 129, 108, 21, 82, 227, 57, 171, 87, 153, 60, 50, 199, 126, 40, 150, 147, 124},
			Payload: []byte(`{
				"chainID": "CHHjUaP3Euuxb4hC63j6otitzLfieFimqBxdhMt3NSE2",
				"bestBlockHeight": 27421809,
				"bestBlockHash": "6k8WnEKMjm7ofBP3wtQFZn3DxAD97FzxZpayAHPX9kyq"
			}`),
		},
	}, TxPayload, "")

	// Transfer
	fn_diff(&types.Tx{Body: &types.TxBody{Type: types.TxType_TRANSFER, Amount: big.NewInt(100).Bytes()}}, TxTransfer, "")
	fn_diff(&types.Tx{Body: &types.TxBody{Type: types.TxType_NORMAL, Amount: big.NewInt(100).Bytes()}}, TxTransfer, "")

	// None
	fn_diff(&types.Tx{Body: &types.TxBody{Type: types.TxType_NORMAL}}, TxNone, "")
}

package transaction

import (
	"strings"

	"github.com/aergoio/aergo-indexer-2.0/types"
)

// TxCategory is a user-friendly categorization of a transaction
type TxCategory string

// Categories
const (
	TxNone       TxCategory = ""
	TxPayload    TxCategory = "payload"
	TxCall       TxCategory = "call"
	TxGovernance TxCategory = "governance"
	TxSystem     TxCategory = "system"
	TxStaking    TxCategory = "staking"
	TxVoting     TxCategory = "voting"
	TxName       TxCategory = "name"
	TxNameCreate TxCategory = "namecreate"
	TxNameUpdate TxCategory = "nameupdate"
	TxEnterprise TxCategory = "enterprise"
	TxConf       TxCategory = "conf"
	TxCluster    TxCategory = "cluster"
	TxDeploy     TxCategory = "deploy"
	TxRedeploy   TxCategory = "redeploy"
	TxMultiCall  TxCategory = "multicall"
	TxTransfer   TxCategory = "transfer"
)

// TxCategories is the list of available categories in order of increasing weight
var TxCategories = []TxCategory{TxNone, TxPayload, TxCall, TxGovernance, TxSystem, TxStaking, TxVoting, TxName, TxNameCreate, TxNameUpdate, TxEnterprise, TxConf, TxCluster, TxDeploy, TxRedeploy, TxMultiCall}

// DetectTxCategory by performing a cascade of checks with fallbacks
func DetectTxCategory(tx *types.Tx) (TxCategory, string) {
	txBody := tx.GetBody()
	txType := txBody.GetType()
	txRecipient := string(txBody.GetRecipient())

	if txType == types.TxType_REDEPLOY {
		return TxRedeploy, ""
	}

	if txType == types.TxType_MULTICALL {
		return TxMultiCall, ""
	}

	if txRecipient == "" && len(txBody.Payload) > 0 {
		return TxDeploy, ""
	}

	if txRecipient == "aergo.enterprise" {
		txCallName, err := GetCallName(tx)
		if err == nil {
			txCallName = strings.ToLower(txCallName)
			if strings.HasSuffix(txCallName, "cluster") {
				return TxCluster, txCallName
			}
			if strings.HasSuffix(txCallName, "conf") {
				return TxConf, txCallName
			}
			return TxEnterprise, txCallName
		}
		return TxEnterprise, ""
	}

	if txRecipient == "aergo.name" {
		txCallName, err := GetCallName(tx)
		if err == nil {
			txCallName = strings.ToLower(txCallName)
			if strings.HasSuffix(txCallName, "updatename") {
				return TxNameUpdate, txCallName
			}
			if strings.HasSuffix(txCallName, "createname") {
				return TxNameCreate, txCallName
			}
			return TxName, txCallName
		}
		return TxName, ""
	}

	if txRecipient == "aergo.system" {
		txCallName, err := GetCallName(tx)
		if err == nil {
			txCallName = strings.ToLower(txCallName)
			if strings.HasSuffix(txCallName, "stake") || strings.HasSuffix(txCallName, "unstake") {
				return TxStaking, txCallName
			}
			if strings.HasSuffix(txCallName, "vote") || strings.HasSuffix(txCallName, "votedao") || strings.HasSuffix(txCallName, "votebp") || strings.HasSuffix(txCallName, "proposal") {
				return TxVoting, txCallName
			}
			return TxSystem, txCallName
		}
		return TxSystem, ""
	}

	if txType == types.TxType_GOVERNANCE {
		return TxGovernance, ""
	}

	txCallName, err := GetCallName(tx)
	if err == nil && txCallName != "" {
		return TxCall, txCallName
	}

	if len(txBody.Payload) > 0 {
		return TxPayload, ""
	}

	if txType == types.TxType_TRANSFER {
		return TxTransfer, ""
	}

	if txType == types.TxType_NORMAL && len(tx.Body.Amount) > 0 && string(tx.Body.Amount) != "0" {
		return TxTransfer, ""
	}

	return TxNone, ""
}

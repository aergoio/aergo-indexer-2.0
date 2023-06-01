package transaction

import (
	"strings"

	"github.com/aergoio/aergo-indexer-2.0/types"
)

// TokenType
type TokenType string

// Categories
const (
	TokenNone TokenType = ""
	TokenARC1 TokenType = "ARC1"
	TokenARC2 TokenType = "ARC2"
)

// MaybeTokenCreation runs a heuristic to determine if tx might be creating a token
func MaybeTokenCreation(tx *types.Tx) TokenType {
	txBody := tx.GetBody()

	// We treat the payload (which is part bytecode, part ABI) as text
	// and check that ALL the ARC1/2 keywords are included
	if !(txBody.GetType() == types.TxType_DEPLOY && len(txBody.Payload) > 30) {
		return TokenNone
	}

	payload := string(txBody.GetPayload())

	keywords := [...]string{"name", "balanceOf", "transfer", "symbol", "totalSupply"}
	for _, keyword := range keywords {
		if !strings.Contains(payload, keyword) {
			return TokenNone
		}
	}

	include := true
	keywords1 := [...]string{"decimals"}
	for _, keyword := range keywords1 {
		if !strings.Contains(payload, keyword) {
			include = false
			break
		}
	}
	if include {
		return TokenARC1
	}

	include = true
	keywords2 := [...]string{"ownerOf"}
	for _, keyword := range keywords2 {
		if !strings.Contains(payload, keyword) {
			include = false
			break
		}
	}
	if include {
		return TokenARC2
	}
	return TokenNone
}

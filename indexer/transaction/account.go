package transaction

import (
	"unicode"

	"github.com/aergoio/aergo-indexer-2.0/types"
)

// Aergo system refer to special accounts that don't need to be resolved
func IsAergoSystem(address string) bool {
	return address == "aergo.system"
}

// Alias refer to special accounts that don't need to be resolved
func IsAlias(address string) bool {
	if len(address) != 12 {
		return false
	}
	for _, c := range address {
		if !unicode.IsUpper(c) && !unicode.IsLower(c) && !unicode.IsNumber(c) {
			return false
		}
	}
	return true
}

// Internal names refer to special accounts that don't need to be resolved
func IsInternalName(name string) bool {
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

// Refer to special accounts that don't need to be resolved - alias or aergo.system
func IsBalanceNotResolved(name string) bool {
	return IsAlias(name) || IsAergoSystem(name)
}

func DecodeAccount(account string) []byte {
	if account == "" {
		return nil
	}
	if IsAlias(account) || IsInternalName(account) {
		return []byte(account)
	}
	dec, _ := types.DecodeAddress(account)
	return dec
}

func EncodeAccount(account []byte) string {
	if account == nil {
		return ""
	}
	if IsAlias(string(account)) || IsInternalName(string(account)) {
		return string(account)
	}
	return types.EncodeAddress(account)
}

func EncodeAndResolveAccount(account []byte, blockNo uint64) string {
	var encoded = EncodeAccount(account)
	return encoded
}

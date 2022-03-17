package types

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/anaskhan96/base58check"
)

type Address = []byte

const AddressVersion = 0x42
const PrivKeyVersion = 0xAA

func EncodeAddress(addr Address) string {
	encoded, _ := base58check.Encode(fmt.Sprintf("%x", AddressVersion), hex.EncodeToString(addr))
	return encoded
}

func DecodeAddress(encodedAddr string) (Address, error) {
	decodedString, err := base58check.Decode(encodedAddr)
	if err != nil {
		return nil, err
	}
	decodedBytes, err := hex.DecodeString(decodedString)
	if err != nil {
		return nil, err
	}
	version := decodedBytes[0]
	if version != AddressVersion {
		return nil, errors.New("Invalid address version")
	}
	decoded := decodedBytes[1:]
	return decoded, nil
}

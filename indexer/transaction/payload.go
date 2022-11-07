package transaction

import (
	"encoding/json"

	"github.com/aergoio/aergo-indexer-2.0/types"
)

// Payload is an unmarshalled contract call payload, but only supports string arguments
type Payload struct {
	Name string   `json:"Name"`
	Args []string `json:"Args"`
}

// PayloadBasic is an unmarshalled contract call without the arguments
type PayloadBasic struct {
	Name string `json:"Name"`
}

// UnmarshalPayload concerts payload bytes into a struct using json
func UnmarshalPayload(tx *types.Tx) (*PayloadBasic, error) {
	payloadSource := tx.GetBody().GetPayload()
	payload := new(PayloadBasic)
	err := json.Unmarshal(payloadSource, payload)
	if err != nil {
		return &PayloadBasic{}, err
	}
	return payload, nil
}

// UnmarshalPayloadWithArgs converts payload bytes into a struct using json. support string arguments
func UnmarshalPayloadWithArgs(tx *types.Tx) (*Payload, error) {
	payloadSource := tx.GetBody().GetPayload()
	payload := new(Payload)
	err := json.Unmarshal(payloadSource, payload)
	if err != nil {
		return &Payload{}, err
	}
	return payload, nil
}

// GetCallName extracts the contract call function name from the payload
func GetCallName(tx *types.Tx) (string, error) {
	payload, err := UnmarshalPayload(tx)
	if err != nil {
		return "", err
	}
	return payload.Name, nil
}

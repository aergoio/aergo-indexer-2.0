package transaction

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aergoio/aergo-indexer-2.0/types"
)

// EventName
type EventName string

// Categories
const (
	EventNone         EventName = ""
	EventNewArc1Token EventName = "new_arc1_token"
	EventNewArc2Token EventName = "new_arc2_token"
	EventMint         EventName = "mint"
	EventTransfer     EventName = "transfer"
	EventBurn         EventName = "burn"
)

func UnmarshalEventNewArcToken(event *types.Event) (tokenType TokenType, contractAddress []byte, err error) {
	if event == nil {
		return TokenNone, nil, errors.New("not exist event")
	}

	// get token type
	switch EventName(event.EventName) {
	case EventNewArc1Token:
		tokenType = TokenARC1
	case EventNewArc2Token:
		tokenType = TokenARC2
	default:
		return TokenNone, nil, fmt.Errorf("invalid event name | %s", event.EventName)
	}

	// parse event args
	var args []interface{}
	err = json.Unmarshal([]byte(event.JsonArgs), &args)
	if err != nil {
		return TokenNone, nil, fmt.Errorf("invalid event args | %s", event.JsonArgs)
	} else if len(args) < 1 || args[0] == nil {
		return TokenNone, nil, fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	// get contract address
	contractAddr, ok := args[0].(string)
	if !ok {
		return TokenNone, nil, fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}
	contractAddress, err = types.DecodeAddress(contractAddr)
	if err != nil {
		return TokenNone, nil, fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	return tokenType, contractAddress, nil
}

func UnmarshalEventMint(event *types.Event) (contractAddress []byte, accountFrom, accountTo, amountOrId string, err error) {
	if event == nil {
		return nil, "", "", "", errors.New("not exist event")
	}
	if EventName(event.EventName) != EventMint {
		return nil, "", "", "", fmt.Errorf("invalid event name | %s", event.EventName)
	}

	// get contract address
	contractAddress = event.GetContractAddress()

	var args []interface{}
	err = json.Unmarshal([]byte(event.JsonArgs), &args)
	if err != nil {
		return nil, "", "", "", err
	} else if len(args) < 2 || args[0] == nil || args[1] == nil {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	// get account from
	accountFrom = "MINT"

	// get account to
	accountTo, ok := args[0].(string)
	if !ok {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}
	// get amount or id
	amountOrId, ok = args[1].(string)
	if !ok {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	return contractAddress, accountFrom, accountTo, amountOrId, nil
}

func UnmarshalEventTransfer(event *types.Event) (contractAddress []byte, accountFrom, accountTo, amountOrId string, err error) {
	if event == nil {
		return nil, "", "", "", errors.New("not exist event")
	}
	if EventName(event.EventName) != EventTransfer {
		return nil, "", "", "", fmt.Errorf("invalid event name | %s", event.EventName)
	}

	// get contract address
	contractAddress = event.GetContractAddress()

	var args []interface{}
	err = json.Unmarshal([]byte(event.JsonArgs), &args)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	} else if args[0] == nil || len(args) < 3 {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}
	// get account from
	accountFrom, ok := args[0].(string)
	if !ok {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	} else if accountFrom == "1111111111111111111111111" {
		accountFrom = "MINT"
	}

	// get account to
	accountTo, ok = args[1].(string)
	if !ok {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	} else if accountTo == "1111111111111111111111111" {
		accountTo = "BURN"
	}

	// get amount or id
	amountOrId, ok = args[2].(string)
	if !ok {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	return contractAddress, accountFrom, accountTo, amountOrId, nil
}

func UnmarshalEventBurn(event *types.Event) (contractAddress []byte, accountFrom, accountTo, amountOrId string, err error) {
	if event == nil {
		return nil, "", "", "", errors.New("not exist event")
	}
	if EventName(event.EventName) != EventBurn {
		return nil, "", "", "", fmt.Errorf("invalid event name | %s", event.EventName)
	}

	// get contract address
	contractAddress = event.GetContractAddress()

	var args []interface{}
	err = json.Unmarshal([]byte(event.JsonArgs), &args)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	} else if len(args) < 2 || args[0] == nil || args[1] == nil {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	// get account from
	accountFrom, ok := args[0].(string)
	if !ok {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	// get account to
	accountTo = "BURN"

	// get amount or id
	amountOrId, ok = args[1].(string)
	if !ok {
		return nil, "", "", "", fmt.Errorf("invalid event args | %s", event.JsonArgs)
	}

	return contractAddress, accountFrom, accountTo, amountOrId, nil
}

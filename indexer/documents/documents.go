package documents

import (
	"fmt"
	"strings"
	"time"

	"github.com/kjunblk/aergo-indexer-2.0/indexer/category"
)

// DocType is an interface for structs to be used as database documents
type DocType interface {
	GetID() string
	SetID(string)
}

// BaseEsType implements DocType and contains the document's id
type BaseEsType struct {
	Id string `json:"-" db:"id"`
}

// GetID returns the document's id
func (m BaseEsType) GetID() string {
	return m.Id
}

// SetID sets the document's id
func (m BaseEsType) SetID(id string) {
	m.Id = id
}

// EsBlock is a block stored in the database
type EsBlock struct {
	*BaseEsType
	Timestamp     time.Time `json:"ts" db:"ts"`
	BlockNo       uint64    `json:"no" db:"no"`
	TxCount       uint      `json:"txs" db:"txs"`
	Size          int64     `json:"size" db:"size"`
	RewardAccount string    `json:"reward_account" db:"reward_account"`
	RewardAmount  string    `json:"reward_amount" db:"reward_amount"`
}

// EsTx is a transaction stored in the database
type EsTx struct {
	*BaseEsType
	Timestamp      time.Time           `json:"ts" db:"ts"`
	BlockNo        uint64              `json:"blockno" db:"blockno"`
	Account        string              `json:"from" db:"from"`
	Recipient      string              `json:"to" db:"to"`
	Amount         string              `json:"amount" db:"amount"`             // string of BigInt
	AmountFloat    float32             `json:"amount_float" db:"amount_float"` // float for sorting
	Type           string              `json:"type" db:"type"`
	Category       category.TxCategory `json:"category" db:"category"`
	Method         string              `json:"method" db:"method"`
	TokenTransfers int                 `json:"token_transfers" db:"token_transfers"`
}

// EsName is a name-address mapping stored in the database
type EsName struct {
	*BaseEsType
	Name		string `json:"name" db:"name"`
	Address		string `json:"address" db:"address"`
	BlockNo		uint64 `json:"blockno" db:"blockno"`
	UpdateTx	string `json:"tx" db:"tx"`
}

// EsTokenTransfer is a transfer of a token
type EsTokenTransfer struct {
	*BaseEsType
	TxId         string    `json:"tx_id" db:"tx_id"`
	Timestamp    time.Time `json:"ts" db:"ts"`
	BlockNo      uint64    `json:"blockno" db:"blockno"`
	TokenAddress string    `json:"address" db:"address"`
	From         string    `json:"from" db:"from"`
	To           string    `json:"to" db:"to"`
	Sender       string    `json:"sender" db:"sender"`
	Amount       string    `json:"amount" db:"amount"`             // string of BigInt
	AmountFloat  float32   `json:"amount_float" db:"amount_float"` // float for sorting
	TokenId      string    `json:"token_id" db:"token_id"`
}

// EsToken is meta data of a token. The id is the contract address.
type EsToken struct {
	*BaseEsType
	TxId		string		`json:"tx_id" db:"tx_id"`
	BlockNo		uint64		`json:"blockno" db:"blockno"`
	Type		category.TokenType	`json:"type" db:"type"`
	Name		string		`json:"name" db:"name"`
	Symbol		string		`json:"symbol" db:"symbol"`
	TokenTransfers	uint64		`json:"token_transfers" db:"token_transfers"`
	Decimals	uint8           `json:"decimals" db:"decimals"`
	Supply		string          `json:"supply" db:"supply"`
	SupplyFloat	float32         `json:"supply_float" db:"supply_float"`
}

type EsTokenUp struct {
	*BaseEsType
	Supply		string          `json:"supply" db:"supply"`
	SupplyFloat	float32         `json:"supply_float" db:"supply_float"`
}

// EsAccountTokens is meta data of a token of an account. The id is account_token address.
type EsAccountTokens struct {
	*BaseEsType
	Account		string	`json:"account" db:"account"`
	TokenAddress	string	`json:"address" db:"address"`
	Type		category.TokenType `json:"type" db:"type"`
	Timestamp    	time.Time `json:"ts" db:"ts"`
	Balance		string	`json:"balance" db:"balance"`
	BalanceFloat    float32	`json:"balance_float" db:"balance_float"`
}

type EsAccountTokensUp struct {
	*BaseEsType
	Timestamp    time.Time `json:"ts" db:"ts"`
	Balance		string	`json:"balance" db:"balance"`
	BalanceFloat    float32	`json:"balance_float" db:"balance_float"`
}

type EsNFT struct {
	*BaseEsType
	TokenAddress string	`json:"address" db:"address"`
	TokenId     string	`json:"token_id" db:"token_id"`
	Account     string	`json:"account" db:"account"`
	BlockNo      uint64    `json:"blockno" db:"blockno"`
	Timestamp	time.Time `json:"ts" db:"ts"`
}

type EsNFTUp struct {
	*BaseEsType
	Account     string	`json:"account" db:"account"`
	BlockNo      uint64    `json:"blockno" db:"blockno"`
	Timestamp	time.Time `json:"ts" db:"ts"`
}

var EsMappings = map[string]string{
	"tx": `{
		"settings" : {
			"number_of_shards" : 20,
			"number_of_replicas" : 0
		},
		"mappings":{
			"properties":{
				"ts": {
					"type": "date"
				},
				"blockno": {
					"type": "long"
				},
				"from": {
					"type": "keyword"
				},
				"to": {
					"type": "keyword"
				},
				"amount": {
					"enabled": false
				},
				"amount_float": {
					"type": "float"
				},
				"type": {
					"type": "keyword"
				},
				"category": {
					"type": "keyword"
				},
				"method": {
					"type": "keyword"
				},
				"token_transfers": {
					"type": "long"
				}
			}
		}
	}`,
	"block": `{
		"settings" : {
			"number_of_shards" : 100,
			"number_of_replicas" : 0
		},
		"mappings":{
			"properties": {
				"ts": {
					"type": "date"
				},
				"no": {
					"type": "long"
				},
				"txs": {
					"type": "long"
				},
				"size": {
					"type": "long"
				},
				"reward_account": {
					"type": "keyword"
				},
				"reward_amount": {
					"type": "float"
				}
			}
		}
	}`,
	"name": `{
		"settings" : {
			"number_of_shards" : 2,
			"number_of_replicas" : 0
		},
		"mappings":{
			"properties": {
				"name": {
					"type": "keyword"
				},
				"address": {
					"type": "keyword"
				},
				"blockno": {
					"type": "long"
				},
				"tx": {
					"type": "keyword"
				}
			}
		}
	}`,
	"token_transfer": `{
		"settings" : {
			"number_of_shards" : 50,
			"number_of_replicas" : 0
		},
		"mappings":{
			"properties":{
				"tx_id": {
					"type": "keyword"
				},
				"blockno": {
					"type": "long"
				},
				"ts": {
					"type": "date"
				},
				"address": {
					"type": "keyword"
				},
				"token_id": {
					"type": "keyword"
				},
				"from": {
					"type": "keyword"
				},
				"to": {
					"type": "keyword"
				},
				"sender": {
					"type": "keyword"
				},
				"amount": {
					"enabled": false
				},
				"amount_float": {
					"type": "float"
				}
			}
		}
	}`,
	"token": `{
		"settings" : {
			"number_of_shards" : 5,
			"number_of_replicas" : 0
		},
		"mappings":{
			"properties":{
				"tx_id": {
					"type": "keyword"
				},
				"blockno": {
					"type": "long"
				},
				"name": {
					"type": "keyword"
				},
				"symbol": {
					"type": "keyword"
				},
				"decimals": {
					"type": "short"
				},
				"supply": {
					"enabled": false
				},
				"supply_float": {
					"type": "float"
				},
				"token_transfers": {
					"type": "long"
				},
				"type": {
					"type": "keyword"
				}
			}
		}
	}`,
	"account_tokens": `{
		"settings" : {
			"number_of_shards" : 10,
			"number_of_replicas" : 0
		},
		"mappings":{
			"properties":{
				"account": {
					"type": "keyword"
				},
				"address": {
					"type": "keyword"
				},
				"type": {
					"type": "keyword"
				},
				"ts": {
					"type": "date"
				},
				"balance": {
					"enabled": false
				},
				"balance_float": {
					"type": "float"
				}
			}
		}
	}`,
	"nft": `{
		"settings" : {
			"number_of_shards" : 50,
			"number_of_replicas" : 0
		},
		"mappings":{
			"properties":{
				"address": {
					"type": "keyword"
				},
				"token_id": {
					"type": "keyword"
				},
				"blockno": {
					"type": "long"
				},
				"ts": {
					"type": "date"
				},
				"account": {
					"type": "keyword"
				}
			}
		}
	}`,
}


func mapCategoriesToStr(categories []category.TxCategory) []string {
	vsm := make([]string, len(categories))
	for i, v := range categories {
		vsm[i] = fmt.Sprintf("'%s'", v)
	}
	return vsm
}

var categories = strings.Join(mapCategoriesToStr(category.TxCategories), ",")


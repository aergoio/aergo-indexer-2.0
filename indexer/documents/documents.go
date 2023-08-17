package documents

import (
	"fmt"
	"strings"
	"time"

	tx "github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
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
func (m *BaseEsType) GetID() string {
	return m.Id
}

// SetID sets the document's id
func (m *BaseEsType) SetID(id string) {
	m.Id = id
}

// EsChainInfo is meta data of a chain information
type EsChainInfo struct {
	*BaseEsType
	Public    bool   `json:"public" db:"public"`
	Mainnet   bool   `json:"mainnet" db:"mainnet"`
	Consensus string `json:"consensus" db:"consensus"`
	Version   uint64 `json:"version" db:"version"`
}

// EsBlock is a block stored in the database
type EsBlock struct {
	*BaseEsType
	Timestamp     time.Time `json:"ts" db:"ts"`
	BlockNo       uint64    `json:"no" db:"no"`
	PreviousBlock string    `json:"previous_block" db:"previous_block"`
	TxCount       uint64    `json:"txs" db:"txs"`
	Size          uint64    `json:"size" db:"size"`
	Coinbase      string    `json:"coinbase" db:"coinbase"`
	BlockProducer string    `json:"block_producer" db:"block_producer"`
	RewardAccount string    `json:"reward_account" db:"reward_account"`
	RewardAmount  string    `json:"reward_amount" db:"reward_amount"`
}

// EsTx is a transaction stored in the database
type EsTx struct {
	*BaseEsType
	BlockNo       uint64        `json:"blockno" db:"blockno"`
	Timestamp     time.Time     `json:"ts" db:"ts"`
	TxIdx         uint64        `json:"tx_idx" db:"tx_idx"`
	Payload       string        `json:"payload" db:"payload"`
	Account       string        `json:"from" db:"from"`
	Recipient     string        `json:"to" db:"to"`
	Amount        string        `json:"amount" db:"amount"`             // string of BigInt
	AmountFloat   float32       `json:"amount_float" db:"amount_float"` // float for sorting
	Type          string        `json:"type" db:"type"`
	Category      tx.TxCategory `json:"category" db:"category"`
	Method        string        `json:"method" db:"method"`
	Status        string        `json:"status" db:"status"`
	Result        string        `json:"result" db:"result"`
	FeeDelegation bool          `json:"fee_delegation" db:"fee_delegation"`
	GasPrice      string        `json:"gas_price" db:"gas_price"`
	GasLimit      uint64        `json:"gas_limit" db:"gas_limit"`
	GasUsed       uint64        `json:"gas_used" db:"gas_used"`
}

type EsContract struct {
	*BaseEsType
	TxId      string    `json:"tx_id" db:"tx_id"`
	Creator   string    `json:"creator" db:"creator"`
	BlockNo   uint64    `json:"blockno" db:"blockno"`
	Timestamp time.Time `json:"ts" db:"ts"`
	Payload   string    `json:"payload" db:"payload"`

	VerifiedToken  string `json:"verified_token" db:"verified_token"`
	VerifiedStatus string `json:"verified_status" db:"verified_status"`
	VerifiedCode   string `json:"verified_code" db:"verified_code"`
}

type EsContractUp struct {
	*BaseEsType
	Payload        string `json:"payload" db:"payload"`
	VerifiedToken  string `json:"verified_token" db:"verified_token"`
	VerifiedStatus string `json:"verified_status" db:"verified_status"`
	VerifiedCode   string `json:"verified_code" db:"verified_code"`
}

// EsEvent is a contract-event mapping stored in the database
type EsEvent struct {
	*BaseEsType
	Contract  string `json:"contract" db:"contract"`
	BlockNo   uint64 `json:"blockno" db:"blockno"`
	TxId      string `json:"tx_id" db:"tx_id"`
	TxIdx     uint64 `json:"tx_idx" db:"tx_idx"`
	EventIdx  uint64 `json:"event_idx" db:"event_idx"`
	EventName string `json:"event_name" db:"event_name"`
	EventArgs string `json:"event_args" db:"event_args"`
}

// EsName is a name-address mapping stored in the database
type EsName struct {
	*BaseEsType
	Name     string `json:"name" db:"name"`
	Address  string `json:"address" db:"address"`
	BlockNo  uint64 `json:"blockno" db:"blockno"`
	UpdateTx string `json:"tx" db:"tx"`
}

// EsToken is meta data of a token. The id is the contract address.
type EsToken struct {
	*BaseEsType
	TxId         string       `json:"tx_id" db:"tx_id"`
	BlockNo      uint64       `json:"blockno" db:"blockno"`
	Creator      string       `json:"creator" db:"creator"`
	Type         tx.TokenType `json:"type" db:"type"`
	Name         string       `json:"name" db:"name"`
	Name_lower   string       `json:"name_lower" db:"name_lower"`
	Symbol       string       `json:"symbol" db:"symbol"`
	Symbol_lower string       `json:"symbol_lower" db:"symbol_lower"`
	Decimals     uint8        `json:"decimals" db:"decimals"`
	Supply       string       `json:"supply" db:"supply"`
	SupplyFloat  float32      `json:"supply_float" db:"supply_float"`
}

type EsTokenUp struct {
	*BaseEsType
	Supply      string  `json:"supply" db:"supply"`
	SupplyFloat float32 `json:"supply_float" db:"supply_float"`
}

type EsTokenVerified struct {
	*BaseEsType
	TokenAddress  string `json:"token_address" db:"token_address"`
	Owner         string `json:"owner" db:"owner"`
	Comment       string `json:"comment" db:"comment"`
	Email         string `json:"email" db:"email"`
	RegDate       string `json:"regdate" db:"regdate"`
	ImageUrl      string `json:"image_url" db:"image_url"`
	HomepageUrl   string `json:"homepage_url" db:"homepage_url"`
	Name          string `json:"name" db:"name"`
	Name_lower    string `json:"name_lower" db:"name_lower"`
	Symbol        string `json:"symbol" db:"symbol"`
	Symbol_lower  string `json:"symbol_lower" db:"symbol_lower"`
	Type          string `json:"type" db:"type"`
	TotalSupply   string `json:"total_supply" db:"total_supply"`
	TotalTransfer uint64 `json:"total_transfer" db:"total_transfer"`
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

// EsAccountTokens is meta data of a token of an account. The id is account_token address.
type EsAccountTokens struct {
	*BaseEsType
	Account      string       `json:"account" db:"account"`
	TokenAddress string       `json:"address" db:"address"`
	Type         tx.TokenType `json:"type" db:"type"`
	Timestamp    time.Time    `json:"ts" db:"ts"`
	Balance      string       `json:"balance" db:"balance"`
	BalanceFloat float32      `json:"balance_float" db:"balance_float"`
}

type EsAccountTokensUp struct {
	*BaseEsType
	Timestamp    time.Time `json:"ts" db:"ts"`
	Balance      string    `json:"balance" db:"balance"`
	BalanceFloat float32   `json:"balance_float" db:"balance_float"`
}

// EsAccountBalance is meta data of a balance of an account. The id is account_balance address.
type EsAccountBalance struct {
	*BaseEsType
	BlockNo      uint64    `json:"blockno" db:"blockno"`
	Timestamp    time.Time `json:"ts" db:"ts"`
	Balance      string    `json:"balance" db:"balance"`
	BalanceFloat float32   `json:"balance_float" db:"balance_float"`
	Staking      string    `json:"staking" db:"staking"`
	StakingFloat float32   `json:"staking_float" db:"staking_float"`
}

type EsNFT struct {
	*BaseEsType
	TokenAddress string    `json:"address" db:"address"`
	TokenId      string    `json:"token_id" db:"token_id"`
	Account      string    `json:"account" db:"account"`
	BlockNo      uint64    `json:"blockno" db:"blockno"`
	Timestamp    time.Time `json:"ts" db:"ts"`
	TokenUri     string    `json:"token_uri" db:"token_uri"`
	ImageUrl     string    `json:"image_url" db:"image_url"`
}

type EsNFTUp struct {
	*BaseEsType
	Account   string    `json:"account" db:"account"`
	BlockNo   uint64    `json:"blockno" db:"blockno"`
	Timestamp time.Time `json:"ts" db:"ts"`
}

var EsMappings map[string]string

func InitEsMappings(clusterMode bool) {
	if clusterMode {
		EsMappings = map[string]string{
			"chain_info": `{
				"settings": {
					"number_of_shards": 10,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"id": {
							"type": "keyword"
						},
						"public": {
							"type": "boolean"
						},
						"mainnet": {
							"type": "boolean"
						},
						"consensus": {
							"type": "keyword"
						},
						"version": {
							"type": "long"
						}
					}
				}
			}`,
			"block": `{
				"settings": {
					"number_of_shards": 100,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"ts": {
							"type": "date"
						},
						"no": {
							"type": "long"
						},
						"previous_block": {
							"type": "keyword"
						},
						"txs": {
							"type": "long"
						},
						"size": {
							"type": "long"
						},
						"coinbase": {
							"type": "keyword"
						},
						"block_producer": {
							"type": "keyword"
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
			"tx": `{
				"settings": {
					"number_of_shards": 50,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"blockno": {
							"type": "long"
						},
						"ts": {
							"type": "date"
						},
						"tx_idx": {
							"type": "long"
						},
						"payload": {
							"type": "keyword"
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
						},
						"status": {
							"type": "keyword"
						},
						"result": {
							"type": "keyword"
						}
						"fee_delegation": {
							"type": "boolean"
						},
						"gas_price": {
							"type": "keyword"
						},
						"gas_used": {
							"type": "long"
						},
						"gas_limit": {
							"type": "long"
						}
					}
				}
			}`,
			"contract": `{
				"settings": {
					"number_of_shards": 10,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"tx_id": {
							"type": "keyword"
						},
						"creator": {
							"type": "keyword"
						},
						"blockno": {
							"type": "long"
						},
						"ts": {
							"type": "date"
						},
						"verified": {
							"type": "keyword"
						},
						"verified_no": {
							"type": "long"
						},
						"verified_txid": {
							"type": "keyword"
						},
						"verified_code": {
							"type": "keyword"
						}
					}
				}
			}`,
			"event": `{
				"settings": {
					"number_of_shards": 30,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"contract": {
							"type": "keyword" 
						},
						"blockno": {
							"type": "long"
						},
						"tx_id": {
							"type": "keyword"
						},
						"tx_idx": {
							"type": "long"
						},
						"event_idx": {
							"type": "long"
						},
						"event_name": {
							"type": "keyword"
						},
						"event_args": {
							"type": "keyword"
						}
					}
				}
			}`,
			"name": `{
				"settings": {
					"number_of_shards": 2,
					"number_of_replicas": 1
				},
				"mappings": {
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
			"token": `{
				"settings": {
					"number_of_shards": 5,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"tx_id": {
							"type": "keyword"
						},
						"blockno": {
							"type": "long"
						},
						"creator": {
							"type": "keyword"
						},
						"name": {
							"type": "keyword"
						},
						"name_lower": {
							"type": "keyword"
						},
						"symbol": {
							"type": "keyword"
						},
						"symbol_lower": {
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
						"type": {
							"type": "keyword"
						}
					}
				}
			}`,
			"token_verified": `{
				"settings": {
					"number_of_shards": 5,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"token_address": {
							"type": "keyword"
						},
						"owner": {
							"type": "keyword"
						},
						"comment": {
							"type": "keyword"
						},
						"email": {
							"type": "keyword"
						},
						"regdate": {
							"type": "keyword"
						},
						"homepage_url": {
							"type": "keyword"
						},
						"image_url": {
							"type": "keyword"
						},
						"name": {
							"type": "keyword"
						},
						"name_lower": {
							"type": "keyword"
						},
						"symbol": {
							"type": "keyword"
						},
						"symbol_lower": {
							"type": "keyword"
						},
						"type": {
							"type": "keyword"
						},
						"total_supply": {
							"type": "keyword"
						},
						"total_transfer": {
							"type": "long"
						}
					}
				}
			}`,
			"token_transfer": `{
				"settings": {
					"number_of_shards": 30,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
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
			"account_tokens": `{
				"settings": {
					"number_of_shards": 10,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
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
				"settings": {
					"number_of_shards": 30,
					"number_of_replicas": 1
				},
				"mappings": {
					"properties": {
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
						},
						"token_uri": {
							"type": "keyword"
						},
						"image_url": {
							"type": "keyword"
						}
					}
				}
			}`,
			"account_balance": `{
				"settings": {
					"number_of_shards": 10,
					"number_of_replicas": 1
				},
				"mappings": {
					"properties": {
						"id": {
							"type": "keyword"
						},
						"blockno": {
							"type": "long"
						},
						"ts": {
							"type": "date"
						},
						"balance": {
							"enabled": false
						},
						"balance_float": {
							"type": "float"
						},
						"staking": {
							"enabled": false
						},
						"staking_float": {
							"type": "float"
						}
					}
				}
			}`,
		}
	} else {
		EsMappings = map[string]string{
			"chain_info": `{
				"settings": {
					"number_of_shards": 3,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"id": {
							"type": "keyword"
						},
						"public": {
							"type": "boolean"
						},
						"mainnet": {
							"type": "boolean"
						},
						"consensus": {
							"type": "keyword"
						},
						"version": {
							"type": "long"
						}
					}
				}
			}`,
			"block": `{
				"settings": {
					"number_of_shards": 20,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"ts": {
							"type": "date"
						},
						"no": {
							"type": "long"
						},
						"previous_block": {
							"type": "keyword"
						},
						"txs": {
							"type": "long"
						},
						"size": {
							"type": "long"
						},
						"coinbase": {
							"type": "keyword"
						},
						"block_producer": {
							"type": "keyword"
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
			"tx": `{
				"settings": {
					"number_of_shards": 10,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"blockno": {
							"type": "long"
						},
						"ts": {
							"type": "date"
						},
						"tx_idx": {
							"type": "long"
						},
						"payload": {
							"type": "keyword"
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
						"status": {
							"type": "keyword"
						},
						"result": {
							"type": "keyword"
						}
						"fee_delegation": {
							"type": "boolean"
						},
						"gas_price": {
							"type": "keyword"
						},
						"gas_used": {
							"type": "long"
						},
						"gas_limit": {
							"type": "long"
						}
					}
				}
			}`,
			"contract": `{
				"settings": {
					"number_of_shards": 10,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"tx_id": {
							"type": "keyword"
						},
						"creator": {
							"type": "keyword"
						},
						"blockno": {
							"type": "long"
						},
						"ts": {
							"type": "date"
						},
						"verified": {
							"type": "keyword"
						},
						"verified_no": {
							"type": "long"
						},
						"verified_txid": {
							"type": "keyword"
						},
						"verified_code": {
							"type": "keyword"
						}
					}
				}
			}`,
			"event": `{
				"settings": {
					"number_of_shards": 10,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"contract": {
							"type": "keyword" 
						},
						"blockno": {
							"type": "long"
						},
						"tx_id": {
							"type": "keyword"
						},
						"tx_idx": {
							"type": "long"
						},
						"event_idx": {
							"type": "long"
						},
						"event_name": {
							"type": "keyword"
						},
						"event_args": {
							"type": "keyword"
						}
					}
				}
			}`,
			"name": `{
				"settings": {
					"number_of_shards": 1,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
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
			"token": `{
				"settings": {
					"number_of_shards": 3,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"tx_id": {
							"type": "keyword"
						},
						"blockno": {
							"type": "long"
						},
						"creator": {
							"type": "keyword"
						},
						"name": {
							"type": "keyword"
						},
						"name_lower": {
							"type": "keyword"
						},
						"symbol": {
							"type": "keyword"
						},
						"symbol_lower": {
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
			"token_verified": `{
				"settings": {
					"number_of_shards": 5,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"token_address": {
							"type": "keyword"
						},
						"owner": {
							"type": "keyword"
						},
						"comment": {
							"type": "keyword"
						},
						"email": {
							"type": "keyword"
						},
						"regdate": {
							"type": "keyword"
						},
						"homepage_url": {
							"type": "keyword"
						},
						"image_url": {
							"type": "keyword"
						},
						"name": {
							"type": "keyword"
						},
						"name_lower": {
							"type": "keyword"
						},
						"symbol": {
							"type": "keyword"
						},
						"symbol_lower": {
							"type": "keyword"
						},
						"type": {
							"type": "keyword"
						},
						"total_supply": {
							"type": "keyword"
						},
						"total_transfer": {
							"type": "long"
						}
					}
				}
			}`,
			"token_transfer": `{
				"settings": {
					"number_of_shards": 3,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
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
			"account_tokens": `{
				"settings": {
					"number_of_shards": 3,
					"number_of_replicas": 1
				},
				"mappings": {
					"properties": {
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
			"account_balance": `{
				"settings": {
					"number_of_shards": 3,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
						"id": {
							"type": "keyword"
						},
						"blockno": {
							"type": "long"
						},
						"ts": {
							"type": "date"
						},
						"balance": {
							"enabled": false
						},
						"balance_float": {
							"type": "float"
						},
						"staking": {
							"enabled": false
						},
						"staking_float": {
							"type": "float"
						}
					}
				}
			}`,
			"nft": `{
				"settings": {
					"number_of_shards": 3,
					"number_of_replicas": 1,
					"index.max_result_window": 100000
				},
				"mappings": {
					"properties": {
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
						},
						"token_uri": {
							"type": "keyword"
						},
						"image_url": {
							"type": "keyword"
						}
					}
				}
			}`,
		}
	}
}

func mapCategoriesToStr(categories []tx.TxCategory) []string {
	vsm := make([]string, len(categories))
	for i, v := range categories {
		vsm[i] = fmt.Sprintf("'%s'", v)
	}
	return vsm
}

var categories = strings.Join(mapCategoriesToStr(tx.TxCategories), ",")

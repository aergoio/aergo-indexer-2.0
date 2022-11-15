# Aergo Metadata Indexer

This is a go program that connects to aergo server over RPC and synchronizes blockchain metadata with a database. It currently supports Elasticsearch.

This creates the indices,
   1. `block`
   2. `tx`
   3. `name` (with a prefix)
   4. `token`
   5. `token_transfer`
   6. `account_tokens`
   7. `nft`
   8. `contract`
Check [indexer/documents/documents.go](./indexer/documents/documents.go) for the exact mappings for all supported databases.

When using Elasticsearch, multiple indexing instances can be run concurrently using these two mechanisms (can be used together):
- The indexer creates a [time-based lock](https://github.com/graup/es-distributed-lock) in ES, excluding other instances writing to the same data set (enabled by default, depending on --prefix).
- When a data conflict occurs upon indexing, the indexer can set itself into an idle mode, assuming that another instance is running (enabled by 5 seconds).

## Indexed data

block
```
Field           Type        Comment
id              string      block hash
ts              timestamp   block creation timestamp
no              uint64      block number
txs             uint        number of transactions
size            uint64      block size in bytes
reward_account  string      reward account
reward_amount   string      reward amount 
```

tx (transactions)
```
Field           Type        Comment
id              string      tx hash
ts              timestamp   block creation timestamp
blockno         uint64      block number
from            string      from address (base58check encoded)
to              string      to address (base58check encoded)
amount          string      Precise BigInt string representation of amount
amount_float    float32     Imprecise float representation of amount, useful for sorting
type            string      tx type
category        string      user-friendly category
method          string      called function name of a contract
token_transfers uint64      number of token transfers in this tx
```

name
```
Field           Type        Comment
id              string      name + tx hash
name            string      name
address         string      address (base58check encoded)
blockno         uint64      block in which name was updated
tx              string      tx in which name was updated
```

token_transfer
```
Field           Type        Comment
id              string      tx hash + index on tx
tx_id           string      tx hash
ts              timestamp   block creation timestamp
blockno         uint64      block number
address         string      contract address (base58check encoded)
from            string      from address (base58check encoded)
to              string      to address (base58check encoded)
sender          string      tx sender address (base58check encoded)
amount          string      Precise BigInt string representation of amount
amount_float    float32     Imprecise float representation of amount, useful for sorting
token_id        string      NFD id (for ARC2)
```

token
```
Field           Type        Comment
id              string      address of token contract
tx_id           string      tx hash 
blockno         uint64      block number
type            string      token type (ARC1/ARC2)
name            string      token name
symbol          string      token symol
token_tranefers uint64      number of token transfers
decimals        uint8       decimals of token
supply          string      Precise BigInt string representation of total supply 
supply_float    float32     Imprecise float representation of amount, useful for sorting
```

account_tokens
```
Field           Type        Comment
id              string      account address + token address
account         string      account address
address         string      token address 
type            string      token type (ARC1/ARC2)
ts              timestamp   last updated timestamp
balance         string      Precise BigInt string representation of total supply
balance_float   float32     Imprecise float representation of amount, useful for sorting
```

nft 
```
Field           Type        Comment
id              string      nft id
address         string      contract address 
token_id        string      nft id
account         string      account address
blockno         uint64      block number
ts              timestamp   last updated timestamp
```

contract
```
Field           Type        Comment
id              string      contract address
tx_id           string      tx hash
creator         string      creators address
blockno         uint64      block number
ts              timestamp   last updated timestamp
```


## Usage

```
Usage:
  indexer [flags]

Flags:
  -A, --aergo string       host and port of aergo server. Alternative to setting host and port separately.
  -E, --dburl string       Database URL (default "http://localhost:9200")
      --from int32         start syncing from this block number
      --to int32           stop syncing at this block number
  -h, --help               help for indexer
  -H, --host string        host address of aergo server (default "localhost")
  -p, --port int32         port number of aergo server (default 7845)
  -X, --prefix string      prefix used for index names (default "testnet_")
      --bulk               size of bulk for batch indexing
      --batch              time limit for batch indexing
      --miner              number of processes minning blocks
      --grpc               number of grpc connections to full nodes
      --check              check and fix index "--from" blocks  
      --cluster            elasticsearch cluster mode
```

Example

    ./bin/indexer -H localhost -p 7845 --dburl http://localhost:9200 --prefix chain_

You can use the `--prefix` parameter and multiple instances of this program to sync several blockchains with one database.

Instead of setting host and port of the aergo server separately, you can also pass them at once with `-A localhost:7845`.

To check (or reindex) 

    ./bin/indexer --check

When reindexing, this creates new indices to sync the blockchain from scratch.

## Build

    go get github.com/aergoio/aergo-indexer
    cd $GOPATH/src/github.com/aergoio/aergo-indexer
    make

## Build and run using Docker

[Automatic latest build from master on Docker Hub](http://hub.docker.com/r/aergo/indexer)

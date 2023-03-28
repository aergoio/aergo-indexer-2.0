# Aergo Metadata Indexer

This is a go program that connects to aergo server over RPC and synchronizes blockchain metadata with a database. It currently supports Elasticsearch.

This creates the indices,
   1. `block`
   2. `tx`
   3. `name` (with a prefix)
   4. `token`
   5. `token_transfer`
   6. `account_balance`
   7. `account_tokens`
   8. `nft`
   9. `contract`

Check [indexer/documents/documents.go](./indexer/documents/documents.go) for the exact mappings for all supported databases.

When using Elasticsearch, multiple indexing instances can be run concurrently using these two mechanisms (can be used together):
- The indexer creates a [time-based lock](https://github.com/graup/es-distributed-lock) in ES, excluding other instances writing to the same data set (enabled by default, depending on --prefix).
- When a data conflict occurs upon indexing, the indexer can set itself into an idle mode, assuming that another instance is running (enabled by 5 seconds).

## Indexed data

block
```
Field           Type        Comment
id              string      block hash
ts              timestamp   block creation timestamp (unixnano)
no              uint64      block number
txs             uint        number of transactions
size            uint64      block size in bytes
block_producer  string      block producer peer id
reward_account  string      reward account
reward_amount   string      reward amount
```

tx (transactions)
```
Field           Type        Comment
id              string      tx hash
ts              timestamp   block creation timestamp (unixnano)
blockno         uint64      block number
from            string      from address (base58check encoded)
to              string      to address (base58check encoded)
amount          string      Precise BigInt string representation of amount
amount_float    float32     Imprecise float representation of amount, useful for sorting
type            string      tx type
category        string      user-friendly category
method          string      called function name of a contract
token_transfers uint64      number of token transfers in this tx
status          string      tx status from receipt (CREATED/SUCCESS/ERROR)
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
name_lower      string      token name lowercase, useful to case-insensitive search
symbol          string      token symbol
symbol_lower    string      token symbol lowercase, useful to case-insensitive search
token_transfers uint64      number of token transfers
decimals        uint8       decimals of token
supply          string      Precise BigInt string representation of total supply 
supply_float    float32     Imprecise float representation of amount, useful for sorting
```

account_balance
```
Field           Type        Comment
id              string      account address
ts              timestamp   last updated timestamp (unixnano)
blockno         uint64      last updated block number
balance         string      Precise BigInt string representation of aergo total balance
balance_float   float32     Imprecise float representation of aergo total balance, useful for sorting
staking         string      Precise BigInt string representation of aergo staking
staking_float   float32     Imprecise float representation of aergo staking, useful for sorting
```

account_tokens
```
Field           Type        Comment
id              string      account address + token address
account         string      account address
address         string      token address
type            string      token type (ARC1/ARC2)
ts              timestamp   last updated timestamp (unixnano)
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
ts              timestamp   last updated timestamp (unixnano)
token_uri       string      token uri
image_url       string      image url
```

contract
```
Field           Type        Comment
id              string      contract address
tx_id           string      tx hash
creator         string      creators address
blockno         uint64      block number
ts              timestamp   last updated timestamp (unixnano)
```


## Usage

```
Usage:
  indexer [flags]

Flags:
  -A, --aergo string        host and port of aergo server. Alternative to setting host and port separately.
      --cccv string         indexing cccv nft by network type ( mainnet or testnet ). only use for cccv
      --check               check and fix indices of range of heights
      --clean               check and clean unexpected data in indices
  -C, --cluster             elasticsearch cluster type
  -E, --dburl string        Database URL (default "localhost:9200")
      --from uint           start syncing from this block number. check only
  -h, --help                help for indexer
  -H, --host string         host address of aergo server (default "localhost")
  -M, --mode string         indexer running mode. Alternative to setting check, clean separately. (default "onsync")
  -p, --port int32          port number of aergo server (default 7845)
  -P, --prefix string       index name prefix
      --to uint             stop syncing at this block number. check only
  -W, --whitelist strings   address for indexing whitelist balance. onsync only
```

Example

    ./bin/indexer -H localhost -p 7845 --dburl http://localhost:9200

You can use the `--prefix` parameter and multiple instances of this program to sync several blockchains with one database.

Instead of setting host and port of the aergo server separately, you can also pass them at once with `-A localhost:7845`.

To check (or reindex) 

    ./bin/indexer --check

When reindexing, this creates new indices to sync the blockchain from scratch.

## Build

    go get github.com/aergoio/aergo-indexer
    cd $GOPATH/src/github.com/aergoio/aergo-indexer
    make

## Build and run using Docker Compose

    docker compose -p aergo_indexer -f docker-compose.testnet.yml up

[Automatic latest build from master on Docker Hub](https://hub.docker.com/r/aergo/indexer2)

# Aergo Metadata Indexer

This is a go program that connects to aergo server over RPC and synchronizes blockchain metadata with a database. It currently supports Elasticsearch and MySQL/MariaDB.

This creates the indices `block`, `tx`, and `name` (with a prefix). These are actually aliases that point to the latest version of the data.
Check [indexer/documents/documents.go](./indexer/documents/documents.go) for the exact mappings for all supported databases.

When using Elasticsearch, multiple indexing instances can be run concurrently using these two mechanisms (can be used together):
- The indexer creates a [time-based lock](https://github.com/graup/es-distributed-lock) in ES, excluding other instances writing to the same data set (enabled by default, depending on --prefix).
- When a data conflict occurs upon indexing, the indexer can set itself into an idle mode, assuming that another instance is running (enabled by e.g. `--conflict 30`).

Using both mechanisms, you achieve both [efficiency-improving locking](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html) and [optimistic concurrency control](https://qbox.io/blog/optimistic-concurrency-control-in-elasticsearch).

## Indexed data

Blocks
```
Field    Type        Comment
id       string      block hash
ts       timestamp   block creation timestamp
no       uint64      block number
txs      uint        number of transactions
size     uint64      block size in bytes
```

Transaction
```
Field          Type        Comment
id             string      tx hash
ts             timestamp   block creation timestamp
blockno        uint64      block number
from           string      from address (base58check encoded)
to             string      to address (base58check encoded)
amount         string      Precise BigInt string representation of amount
amount_float   f32         Imprecise float representation of amount, useful for sorting
type           string      "0" or "1"
category       string      user-friendly category
```

Names
```
Field    Type        Comment
id       string      name + tx hash
name     string
address  string      address (base58check encoded)
blockno  uint64      block in which name was updated
tx       string      tx in which name was updated
```

## Usage

```
Usage:
  indexer [flags]

Flags:
  -A, --aergo string       host and port of aergo server. Alternative to setting host and port separately.
      --conflict int32     time to idle when a conflict occurs (in seconds). Use this for optimistic concurrency. Elasticsearch only
  -E, --dburl string       Database URL (default "http://localhost:9200")
      --from int32         start syncing from this block number
  -h, --help               help for indexer
  -H, --host string        host address of aergo server (default "localhost")
  -p, --port int32         port number of aergo server (default 7845)
  -X, --prefix string      prefix used for index names (default "chain_")
      --rebuild            reindex blocks "--from" block height and swap index after catching up
      --check              check and fix index "--from"  blocks  
      --to int32           stop syncing at this block number (default -1)
```

Example

    ./bin/indexer -H localhost -p 7845 --dburl http://localhost:9200 --prefix chain_

You can use the `--prefix` parameter and multiple instances of this program to sync several blockchains with one database.

Instead of setting host and port of the aergo server separately, you can also pass them at once with `-A localhost:7845`.

To reindex (starting from scratch):

    ./bin/indexer --reindex

When reindexing, this creates new indices to sync the blockchain from scratch.
After catching up, the aliases are replaced with the new data and the old indices removed.
This means the old data can still be accessed until the sync is complete.

## Build

    go get github.com/aergoio/aergo-indexer
    cd $GOPATH/src/github.com/aergoio/aergo-indexer
    make

Requires Go Modules (`GO111MODULE=on`)

## Build and run using Docker

    docker build -t aergo/indexer .
    docker run aergo/indexer indexer -A ip:7845 -E ip:9200 --prefix chain_

[Automatic latest build from master on Docker Hub](http://hub.docker.com/r/aergo/indexer)

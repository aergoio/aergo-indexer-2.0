package indexer

import (
	"time"

	"github.com/aergoio/aergo-lib/log"
)

type IndexerOptionFunc func(*Indexer) error

func SetLogger(logger *log.Logger) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.log = logger
		return nil
	}
}

func SetServerAddr(serverAddr string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.serverAddr = serverAddr
		return nil
	}
}

func SetDBAddr(dbAddr string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.dbAddr = dbAddr
		return nil
	}
}

func SetNetworkType(networkType string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.networkType = networkType
		return nil
	}
}

func SetRunMode(runMode string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.runMode = runMode
		return nil
	}
}

func SetBulkSize(bulkSize int32) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.bulkSize = bulkSize
		return nil
	}
}

func SetBatchTime(batchTime int32) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.batchTime = time.Duration(batchTime) * time.Second
		return nil
	}
}

func SetStartHeight(starHeight uint64) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.startHeight = starHeight
		return nil
	}
}

func SetMinerNum(minerNum int) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.minerNum = minerNum
		return nil
	}
}

func SetGrpcNum(grpcNum int) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.grpcNum = grpcNum
		return nil
	}
}

func SetWhiteListAddresses(whiteListAddresses []string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.whiteListAddresses = whiteListAddresses
		return nil
	}
}

func SetWhiteListBlockInterval(whiteListBlockInterval uint64) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.whiteListBlockInterval = whiteListBlockInterval
		return nil
	}
}

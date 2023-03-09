package indexer

import (
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

func SetPrefix(prefix string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.prefix = prefix
		return nil
	}
}

func SetInitCccvNft(initCccvNft string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.initCccvNft = initCccvNft
		return nil
	}
}

func SetRunMode(runMode string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.runMode = runMode
		return nil
	}
}

func SetStartHeight(starHeight uint64) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.startHeight = starHeight
		return nil
	}
}

func SetWhiteListAddresses(whiteListAddresses []string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		for _, addr := range whiteListAddresses {
			indexer.whiteListAddresses.Store(addr, true)
		}
		return nil
	}
}

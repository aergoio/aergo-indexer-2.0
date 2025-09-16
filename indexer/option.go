package indexer

import (
	"github.com/aergoio/aergo-indexer-2.0/types"
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

func SetNetworkTypeForCccv(initCccvNft string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.networkTypeForCccv = initCccvNft
		return nil
	}
}

func SetRunMode(runMode string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.runMode = runMode
		return nil
	}
}

func SetFix(fix bool) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.fix = fix
		return nil
	}
}

func SetWhiteListAddresses(whiteListAddresses []string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.balanceWhitelist = whiteListAddresses
		return nil
	}
}

func SetTokenVerifyAddress(verifyTokenAddress string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		if verifyTokenAddress == "" {
			return nil
		}
		raw, err := types.DecodeAddress(verifyTokenAddress)
		if err != nil {
			return err
		}

		indexer.tokenVerifyAddr = raw
		return nil
	}
}

func SetContractVerifyAddress(verifyContractAddress string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		if verifyContractAddress == "" {
			return nil
		}
		raw, err := types.DecodeAddress(verifyContractAddress)
		if err != nil {
			return err
		}

		indexer.contractVerifyAddr = raw
		return nil
	}
}

func SetTokenVerifyWhitelist(verifyTokenWhitelist []string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.tokenVerifyWhitelist = verifyTokenWhitelist
		return nil
	}
}

func SetContractVerifyWhitelist(verifyContractWhitelist []string) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		indexer.contractVerifyWhitelist = verifyContractWhitelist
		return nil
	}
}

func SetMaxESConnection(maxESConn int) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		if maxESConn > 0 {
			indexer.maxConnections = maxESConn
			if indexer.maxIdleConns > maxESConn {
				indexer.maxIdleConns = minInt(indexer.maxIdleConns, maxESConn)
			}
		}
		return nil
	}
}

func SetTraceESWrite(trace bool) IndexerOptionFunc {
	return func(indexer *Indexer) error {
		if trace {
			indexer.traceESWrite = trace
		}
		return nil
	}
}

func minInt(v1 int, v2 int) int {
	if v1 < v2 {
		return v1
	}
	return v2
}

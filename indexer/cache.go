package indexer

import (
	"io"
	"sync"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
)

type Cache struct {
	idxer *Indexer

	accToken     sync.Map
	peerId       sync.Map
	addrsBalance sync.Map
	// addrsVerifiedToken    sync.Map
	// addrsVerifiedContract sync.Map
}

func NewCache(idxer *Indexer) *Cache {
	cache := &Cache{
		idxer: idxer,
	}

	for _, tokenAddr := range idxer.tokenVerifyWhitelist {
		idxer.addWhitelist(doc.ConvWhitelist(tokenAddr, "", "token"))
	}
	for _, contractAddr := range idxer.contractVerifyWhitelist {
		idxer.addWhitelist(doc.ConvWhitelist(contractAddr, "", "contract"))
	}
	for _, balanceAddr := range idxer.balanceWhitelist {
		cache.storeBalance(balanceAddr)
	}
	return cache
}

// register staking account to white list. ( staking addresses receive rewards by block creation )
func (c *Cache) registerVariables() {
	// register verify token
	scroll := c.idxer.db.Scroll(db.QueryParams{
		IndexName: c.idxer.indexNamePrefix + "token",
		SortField: "blockno",
		Size:      100,
		From:      0,
		SortAsc:   true,
		StringMatch: &db.StringMatchQuery{
			Field: "verified_status",
			Value: string(Verified),
		},
	}, func() doc.DocType {
		token := new(doc.EsToken)
		token.BaseEsType = new(doc.BaseEsType)
		return token
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if tokenVerified, ok := document.(*doc.EsToken); ok {
			if tokenVerified.TokenAddress != "" {
				c.idxer.addWhitelist(doc.ConvWhitelist(tokenVerified.TokenAddress, tokenVerified.Id, "token"))
			}
		}
	}

	// register verify contract
	scroll = c.idxer.db.Scroll(db.QueryParams{
		IndexName: c.idxer.indexNamePrefix + "contract",
		SortField: "blockno",
		Size:      100,
		From:      0,
		SortAsc:   true,
		StringMatch: &db.StringMatchQuery{
			Field: "verified_status",
			Value: string(Verified),
		},
	}, func() doc.DocType {
		contract := new(doc.EsContract)
		contract.BaseEsType = new(doc.BaseEsType)
		return contract
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if contract, ok := document.(*doc.EsContract); ok {
			if contract.VerifiedToken != "" {
				c.idxer.addWhitelist(doc.ConvWhitelist(contract.VerifiedToken, contract.Id, "contract"))
			}
		}
	}

	// register balances
	scroll = c.idxer.db.Scroll(db.QueryParams{
		IndexName: c.idxer.indexNamePrefix + "account_balance",
		SortField: "staking_float",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		balance := new(doc.EsAccountBalance)
		balance.BaseEsType = new(doc.BaseEsType)
		return balance
	})
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if balance, ok := document.(*doc.EsAccountBalance); ok {
			c.storeBalance(balance.Id)
		}
	}
}

func (ns *Cache) refreshVariables(info BlockInfo, blockDoc *doc.EsBlock, minerGRPC *client.AergoClientController) {
	// update verify token, contract
	scroll := ns.idxer.db.Scroll(db.QueryParams{
		IndexName: ns.idxer.indexNamePrefix + "whitelist",
		SortField: "type",
		Size:      10000,
		From:      0,
		SortAsc:   true,
	}, func() doc.DocType {
		whitelist := new(doc.EsWhitelist)
		whitelist.BaseEsType = new(doc.BaseEsType)
		return whitelist
	})
	mapWhitelist := make(map[string][2]string)
	for {
		document, err := scroll.Next()
		if err == io.EOF {
			break
		}
		if whitelist, ok := document.(*doc.EsWhitelist); ok {
			metadata := minerGRPC.QueryMetadataOf(ns.idxer.tokenVerifyAddr, whitelist.Id)

			var updateContractAddress string
			if whitelist.Type == "token" {
				ns.idxer.log.Info().Str("tokenAddress", whitelist.Id).Msg("update verified token")
				updateContractAddress = ns.idxer.MinerTokenVerified(whitelist.Id, whitelist.Contract, metadata, minerGRPC)
			}
			if whitelist.Type == "contract" {
				ns.idxer.log.Info().Str("tokenAddress", whitelist.Id).Msg("update verified contract")
				updateContractAddress = ns.idxer.MinerContractVerified(whitelist.Id, whitelist.Contract, metadata, minerGRPC)
			}
			mapWhitelist[whitelist.Id] = [2]string{updateContractAddress, whitelist.Type}
		}
	}
	// refresh verify token, contract
	for tokenAddr, contractAddr := range mapWhitelist {
		ns.idxer.addWhitelist(doc.ConvWhitelist(tokenAddr, contractAddr[0], contractAddr[1]))
	}

	// update whitelist balance
	mapBalance := make(map[string]string)
	ns.addrsBalance.Range(func(k, v interface{}) bool {
		if addr, ok := k.(string); ok {
			if isExiststake := ns.idxer.MinerBalance(blockDoc, addr, minerGRPC); isExiststake == false {
				mapBalance[addr] = ""
			}
		}
		return true
	})
	// if stake amount not exist, remove balance from whitelist ( prevent repeat update )
	for addr := range mapBalance {
		ns.addrsBalance.Delete(addr)
	}
}

func (c *Cache) getPeerId(pubKey []byte) string {
	// if exist, return peerId
	if peerId, exist := c.peerId.Load(string(pubKey)); exist == true {
		return peerId.(string)
	}
	// if not exist, make peerId
	peerId := transaction.MakePeerId(pubKey)
	c.peerId.Store(string(pubKey), peerId)
	return peerId
}

func (c *Cache) getAccTokens(id string) (exist bool) {
	_, exist = c.accToken.Load(id)
	if exist != true {
		c.accToken.Store(id, true)
	}
	return exist
}

func (c *Cache) storeBalance(id string) {
	c.addrsBalance.Store(id, true)
}

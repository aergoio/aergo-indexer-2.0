package indexer

import (
	"io"
	"sync"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	"github.com/aergoio/aergo-indexer-2.0/indexer/db"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
)

type Cache struct {
	idxer *Indexer

	accToken              sync.Map
	peerId                sync.Map
	addrsWhiteListAddr    sync.Map
	addrsVerifiedToken    sync.Map
	addrsVerifiedContract sync.Map
}

func NewCache(idxer *Indexer) *Cache {
	cache := &Cache{
		idxer: idxer,
	}

	for _, whitelistAddr := range idxer.whitelistAddresses {
		cache.addrsWhiteListAddr.Store(whitelistAddr, true)
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
			Field: "verified",
			Value: string(TokenVerified),
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
			c.storeVerifiedToken(tokenVerified.TokenAddress)
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
			Value: string(TokenVerified),
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
			c.storeVerifiedContract(contract.VerifiedToken)
		}
	}

	// register whitelist
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
		if balance, ok := document.(*doc.EsAccountBalance); ok && balance.StakingFloat >= 10000 {
			c.storeWhiteList(balance.Id)
		}
	}
}

func (ns *Cache) refreshVariables(info BlockInfo, blockDoc *doc.EsBlock, minerGRPC *client.AergoClientController) {
	// update verify token
	ns.addrsVerifiedToken.Range(func(k, v interface{}) bool {
		if tokenAddress, ok := k.(string); ok {
			metadata := minerGRPC.QueryMetadataOf(ns.idxer.tokenVerifyAddr, tokenAddress)
			ns.idxer.MinerTokenVerified(tokenAddress, metadata, minerGRPC)
		}
		return true
	})

	// update verify code
	ns.addrsVerifiedContract.Range(func(k, v interface{}) bool {
		if contractAddress, ok := k.(string); ok {
			metadata := minerGRPC.QueryMetadataOf(ns.idxer.contractVerifyAddr, contractAddress)
			ns.idxer.MinerContractVerified(contractAddress, metadata, minerGRPC)
		}
		return true
	})

	// update whitelist balance
	ns.addrsWhiteListAddr.Range(func(k, v interface{}) bool {
		if addr, ok := k.(string); ok {
			if addr, err := types.DecodeAddress(addr); err == nil {
				ns.idxer.MinerBalance(blockDoc, addr, minerGRPC)
			}
		}
		return true
	})
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

func (c *Cache) storeVerifiedToken(id string) {
	c.addrsVerifiedToken.Store(id, true)
}

func (c *Cache) storeVerifiedContract(id string) {
	c.addrsVerifiedContract.Store(id, true)
}

func (c *Cache) storeWhiteList(id string) {
	c.addrsWhiteListAddr.Store(id, true)
}

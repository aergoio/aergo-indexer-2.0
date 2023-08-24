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

	accToken              sync.Map
	peerId                sync.Map
	addrsBalance          sync.Map
	addrsVerifiedToken    sync.Map
	addrsVerifiedContract sync.Map
}

func NewCache(idxer *Indexer) *Cache {
	cache := &Cache{
		idxer: idxer,
	}

	for _, balanceAddr := range idxer.balanceAddresses {
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
				c.storeVerifiedToken(tokenVerified.TokenAddress, tokenVerified.Id)
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
				c.storeVerifiedContract(contract.VerifiedToken, contract.Id)
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
	// update verify token
	mapTmp := make(map[string]string)
	ns.addrsVerifiedToken.Range(func(k, v interface{}) bool {
		tokenAddress, ok1 := k.(string)
		contractAddress, ok2 := v.(string)
		if ok1 && ok2 {
			ns.idxer.log.Info().Str("tokenAddress", tokenAddress).Msg("update verified token")
			metadata := minerGRPC.QueryMetadataOf(ns.idxer.tokenVerifyAddr, tokenAddress)
			updateContractAddress := ns.idxer.MinerTokenVerified(tokenAddress, contractAddress, metadata, minerGRPC)
			mapTmp[tokenAddress] = updateContractAddress
		}
		return true
	})
	// refresh verify token
	for tokenAddr, contractAddr := range mapTmp {
		ns.storeVerifiedToken(tokenAddr, contractAddr)
	}

	// update verify code
	mapTmp = make(map[string]string)
	ns.addrsVerifiedContract.Range(func(k, v interface{}) bool {
		tokenAddress, ok1 := k.(string)
		contractAddress, ok2 := v.(string)
		if ok1 && ok2 {
			ns.idxer.log.Info().Str("contractAddress", tokenAddress).Msg("update verified contract")
			metadata := minerGRPC.QueryMetadataOf(ns.idxer.contractVerifyAddr, tokenAddress)
			updateContractAddress := ns.idxer.MinerContractVerified(tokenAddress, contractAddress, metadata, minerGRPC)
			mapTmp[tokenAddress] = updateContractAddress
		}
		return true
	})
	// refresh verify contract
	for tokenAddr, contractAddr := range mapTmp {
		ns.storeVerifiedContract(tokenAddr, contractAddr)
	}

	// update whitelist balance
	mapTmp = make(map[string]string)
	ns.addrsBalance.Range(func(k, v interface{}) bool {
		if addr, ok := k.(string); ok {
			if isExiststake := ns.idxer.MinerBalance(blockDoc, addr, minerGRPC); isExiststake == false {
				mapTmp[addr] = ""
			}
		}
		return true
	})
	// if stake amount not exist, remove balance from whitelist ( prevent repeat update )
	for addr := range mapTmp {
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

func (c *Cache) storeVerifiedToken(tokenAddr, contractAddr string) {
	if tokenAddr != "" {
		c.addrsVerifiedToken.Store(tokenAddr, contractAddr)
	}
}

func (c *Cache) storeVerifiedContract(tokenAddr, contractAddr string) {
	if tokenAddr != "" {
		c.addrsVerifiedContract.Store(tokenAddr, contractAddr)
	}
}

func (c *Cache) storeBalance(id string) {
	c.addrsBalance.Store(id, true)
}

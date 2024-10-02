package indexer

import (
	"sync"

	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
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
	for _, tokenAddr := range idxer.contractVerifyWhitelist {
		idxer.addWhitelist(doc.ConvWhitelist(tokenAddr, "", "contract"))
	}
	for _, balanceAddr := range idxer.balanceWhitelist {
		cache.storeBalance(balanceAddr)
	}
	return cache
}

// register to white list.
func (c *Cache) registerVariables() {
	// register verify token
	if err := c.idxer.ScrollToken(func(tokenDoc *doc.EsToken) {
		if tokenDoc.TokenAddress != "" {
			c.idxer.addWhitelist(doc.ConvWhitelist(tokenDoc.TokenAddress, tokenDoc.Id, "token"))
		}
	}); err != nil {
		c.idxer.log.Error().Err(err).Str("func", "registerVariables").Msg("error while scroll token")
	}

	// register verify contract
	if err := c.idxer.ScrollContract(func(contractDoc *doc.EsContract) {
		if contractDoc.VerifiedToken != "" {
			c.idxer.addWhitelist(doc.ConvWhitelist(contractDoc.VerifiedToken, contractDoc.Id, "contract"))
		}
	}); err != nil {
		c.idxer.log.Error().Err(err).Str("func", "registerVariables").Msg("error while scroll contract")
	}

	// register balances
	if err := c.idxer.ScrollBalance(func(balanceDoc *doc.EsAccountBalance) {
		c.storeBalance(balanceDoc.Id)
	}); err != nil {
		c.idxer.log.Error().Err(err).Str("func", "registerVariables").Msg("error while scroll balance")
	}
}

func (ns *Cache) refreshVariables(info BlockInfo, blockDoc *doc.EsBlock, minerGRPC *client.AergoClientController) {
	mapWhitelist := make(map[string][2]string)

	// update verify token, contract
	ns.idxer.ScrollWhitelist(func(whitelistDoc *doc.EsWhitelist) {
		var updateContractAddress string
		if whitelistDoc.Type == "token" {
			metadata := minerGRPC.QueryMetadataOf(ns.idxer.tokenVerifyAddr, whitelistDoc.Id)
			ns.idxer.log.Info().Str("tokenAddress", whitelistDoc.Id).Msg("update verified token")
			updateContractAddress = ns.idxer.MinerTokenVerified(whitelistDoc.Id, whitelistDoc.Contract, metadata, minerGRPC)
		}
		if whitelistDoc.Type == "contract" {
			metadata := minerGRPC.QueryMetadataOf(ns.idxer.contractVerifyAddr, whitelistDoc.Id)
			ns.idxer.log.Info().Str("tokenAddress", whitelistDoc.Id).Msg("update verified contract")
			updateContractAddress = ns.idxer.MinerContractVerified(whitelistDoc.Id, whitelistDoc.Contract, metadata, minerGRPC)
		}
		// contract 변경 시 갱신
		if whitelistDoc.Contract != updateContractAddress {
			mapWhitelist[whitelistDoc.Id] = [2]string{updateContractAddress, whitelistDoc.Type}
		}
	})
	// refresh verify token, contract
	for tokenAddr, contractAddr := range mapWhitelist {
		ns.idxer.addWhitelist(doc.ConvWhitelist(tokenAddr, contractAddr[0], contractAddr[1]))
	}

	// update whitelist balance
	mapBalance := make(map[string]string)
	ns.addrsBalance.Range(func(k, v interface{}) bool {
		if addr, ok := k.(string); ok {
			if stakeExists := ns.idxer.MinerBalance(blockDoc, addr, minerGRPC); stakeExists == false {
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

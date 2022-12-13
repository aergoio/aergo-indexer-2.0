package indexer

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aergoio/aergo-indexer-2.0/indexer/category"
	"github.com/aergoio/aergo-indexer-2.0/indexer/client"
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/aergoio/aergo-indexer-2.0/indexer/transaction"
	"github.com/aergoio/aergo-indexer-2.0/types"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mr-tron/base58/base58"
)

// ConvBlock converts Block from RPC into Elasticsearch type - 1.0
func (ns *Indexer) ConvBlock(block *types.Block) doc.EsBlock {
	rewardAmount := ""
	if len(block.Header.Consensus) > 0 {
		rewardAmount = "160000000000000000"
	}
	return doc.EsBlock{
		BaseEsType:    &doc.BaseEsType{Id: base58.Encode(block.Hash)},
		Timestamp:     time.Unix(0, block.Header.Timestamp),
		BlockNo:       block.Header.BlockNo,
		TxCount:       uint(len(block.Body.Txs)),
		Size:          uint64(proto.Size(block)),
		BlockProducer: ns.makePeerId(block.Header.PubKey),
		RewardAccount: ns.encodeAndResolveAccount(block.Header.Consensus, block.Header.BlockNo),
		RewardAmount:  rewardAmount,
	}
}

func (ns *Indexer) makePeerId(pubKey []byte) string {
	if peerId, is_ok := ns.peerId.Load(string(pubKey)); is_ok == true {
		return peerId.(string)
	}
	cryptoPubKey, err := crypto.UnmarshalPublicKey(pubKey)
	if err != nil {
		return ""
	}
	Id, err := peer.IDFromPublicKey(cryptoPubKey)
	if err != nil {
		return ""
	}
	peerId := peer.IDB58Encode(Id)
	ns.peerId.Store(string(pubKey), peerId)
	return peerId
}

// Internal names refer to special accounts that don't need to be resolved
func isInternalName(name string) bool {
	switch name {
	case
		"aergo.name",
		"aergo.system",
		"aergo.enterprise",
		"aergo.vault":
		return true
	}
	return false
}

func encodeAccount(account []byte) string {
	if account == nil {
		return ""
	}
	if len(account) <= 12 || isInternalName(string(account)) {
		return string(account)
	}
	return types.EncodeAddress(account)
}

func (ns *Indexer) encodeAndResolveAccount(account []byte, blockNo uint64) string {
	var encoded = encodeAccount(account)
	// Seo
	return encoded
	/*
		if len(encoded) > 12 || isInternalName(encoded) || encoded == "" {
			return encoded
		}

		// Resolve name
		var nameRequest = &types.Name{
			Name:    encoded,
			BlockNo: blockNo,
		}

		ctx := context.Background()
		nameInfo, err := ns.grpcClient.GetNameInfo(ctx, nameRequest)

		if err != nil {
			return "UNRESOLVED: " + encoded
		}

		return encodeAccount(nameInfo.GetDestination())
	*/
}

// bigIntToFloat takes a big.Int, divides it by 10^exp and returns the resulting float
// Note that this float is not precise. It can be used for sorting purposes
func bigIntToFloat(a *big.Int, exp int64) float32 {
	var y, e = big.NewInt(10), big.NewInt(exp)
	y.Exp(y, e, nil)
	z := new(big.Float).Quo(
		new(big.Float).SetInt(a),
		new(big.Float).SetInt(y),
	)
	f, _ := z.Float32()
	return f
}

// ConvTx converts Tx from RPC into Elasticsearch type
func (ns *Indexer) ConvTx(tx *types.Tx, blockD doc.EsBlock) doc.EsTx {
	amount := big.NewInt(0).SetBytes(tx.GetBody().Amount)
	category, method := category.DetectTxCategory(tx)
	if len(method) > 50 {
		method = method[:50]
	}
	document := doc.EsTx{
		BaseEsType:     &doc.BaseEsType{Id: base58.Encode(tx.Hash)},
		Account:        ns.encodeAndResolveAccount(tx.Body.Account, blockD.BlockNo),
		Recipient:      ns.encodeAndResolveAccount(tx.Body.Recipient, blockD.BlockNo),
		Amount:         amount.String(),
		AmountFloat:    bigIntToFloat(amount, 18),
		Type:           fmt.Sprintf("%d", tx.Body.Type),
		Category:       category,
		Method:         method,
		Timestamp:      blockD.Timestamp,
		BlockNo:        blockD.BlockNo,
		TokenTransfers: 0,
	}
	return document
}

// ConvName parses a name transaction into Elasticsearch type
func (ns *Indexer) ConvName(tx *types.Tx, blockNo uint64) doc.EsName {
	var name = "error"
	var address string

	payload, err := transaction.UnmarshalPayloadWithArgs(tx)
	if err == nil {
		name = payload.Args[0]
		if payload.Name == "v1createName" {
			address = ns.encodeAndResolveAccount(tx.Body.Account, blockNo)
		}
		if payload.Name == "v1updateName" {
			address = payload.Args[1]
		}
	}
	hash := base58.Encode(tx.Hash)

	document := doc.EsName{
		BaseEsType: &doc.BaseEsType{Id: fmt.Sprintf("%s-%s", name, hash)},
		Name:       name,
		Address:    address,
		UpdateTx:   hash,
		BlockNo:    blockNo,
	}
	return document
}

func (ns *Indexer) ConvNFT(contractAddress []byte, ttDoc doc.EsTokenTransfer, account string, tokenUri string, imageUrl string) doc.EsNFT {
	document := doc.EsNFT{
		BaseEsType:   &doc.BaseEsType{Id: fmt.Sprintf("%s-%s", ttDoc.TokenAddress, ttDoc.TokenId)},
		TokenAddress: ttDoc.TokenAddress,
		TokenId:      ttDoc.TokenId,
		Timestamp:    ttDoc.Timestamp,
		BlockNo:      ttDoc.BlockNo,
		Account:      account,
		TokenUri:     tokenUri,
		ImageUrl:     imageUrl,
	}

	return document
}

func (ns *Indexer) UpdateNFT(Type BlockType, contractAddress []byte, tokenTransfer doc.EsTokenTransfer, grpcc *client.AergoClientController) {
	tokenUri, imageUrl := grpcc.QueryNFTMetadata(contractAddress, tokenTransfer.TokenId)

	// ARC2.tokenTransfer.Amount --> nft.Account (ownerOf)
	nft := ns.ConvNFT(contractAddress, tokenTransfer, tokenTransfer.Amount, tokenUri, imageUrl)
	ns.rec_NFT(Type, nft)
}

func (ns *Indexer) UpdateAccountTokens(Type BlockType, contractAddress []byte, tokenTransfer doc.EsTokenTransfer, account string, grpcc *client.AergoClientController) {
	id := fmt.Sprintf("%s-%s", account, tokenTransfer.TokenAddress)
	if Type == 1 {

		if _, ok := ns.accToken.Load(id); ok {
			// fmt.Println("succs", fmt.Sprintf("%s-%s", account, tokenTransfer.TokenAddress))
			return
		} else {
			// fmt.Println("fail", fmt.Sprintf("%s-%s", account, tokenTransfer.TokenAddress))
			aTokens := ns.ConvAccountTokens(contractAddress, tokenTransfer, account, id, grpcc)
			ns.BChannel.AccTokens <- ChanInfo{1, aTokens}
			ns.accToken.Store(id, true)
		}
	} else {
		aTokens := ns.ConvAccountTokens(contractAddress, tokenTransfer, account, id, grpcc)
		ns.db.Insert(aTokens, ns.indexNamePrefix+"account_tokens")
	}
}

func (ns *Indexer) ConvAccountTokens(contractAddress []byte, ttDoc doc.EsTokenTransfer, account string, id string, grpcc *client.AergoClientController) doc.EsAccountTokens {
	document := doc.EsAccountTokens{
		BaseEsType:   &doc.BaseEsType{Id: id},
		Account:      account,
		TokenAddress: ttDoc.TokenAddress,
		Timestamp:    ttDoc.Timestamp,
	}

	if ttDoc.TokenId == "" {
		document.Type = category.ARC1
	} else {
		document.Type = category.ARC2
	}

	document.Balance, document.BalanceFloat = grpcc.QueryBalanceOf(contractAddress, account, bytes.Equal(contractAddress, cccv_nft_address))
	return document
}

func (ns *Indexer) ConvTokenTransfer(contractAddress []byte, txDoc doc.EsTx, idx int, from string, to string, args interface{}, grpcc *client.AergoClientController) doc.EsTokenTransfer {
	document := doc.EsTokenTransfer{
		BaseEsType:   &doc.BaseEsType{Id: fmt.Sprintf("%s-%d", txDoc.Id, idx)},
		TxId:         txDoc.GetID(),
		BlockNo:      txDoc.BlockNo,
		Timestamp:    txDoc.Timestamp,
		TokenAddress: ns.encodeAndResolveAccount(contractAddress, txDoc.BlockNo),
		Sender:       txDoc.Account,
		From:         from,
		To:           to,
	}

	if data, ok := args.(string); ok {
		document.TokenId, document.Amount, document.AmountFloat = grpcc.QueryOwnerOf(contractAddress, data, bytes.Equal(contractAddress, cccv_nft_address))
	} else {
		document.Amount = ""
	}
	return document
}

func (ns *Indexer) UpdateToken(contractAddress []byte, grpcc *client.AergoClientController) {
	supply, supplyFloat := grpcc.QueryTotalSupply(contractAddress, bytes.Equal(contractAddress, cccv_nft_address))
	document := doc.EsTokenUp{
		Supply:      supply,
		SupplyFloat: supplyFloat,
	}

	ns.db.Update(document, ns.indexNamePrefix+"token", encodeAccount(contractAddress))
}

// ConvContractCreateTx creates document for token creation
func (ns *Indexer) ConvContract(txDoc doc.EsTx, contractAddress []byte) doc.EsContract {
	document := doc.EsContract{
		BaseEsType: &doc.BaseEsType{Id: ns.encodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		Creator:    txDoc.Account,
		TxId:       txDoc.GetID(),
		BlockNo:    txDoc.BlockNo,
		Timestamp:  txDoc.Timestamp,
	}
	return document
}

// ConvContractCreateTx creates document for token creation
func (ns *Indexer) ConvToken(txDoc doc.EsTx, contractAddress []byte, grpcc *client.AergoClientController) doc.EsToken {
	name, symbol, decimals := grpcc.QueryTokenInfo(contractAddress)
	supply, supplyFloat := grpcc.QueryTotalSupply(contractAddress, bytes.Equal(contractAddress, cccv_nft_address))

	document := doc.EsToken{
		BaseEsType:     &doc.BaseEsType{Id: ns.encodeAndResolveAccount(contractAddress, txDoc.BlockNo)},
		TxId:           txDoc.GetID(),
		BlockNo:        txDoc.BlockNo,
		TokenTransfers: uint64(0),
		Name:           name,
		Name_lower:     strings.ToLower(name),
		Symbol:         symbol,
		Symbol_lower:   strings.ToLower(symbol),
		Decimals:       decimals,
		Supply:         supply,
		SupplyFloat:    supplyFloat,
	}
	return document
}

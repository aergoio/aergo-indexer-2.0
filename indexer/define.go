package indexer

import doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"

// bulk type
type ChanType uint

const (
	ChanType_StopBulk ChanType = iota
	ChanType_Add
	ChanType_Commit
)

type ChanInfo struct {
	Type ChanType // 0:stop_bulk, 1:add, 2:commit
	Doc  doc.DocType
}

type BlockType uint

const (
	BlockType_StopMiner BlockType = iota
	BlockType_Bulk
	BlockType_Sync
)

type BlockInfo struct {
	Type   BlockType // 0:stop_miner, 1:bulk, 2:sync
	Height uint64
}

type ChanInfoType struct {
	Block         chan ChanInfo
	Tx            chan ChanInfo
	Event         chan ChanInfo
	Contract      chan ChanInfo
	TokenTransfer chan ChanInfo
	AccTokens     chan ChanInfo
}

type VerifiedStatus string

const (
	NotVerified VerifiedStatus = ""
	Verified    VerifiedStatus = "verified"
)

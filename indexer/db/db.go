package db

import (
	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
)

type DbController interface {
	Insert(document doc.DocType, indexName string) error
	InsertBulk(indexName string) BulkInstance
	Update(document doc.DocType, indexName string, id string) error
	Delete(params QueryParams) (uint64, error)
	Count(params QueryParams) (int64, error)
	SelectOne(params QueryParams, createDocument CreateDocFunction) (doc.DocType, error)
	Scroll(params QueryParams, createDocument CreateDocFunction) ScrollInstance
	GetExistingIndexPrefix(aliasName string, documentType string) (bool, string, error)
	CreateIndex(indexName string, documentType string) error
	UpdateAlias(aliasName string, indexName string) error
}

type IntegerRangeQuery struct {
	Field string
	Min   uint64
	Max   uint64
}

type StringMatchQuery struct {
	Field string
	Value string
}

type QueryParams struct {
	IndexName    string
	TypeName     string
	From         int
	To           int
	Size         int
	SortField    string
	SortAsc      bool
	SelectFields []string
	IntegerRange *IntegerRangeQuery
	StringMatch  *StringMatchQuery
}

type CreateDocFunction = func() doc.DocType

type ScrollInstance interface {
	Next() (doc.DocType, error)
}

type BulkInstance interface {
	Add(document doc.DocType)
	Commit() error
}

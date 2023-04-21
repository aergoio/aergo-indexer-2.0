package db

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/olivere/elastic/v7"
)

// ElasticsearchDbController implements DbController
type ElasticsearchDbController struct {
	client *elastic.Client
}

// NewElasticClient creates a new instance of elastic.Client
func NewElasticClient(esURL string) (*elastic.Client, error) {
	url := esURL
	if !strings.HasPrefix(url, "http") {
		url = fmt.Sprintf("http://%s", url)
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}
	client, err := elastic.NewClient(
		elastic.SetHttpClient(httpClient),
		elastic.SetURL(url),
		elastic.SetHealthcheckTimeoutStartup(30*time.Second),
		elastic.SetSniff(false),
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewElasticsearchDbController creates a new instance of ElasticsearchDbController
func NewElasticsearchDbController(esURL string) (*ElasticsearchDbController, error) {
	client, err := NewElasticClient(esURL)
	if err != nil {
		return nil, err
	}
	return &ElasticsearchDbController{client: client}, nil
}

func (esdb *ElasticsearchDbController) Exists(indexName string, id string) bool {
	ans, _ := esdb.client.Exists().Index(indexName).Id(id).Do(context.Background())
	return ans
}

func (esdb *ElasticsearchDbController) Update(document doc.DocType, indexName string, id string) error {
	_, err := esdb.client.Update().Index(indexName).Id(id).Doc(document).Upsert(document).Do(context.Background())
	if errConflict, ok := err.(*elastic.Error); ok && errConflict.Status == 409 {
		return nil // ignore version conflict exception
	}
	return err
}

// Insert inserts a single document using the updata params
// It returns the number of inserted documents (1) or an error
func (esdb *ElasticsearchDbController) Insert(document doc.DocType, indexName string) error {
	_, err := esdb.client.Index().Index(indexName).OpType("index").Id(document.GetID()).BodyJson(document).Do(context.Background())
	return err
}

// Delete removes documents specified by the query params
func (esdb *ElasticsearchDbController) Delete(params QueryParams) (uint64, error) {
	var query elastic.Query
	if params.IntegerRange != nil {
		query = elastic.NewRangeQuery(params.IntegerRange.Field).From(params.IntegerRange.Min).To(params.IntegerRange.Max)
	} else if params.StringMatch != nil {
		query = elastic.NewMatchQuery(params.StringMatch.Field, params.StringMatch.Value)
	}

	res, err := esdb.client.DeleteByQuery().Index(params.IndexName).Query(query).Do(context.Background())
	if err != nil {
		return 0, err
	}
	return uint64(res.Deleted), nil
}

// Count returns the number of indexed documents
func (esdb *ElasticsearchDbController) Count(params QueryParams) (int64, error) {
	return esdb.client.Count(params.IndexName).Do(context.Background())
}

// SelectOne selects a single document
func (esdb *ElasticsearchDbController) SelectOne(params QueryParams, createDocument CreateDocFunction) (doc.DocType, error) {
	service := esdb.client.Search().Index(params.IndexName)
	if params.IntegerRange != nil {
		query := elastic.NewRangeQuery(params.IntegerRange.Field).From(params.IntegerRange.Min).To(params.IntegerRange.Max)
		service = service.Query(query)
	}
	if params.StringMatch != nil {
		query := elastic.NewMatchQuery(params.StringMatch.Field, params.StringMatch.Value)
		service = service.Query(query)
	}
	if params.SortField != "" {
		service = service.Sort(params.SortField, params.SortAsc).From(params.From)
	}

	res, err := service.Size(1).Do(context.Background())
	if err != nil {
		return nil, err
	}
	if res == nil || res.TotalHits() == 0 || len(res.Hits.Hits) == 0 {
		return nil, nil
	}

	// Unmarshall document
	hit := res.Hits.Hits[0]
	document := createDocument()
	if err := json.Unmarshal([]byte(hit.Source), document); err != nil {
		return nil, err
	}
	document.SetID(hit.Id)
	return document, nil
}

// UpdateAlias updates an alias with a new index name and delete stale indices
func (esdb *ElasticsearchDbController) UpdateAlias(aliasName string, indexName string) error {
	ctx := context.Background()
	svc := esdb.client.Alias()
	res, err := esdb.client.Aliases().Index("_all").Do(ctx)
	if err != nil {
		return err
	}

	indices := res.IndicesByAlias(aliasName)
	if len(indices) > 0 {
		// Remove old aliases
		for _, indexName := range indices {
			svc.Remove(indexName, aliasName)
		}
	}

	// Add new alias
	svc.Add(indexName, aliasName)

	_, err = svc.Do(ctx)
	for _, indexName := range indices {
		esdb.client.DeleteIndex(indexName).Do(ctx)
	}
	return err
}

// GetExistingIndexPrefix checks for existing indices and returns the prefix, if any
func (esdb *ElasticsearchDbController) GetExistingIndexPrefix(aliasName string, documentType string) (bool, string, error) {
	res, err := esdb.client.Aliases().Index("_all").Do(context.Background())
	if err != nil {
		return false, "", err
	}
	indices := res.IndicesByAlias(aliasName)
	if len(indices) > 0 {
		indexNamePrefix := strings.TrimSuffix(indices[0], documentType)
		return true, indexNamePrefix, nil
	}
	return false, "", nil
}

// CreateIndex creates index according to documentType definition
func (esdb *ElasticsearchDbController) CreateIndex(indexName string, documentType string) error {
	createIndex, err := esdb.client.CreateIndex(indexName).BodyString(doc.EsMappings[documentType]).Do(context.Background())
	if err != nil {
		return err
	}
	if !createIndex.Acknowledged {
		return errors.New("CreateIndex not acknowledged")
	}
	return nil
}

// Scroll creates a new scroll instance with the specified query and unmarshal function
func (esdb *ElasticsearchDbController) Scroll(params QueryParams, createDocument CreateDocFunction) ScrollInstance {
	fsc := elastic.NewFetchSourceContext(true).Include(params.SelectFields...)

	// seo
	scroll := esdb.client.Scroll(params.IndexName)
	if params.SortField != "" {
		query := elastic.NewRangeQuery(params.SortField)
		if params.From != 0 {
			query = query.From(params.From)
		}
		if params.To != 0 {
			query = query.To(params.To)
		}
		scroll = scroll.Query(query)
	}
	scroll = scroll.Size(params.Size).Sort(params.SortField, params.SortAsc).FetchSourceContext(fsc)
	return &EsScrollInstance{
		scrollService:  scroll,
		ctx:            context.Background(),
		createDocument: createDocument,
	}
}

// EsScrollInstance is an instance of a scroll for ES
type EsScrollInstance struct {
	scrollService  *elastic.ScrollService
	result         *elastic.SearchResult
	current        int
	currentLength  int
	ctx            context.Context
	createDocument CreateDocFunction
}

// Next returns the next document of a scroll or io.EOF
func (scroll *EsScrollInstance) Next() (doc.DocType, error) {
	// Load next part of scroll
	if scroll.result == nil || scroll.current >= scroll.currentLength {
		result, err := scroll.scrollService.Do(scroll.ctx)
		if err != nil {
			return nil, err // returns io.EOF when scroll is done
		}
		scroll.result = result
		scroll.current = 0
		scroll.currentLength = len(result.Hits.Hits)
	}

	// Return next document
	if scroll.current < scroll.currentLength {
		doc := scroll.result.Hits.Hits[scroll.current]
		scroll.current++

		unmarshalled := scroll.createDocument()
		// seo
		if err := json.Unmarshal([]byte(doc.Source), unmarshalled); err != nil {
			// if err := json.Unmarshal(*doc.Source, unmarshalled); err != nil {
			return nil, err
		}
		unmarshalled.SetID(doc.Id)
		return unmarshalled, nil
	}
	return nil, io.EOF
}

func (esdb *ElasticsearchDbController) InsertBulk(indexName string) BulkInstance {
	return &EsBulkInstance{
		bulk: esdb.client.Bulk().Index(indexName),
		ctx:  context.Background(),
	}
}

type EsBulkInstance struct {
	bulk *elastic.BulkService
	ctx  context.Context
}

func (bulk *EsBulkInstance) Add(document doc.DocType) {
	req := elastic.NewBulkIndexRequest().OpType("create").Id(document.GetID()).Doc(document)
	// req := elastic.NewBulkUpdateRequest().Id(document.GetID()).Doc(document).DocAsUpsert(true)
	bulk.bulk.Add(req)
}

func (bulk *EsBulkInstance) Commit() error {
	_, err := bulk.bulk.Do(bulk.ctx)
	return err
}

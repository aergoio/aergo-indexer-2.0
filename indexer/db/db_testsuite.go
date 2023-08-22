package db

import (
	"strings"
	"testing"
	"time"

	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/stretchr/testify/require"
)

// there are common tests for all db implementations:
//
//	test 1. Index  = CreateIndexOld - UpdateAlias - GetExistingIndexPrefix - CreateIndexNew - UpdateAlias - GetExistingIndexPrefix(old -> new)
//	test 2. Count  = Insert - Count - Delete - Count
//	test 3. Select = Insert - SelectOne - Update - SelectOne
//	test 4. Scroll = Insert - Scroll
//	test 5. Bulk   = Bulk - Count
func TestDatabaseSuite(t *testing.T, New func() DbController) {
	t.Run("Index", func(t *testing.T) {
		tests := []struct {
			idxOld  string
			idxNew  string
			docType string
			alias   string
		}{
			{"idx_old_block", "idx_new_block", "block", "alias_block"},
			{"idx_old_tx", "idx_new_tx", "tx", "alias_tx"},
			{"idx_old_name", "idx_new_name", "name", "alias_name"},
			{"idx_old_token", "idx_new_token", "token", "alias_token"},
			{"idx_old_token_transfer", "idx_new_token_transfer", "token_transfer", "alias_token_transfer"},
			{"idx_old_account_tokens", "idx_new_account_tokens", "account_tokens", "alias_account_tokens"},
			{"idx_old_nft", "idx_new_nft", "nft", "alias_nft"},
			{"idx_old_contract", "idx_new_contract", "contract", "alias_contract"},
		}
		for i, test := range tests {
			db := New()
			err := db.CreateIndex(test.idxOld, test.docType)
			require.NoErrorf(t, err, "error in [Index] [test %d]", i)

			err = db.UpdateAlias(test.alias, test.idxOld)
			require.NoErrorf(t, err, "error in [Index] [test %d]", i)

			exists, idxNamePrefix, err := db.GetExistingIndexPrefix(test.alias, test.docType)
			require.NoErrorf(t, err, "error in [Index] [test %d]", i)
			require.Truef(t, exists, "error in [Index] [test %d]", i)
			require.Equalf(t, strings.TrimSuffix(test.idxOld, test.docType), idxNamePrefix, "error in [Index] [test %d]", i)

			err = db.CreateIndex(test.idxNew, test.docType)
			require.NoErrorf(t, err, "error in [Index] [test %d]", i)

			err = db.UpdateAlias(test.alias, test.idxNew)
			require.NoErrorf(t, err, "error in [Index] [test %d]", i)

			exists, idxNamePrefix, err = db.GetExistingIndexPrefix(test.alias, test.docType)
			require.NoErrorf(t, err, "error in [Index] [test %d]", i)
			require.Truef(t, exists, "error in [Index] [test %d]", i)
			require.Equalf(t, strings.TrimSuffix(test.idxNew, test.docType), idxNamePrefix, "error in [Index] [test %d]", i)
			require.NotEqualf(t, strings.TrimSuffix(test.idxOld, test.docType), idxNamePrefix, "error in [Index] [test %d]", i)
		}
	})

	t.Run("Count", func(t *testing.T) {
		tests := []struct {
			idxName      string
			aliasName    string
			docType      string
			data         doc.DocType
			integerRange *IntegerRangeQuery
			stringRange  *StringMatchQuery
		}{
			{
				"idx_count_block", "alias_count_block", "block", &doc.EsBlock{
					BaseEsType: &doc.BaseEsType{Id: "31SwcH9K5BxahtRQt1pC4UyboaXYWvnWSovvpmeneuu3"},
					BlockNo:    uint64(1),
				}, &IntegerRangeQuery{Field: "no", Min: 0, Max: 1}, nil,
			},
			{
				"idx_count_block", "alias_count_block", "block", &doc.EsBlock{
					BaseEsType:    &doc.BaseEsType{Id: "31SwcH9K5BxahtRQt1pC4UyboaXYWvnWSovvpmeneuu3"},
					BlockNo:       uint64(1),
					RewardAccount: "AmgWzwNRgqF1vmvCyZqoR6YKNWSG2JFNHewHvb56mqhXoQcepLiy",
				}, nil, &StringMatchQuery{Field: "reward_account", Value: "AmgWzwNRgqF1vmvCyZqoR6YKNWSG2JFNHewHvb56mqhXoQcepLiy"},
			},
			{
				"idx_count_tx", "alias_count_tx", "tx", &doc.EsTx{
					BaseEsType: &doc.BaseEsType{Id: "5mxrxYHkANW44jffCcwLLTUovfqjDFWBr51YgYg8meQc"},
					BlockNo:    uint64(1),
				}, &IntegerRangeQuery{Field: "blockno", Min: 0, Max: 1}, nil,
			},
			{
				"idx_count_tx", "alias_count_tx", "tx", &doc.EsTx{
					BaseEsType: &doc.BaseEsType{Id: "5mxrxYHkANW44jffCcwLLTUovfqjDFWBr51YgYg8meQc"},
					BlockNo:    uint64(1),
				}, nil, &StringMatchQuery{Field: "blockno", Value: "1"},
			},
		}

		for i, test := range tests {
			db := New()
			err := db.CreateIndex(test.idxName, test.docType)
			require.NoErrorf(t, err, "error in [Count] [test %d]", i)

			err = db.UpdateAlias(test.aliasName, test.idxName)
			require.NoErrorf(t, err, "error in [Count] [test %d]", i)

			err = db.Insert(test.data, test.idxName)
			require.NoErrorf(t, err, "error in [Count] [test %d]", i)

			time.Sleep(time.Second) // sleep 1 sec to refresh index
			count, err := db.Count(QueryParams{IndexName: test.idxName})
			require.NoErrorf(t, err, "error in [Count] [test %d]", i)
			require.Equalf(t, int64(1), count, "error in [Count] [test %d]", i)

			// delete range query
			_, err = db.Delete(QueryParams{IndexName: test.idxName, TypeName: test.docType, IntegerRange: test.integerRange, StringMatch: test.stringRange})
			require.NoErrorf(t, err, "error in [Count] [test %d]", i)

			time.Sleep(time.Second) // sleep 1 sec to refresh index
			count, err = db.Count(QueryParams{IndexName: test.idxName})
			require.NoErrorf(t, err, "error in [Count] [test %d]", i)
			require.Equalf(t, int64(0), count, "error in [Count] [test %d]", i)
		}
	})

	t.Run("Select", func(t *testing.T) {
		tests := []struct {
			idxName   string
			aliasName string
			docType   string
			sortField string
			sortAsc   bool

			docInsert []doc.DocType
			docSelect doc.DocType
			docUpdate doc.DocType
		}{
			{
				"idx_select_block", "alias_select_block", "block", "no", false, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
						BlockNo:    104776384,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					},
				}, &doc.EsBlock{
					BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
					BlockNo:    104776385,
				}, &doc.EsBlock{
					BaseEsType:    &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
					BlockNo:       104776385,
					RewardAccount: "updateRewardAccount",
				},
			},
			{
				"idx_select_block", "alias_select_block", "block", "no", true, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
						BlockNo:    104776384,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					},
				}, &doc.EsBlock{
					BaseEsType: &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
					BlockNo:    104776384,
				}, &doc.EsBlock{
					BaseEsType:    &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
					BlockNo:       104776384,
					RewardAccount: "updateRewardAccount",
				},
			},
		}

		for i, test := range tests {
			db := New()
			err := db.CreateIndex(test.idxName, test.docType)
			require.NoErrorf(t, err, "error in [SelectOne] [test %d]", i)

			err = db.UpdateAlias(test.aliasName, test.idxName)
			require.NoErrorf(t, err, "error in [SelectOne] [test %d]", i)

			for _, data := range test.docInsert {
				err = db.Insert(data, test.idxName)
				require.NoErrorf(t, err, "error in [SelectOne] [test %d]", i)
			}

			time.Sleep(time.Second * 1) // sleep 1 sec to refresh index
			docResult, err := db.SelectOne(QueryParams{IndexName: test.idxName, SortField: test.sortField, SortAsc: test.sortAsc}, func() doc.DocType {
				return getDocType(test.docType)
			})
			require.NoErrorf(t, err, "error in [SelectOne] [test %d]", i)
			require.EqualValuesf(t, test.docSelect, docResult, "error in [SelectOne] [test %d]", i)

			err = db.Update(test.docUpdate, test.idxName, test.docUpdate.GetID())
			require.NoErrorf(t, err, "error in [SelectOne] [test %d]", i)

			time.Sleep(time.Second * 1) // sleep 1 sec to refresh index
			docResult, err = db.SelectOne(QueryParams{IndexName: test.idxName, SortField: test.sortField, SortAsc: test.sortAsc}, func() doc.DocType {
				return getDocType(test.docType)
			})
			require.NoErrorf(t, err, "error in [SelectOne] [test %d]", i)
			require.EqualValuesf(t, test.docUpdate, docResult, "error in [SelectOne] [test %d]", i)
		}
	})

	t.Run("Scroll", func(t *testing.T) {
		tests := []struct {
			idxName   string
			aliasName string
			docType   string
			sortField string
			sortAsc   bool
			from      int
			to        int

			docInsert []doc.DocType
			docScroll []doc.DocType
		}{
			{
				"idx_scroll_block", "alias_scroll_block", "block", "no", false, 104776385, 104776386, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
						BlockNo:    104776384,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "9o5C5ukWrXiNCFSU2Y1XQwcqY1fgTimrnTpnYu2CsG1h"},
						BlockNo:    104776386,
					},
				}, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "9o5C5ukWrXiNCFSU2Y1XQwcqY1fgTimrnTpnYu2CsG1h"},
						BlockNo:    104776386,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					},
				},
			},
			{
				"idx_scroll_block", "alias_scroll_block", "block", "no", true, 104776385, 104776386, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
						BlockNo:    104776384,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "9o5C5ukWrXiNCFSU2Y1XQwcqY1fgTimrnTpnYu2CsG1h"},
						BlockNo:    104776386,
					},
				}, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "9o5C5ukWrXiNCFSU2Y1XQwcqY1fgTimrnTpnYu2CsG1h"},
						BlockNo:    104776386,
					},
				},
			},
		}

		for i, test := range tests {
			db := New()
			err := db.CreateIndex(test.idxName, test.docType)
			require.NoErrorf(t, err, "error in [Scroll] [test %d]", i)

			err = db.UpdateAlias(test.aliasName, test.idxName)
			require.NoErrorf(t, err, "error in [Scroll] [test %d]", i)

			for _, data := range test.docInsert {
				err = db.Insert(data, test.idxName)
				require.NoErrorf(t, err, "error in [Scroll] [test %d]", i)
			}

			time.Sleep(time.Second * 1) // sleep 1 sec to refresh index
			scroll := db.Scroll(QueryParams{IndexName: test.idxName, SortField: test.sortField, SortAsc: test.sortAsc, Size: 100, From: test.from, To: test.to}, func() doc.DocType {
				return getDocType(test.docType)
			})

			for _, expect := range test.docScroll {
				doc, err := scroll.Next()
				require.NoErrorf(t, err, "error in [Scroll] [test %d]", i)
				require.EqualValuesf(t, expect, doc, "error in [Scroll] [test %d]", i)
			}
		}
	})

	t.Run("Bulk", func(t *testing.T) {
		tests := []struct {
			idxName   string
			aliasName string
			docType   string
			sortField string
			sortAsc   bool
			from      int
			to        int

			docInsert []doc.DocType
			count     int
		}{
			{
				"idx_bulk_block", "alias_bulk_block", "block", "no", false, 104776385, 104776386, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
						BlockNo:    104776384,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "9o5C5ukWrXiNCFSU2Y1XQwcqY1fgTimrnTpnYu2CsG1h"},
						BlockNo:    104776386,
					},
				}, 3,
			},
			{
				"idx_bulk_block", "alias_bulk_block", "block", "no", true, 104776385, 104776386, []doc.DocType{
					&doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "FpzaEXtEab9g99fhAF7C6XhUUJAc6TjJTjvpWNkdqAn1"},
						BlockNo:    104776384,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "E8Lbg1B6QRa8VfFLPLrsW99vrTs1fYW5VhQaPQAFrT3R"},
						BlockNo:    104776385,
					}, &doc.EsBlock{
						BaseEsType: &doc.BaseEsType{Id: "9o5C5ukWrXiNCFSU2Y1XQwcqY1fgTimrnTpnYu2CsG1h"},
						BlockNo:    104776386,
					},
				}, 3,
			},
		}

		for i, test := range tests {
			db := New()
			err := db.CreateIndex(test.idxName, test.docType)
			require.NoErrorf(t, err, "error in [Bulk] [test %d]", i)

			err = db.UpdateAlias(test.aliasName, test.idxName)
			require.NoErrorf(t, err, "error in [Bulk] [test %d]", i)

			bulk := db.InsertBulk(test.idxName)
			for _, data := range test.docInsert {
				bulk.Add(data)
			}
			err = bulk.Commit()
			require.NoErrorf(t, err, "error in [Bulk] [test %d]", i)

			time.Sleep(time.Second * 1) // sleep 1 sec to refresh index
			count, err := db.Count(QueryParams{IndexName: test.idxName})
			require.NoErrorf(t, err, "error in [Bulk] [test %d]", i)
			require.EqualValuesf(t, test.count, count, "error in [Bulk] [test %d]", i)
		}
	})

}

func getDocType(docType string) doc.DocType {
	switch docType {
	case "block":
		return &doc.EsBlock{BaseEsType: &doc.BaseEsType{}}
	case "name":
		return &doc.EsTx{BaseEsType: &doc.BaseEsType{}}
	case "tx":
		return &doc.EsTx{BaseEsType: &doc.BaseEsType{}}
	case "token":
		return &doc.EsToken{BaseEsType: &doc.BaseEsType{}}
	case "token_transfer":
		return &doc.EsTokenTransfer{BaseEsType: &doc.BaseEsType{}}
	case "account_tokens":
		return &doc.EsAccountTokens{BaseEsType: &doc.BaseEsType{}}
	case "nft":
		return &doc.EsNFT{BaseEsType: &doc.BaseEsType{}}
	case "contract":
		return &doc.EsContract{BaseEsType: &doc.BaseEsType{}}
	}
	return nil
}

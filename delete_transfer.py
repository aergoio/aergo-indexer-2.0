from elasticsearch import Elasticsearch
from elasticsearch_dsl import Search
import sys

es = Elasticsearch('localhost:9200')

prefix = sys.argv[1]

s = Search(using=es, index=prefix+'_token')
s.execute()
token_list = [h.meta.id for h in s.scan()]


print("delete _tokens_transfer")
s = Search(using=es, index=prefix+'_token_transfer')
s.aggs.bucket('tx', 'terms', field='address', size=10000)
tx = s.execute()

for x in tx.aggregations.tx.buckets:
    if x.key not in token_list :
        print("delete token tx", x.doc_count, x.key)
        es.delete_by_query(index=prefix+'_token_transfer', body={"query":{"match": {"address": x.key}}})


print("delete _account_tokens")
s = Search(using=es, index=prefix+'_account_tokens')
s.aggs.bucket('tx', 'terms', field='address', size=10000)
tx = s.execute()

for x in tx.aggregations.tx.buckets:
    if x.key not in token_list :
        print("delete account token", x.doc_count, x.key)
        es.delete_by_query(index=prefix+'_account_tokens', body={"query":{"match": {"address": x.key}}})


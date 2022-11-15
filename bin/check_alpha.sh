echo "Starting indexer"

AERGO_URL="alpha-api.aergo.io:7845"
ES_URL="http://localhost:9200"
INDEX_PREFIX="alpha_"
SYNC_FROM=10000
SYNC_TO=20000
MINER=32
BULK=4000
BATCH=60
GRPC=16

/home/bin/indexer  -A $AERGO_URL --dburl $ES_URL --prefix $INDEX_PREFIX --from $SYNC_FROM --to $SYNC_TO --bulk $BULK --batch $BATCH --miner $MINER --grpc $GRPC --check true

/home/bin/clean_index $INDEX_PREFIX

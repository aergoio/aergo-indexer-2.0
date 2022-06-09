echo "Starting indexer"

AERGO_URL="mainnet-api.aergo.io:7845"
#AERGO_URL="mainnet-node3.aergo.io:7845"
#AERGO_URL="218.147.120.149:7845"
ES_URL="http://localhost:9200"
CHAIN_PREFIX="mainnet_"
SYNC_TO=95000000
SYNC_FROM=94000000
MINER=32
BULK=4000
BATCH=60
GRPC=8

./bin/indexer_single  -A $AERGO_URL --dburl $ES_URL --prefix $CHAIN_PREFIX --from $SYNC_FROM --to $SYNC_TO --bulk $BULK --batch $BATCH --miner $MINER --grpc $GRPC --check true


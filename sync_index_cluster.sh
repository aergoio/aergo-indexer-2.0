echo "Starting indexer"

#AERGO_URL="mainnet-api.aergo.io:7845"
ES_URL="http://localhost:9200"
#CHAIN_PREFIX="main_"
SYNC_FROM=0
SYNC_TO=0
MINER=4
BULK=500
BATCH=5
GRPC=8

/home/bin/indexer_cluster  -A $AERGO_URL --dburl $ES_URL --prefix $CHAIN_PREFIX --from $SYNC_FROM --to $SYNC_TO --bulk $BULK --batch $BATCH --miner $MINER --grpc $GRPC


echo "Starting indexer"

AERGO_URL="testnet-api.aergo.io:7845"
ES_URL="http://localhost:9200"
INDEX_PREFIX="testnet_"
MINER=8
BULK=500
BATCH=10
GRPC=4

/home/bin/indexer  -A $AERGO_URL --dburl $ES_URL --prefix $INDEX_PREFIX --bulk $BULK --batch $BATCH --miner $MINER --grpc $GRPC

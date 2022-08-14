echo "Starting indexer"

AERGO_URL="alpha-api.aergo.io:7845"
ES_URL="http://localhost:9200"
INDEX_PREFIX="alpha_"
MINER=8
BULK=500
BATCH=10
GRPC=4

/home/bin/indexer_single  -A $AERGO_URL --dburl $ES_URL --prefix $INDEX_PREFIX --bulk $BULK --batch $BATCH --miner $MINER --grpc $GRPC 


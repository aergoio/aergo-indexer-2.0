echo "Starting indexer"

#AERGO_URL="mainnet-api.aergo.io:7845"
ES_URL="http://localhost:9200"
#CHAIN_PREFIX="main_"
SYNC_FROM=0
SYNC_TO=0
MINER=32
BULK=4000
BATCH=60
GRPC=16

/home/bin/indexer_single  -A $AERGO_URL --dburl $ES_URL --prefix $CHAIN_PREFIX --from $SYNC_FROM --to $SYNC_TO --bulk $BULK --batch $BATCH --miner $MINER --grpc $GRPC --check true

python3 ./delete_transfer.py 

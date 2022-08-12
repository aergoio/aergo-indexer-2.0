OS=ubuntu:22.04
docker pull $OS
docker rm -f sync_idx
docker run -d -it --name sync_idx --net=host --privileged \
	-v $(pwd):/home \
	$OS /home/bin/indexer_cluster \
	-A "mainnet-node1.aergo.io:7845" \
	--dburl "http://localhost:9200" \
	--prefix ="mainnet_" \
	--bulk 500 \
	--batch 5 \
	--miner 4 \
	--grpc 8

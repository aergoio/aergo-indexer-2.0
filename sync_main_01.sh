docker pull ubuntu:22.04
docker rm -f idx_main_01
docker run -d -it --name idx_main_01 --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="mainnet-node1.aergo.io:7845" \
	-e CHAIN_PREFIX="mainnet_" \
	ubuntu:22.04 bash /home/sync_index_cluster.sh

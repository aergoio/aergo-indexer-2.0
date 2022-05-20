docker pull ubuntu:21.10
docker rm -f idx_main_02
docker run -d -it --name idx_main_02 --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="mainnet-node2.aergo.io:7845" \
	-e CHAIN_PREFIX="main_" \
	ubuntu:21.10 bash /home/sync_index_cluster.sh

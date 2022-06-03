docker pull ubuntu:21.10
docker rm -f idx_main
docker run -d -it --name idx_main --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="218.147.120.149:7845" \
	-e CHAIN_PREFIX="mainnet_" \
	ubuntu:21.10 bash /home/sync_index_single.sh

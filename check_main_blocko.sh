docker pull ubuntu:21.10
docker rm -f check_main
docker run -it --name check_main --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="218.147.120.149:7845" \
	-e CHAIN_PREFIX="mainnet_" \
	ubuntu:21.10 bash /home/check_index_single.sh

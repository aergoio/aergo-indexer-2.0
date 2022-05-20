docker pull ubuntu:21.10
docker rm -f check_main
docker run -it --name check_main --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="mainnet-api.aergo.io:7845" \
	-e CHAIN_PREFIX="main_" \
	ubuntu:21.10 bash /home/check_index_cluster.sh

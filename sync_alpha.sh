docker pull ubuntu:21.10
docker rm -f idx_alpha
docker run -d -it --name idx_alpha --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="alpha-api.aergo.io:7845" \
	-e CHAIN_PREFIX="chain_" \
	ubuntu:21.10 bash /home/sync_index_single.sh

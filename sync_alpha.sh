docker pull ubuntu:22.04
docker rm -f idx_alpha
docker run -d -it --name idx_alpha --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="alpha-api.aergo.io:7845" \
	-e CHAIN_PREFIX="alpha_" \
	ubuntu:22.04 bash /home/sync_index_single.sh

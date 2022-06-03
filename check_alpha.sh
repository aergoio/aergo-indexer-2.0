docker pull ubuntu:21.10
docker rm -f check_alpha
docker run -it --name check_alpha --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="alpha-api.aergo.io:7845" \
	-e CHAIN_PREFIX="alpha_" \
	ubuntu:21.10 bash /home/check_index_single.sh

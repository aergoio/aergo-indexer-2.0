docker pull ubuntu:22.04
docker rm -f check_test
docker run -d -it --name check_test --net=host --privileged \
	-v $(pwd):/home \
	-e AERGO_URL="testnet-api.aergo.io:7845" \
	-e CHAIN_PREFIX="testnet_" \
	ubuntu:22.04 bash /home/check_index_single.sh

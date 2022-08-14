OS=ubuntu:22.04
docker rm -f check_idx
docker run --rm -it --name check_idx --net=host --privileged \
	-v $(pwd):/home \
	$OS /home/bin/check_$1.sh

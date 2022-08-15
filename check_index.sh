OS=ubuntu:22.04
docker rm -f check_idx
docker run --rm -d -it --name check_idx --net=host --privileged \
	-v $(pwd):/home \
	$OS bash /home/bin/check_$1.sh

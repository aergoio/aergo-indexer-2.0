#docker load < ubuntu2204.tar.gz
OS=ubuntu:22.04
docker rm -f check_idx
docker run -it --name check_idx --net=host --privileged \
	-v $(pwd):/home \
	$OS bash /home/check_$1.sh

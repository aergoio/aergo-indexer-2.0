docker load < ubuntu2204.tar.gz
docker rm -f check_idx
docker run -it --name check_idx --net=host --privileged \
	-v $(pwd):/home \
	ubuntu-custom:22.04 bash /home/check_$1.sh

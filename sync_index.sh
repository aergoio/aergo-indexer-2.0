OS=ubuntu:22.04
docker rm -f sync_idx
docker run -d --rm -it --name sync_idx --net=host --privileged \
	-v $(pwd):/home \
	$OS bash /home/bin/sync_$1.sh

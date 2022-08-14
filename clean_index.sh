OS=ubuntu:22.04
docker run --rm -it --name clean_idx --net=host --privileged \
	-v $(pwd):/home \
	$OS /home/bin/clean_index $1_

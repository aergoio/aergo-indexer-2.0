OS=ubuntu:22.04
docker run -it --name clean_idx --net=host --privileged \
	-v $(pwd):/home \
	$OS bash /home/bin/clean_index $1_

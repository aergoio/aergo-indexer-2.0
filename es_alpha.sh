#!/bin/sh

VERSION=${VERSION:=7.15.2}
ELASTIC=${ELASTIC:=docker.elastic.co/elasticsearch/elasticsearch:$VERSION}

echo $ELASTIC
docker rm -f es_node
echo "Starting elasticsearch"
docker pull $ELASTIC
docker run -d --rm -p 9200:9200 -p 9300:9300 --name es_node \
        -v /data/eldata/data:/usr/share/elasticsearch/data \
        -v /data/eldata/logs:/usr/share/elasticsearch/logs \
        -e "discovery.type=single-node"  \
        -e "xpack.security.enabled=false" \
        -e "bootstrap.memory_lock=true" --ulimit memlock=-1:-1 \
        $ELASTIC
VERSION=${VERSION:=7.15.2}
ELASTIC=${ELASTIC:=docker.elastic.co/elasticsearch/elasticsearch:$VERSION}

echo $ELASTIC
docker rm -f es_main_03
echo "Starting elasticsearch es_main_03"
docker pull $ELASTIC
docker run -d --net=host --rm --name es_main_03 \
        -v /data/eldata/data:/usr/share/elasticsearch/data \
        -v /data/eldata/logs:/usr/share/elasticsearch/logs \
        -e cluster.name=es_main_v2  \
        -e node.name=es_main_03_v2  \
        -e node.master=true \
        -e node.data=true  \
        -e network.host=0.0.0.0 \
        -e discovery.seed_hosts=v2-main-scan01,v2-main-scan02\
        -e cluster.initial_master_nodes=es_main_01_v2 \
        -e xpack.security.enabled=false \
        -e bootstrap.memory_lock=true --ulimit memlock=-1:-1 \
        -e "ES_JAVA_OPTS=-Xms24g -Xmx24g" \
        $ELASTIC

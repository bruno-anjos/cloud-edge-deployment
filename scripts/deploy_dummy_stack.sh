#!/bin/bash

REL_PATH="$(dirname "$0")"

docker stop $(docker ps -a -q)
docker rm $(docker ps -a -q)
docker network rm nodes-network

set -e

echo "---------------------------- BUILDING IMAGE ----------------------------"

bash "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/build_dummy_node.sh

ALTERNATIVES_DIR="$CLOUD_EDGE_DEPLOYMENT/build/dummy_node/alternatives"
OPTIONS="--mount type=bind,source=$ALTERNATIVES_DIR,target=/alternatives"

function run() {
	docker run -d --network=nodes-network --name="$HOSTNAME" -p $DEPLOYER_PORT_NUMBER:50002 -p \
		$ARCHIMEDES_PORT_NUMBER:50000 $OPTIONS --hostname "$HOSTNAME" brunoanjos/dummy_node:latest
}

docker system prune -f
docker network create nodes-network

if [ $# -ne 1 ]; then
    echo "usage: deploy_dummy_stack.sh num_nodes"
    exit
fi



for (( c=1; c<=$1; c++ ))
do
	HOSTNAME="dummy$c"
	DEPLOYER_PORT_NUMBER=$((c+30000))
	ARCHIMEDES_PORT_NUMBER=$((c+40000))
	echo "STARTING dummy$c with ports $DEPLOYER_PORT_NUMBER $ARCHIMEDES_PORT_NUMBER"
	run &
done

wait

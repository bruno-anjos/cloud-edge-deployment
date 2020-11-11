#!/bin/bash

REL_PATH="$(dirname "$0")"

function del_everything() {
	docker stop $(docker ps -a -q)
	docker rm $(docker ps -a -q)
	docker network rm nodes-network
}

del_everything &

set -e

echo "---------------------------- BUILDING IMAGE ----------------------------"

bash "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/build_dummy_node.sh

wait

ALTERNATIVES_DIR="$CLOUD_EDGE_DEPLOYMENT/build/dummy_node/alternatives"
OPTIONS="--mount type=bind,source=$ALTERNATIVES_DIR,target=/alternatives"

function run() {
	docker run -d --network=nodes-network --ip "192.168.19$((3+CARRY)).$NODE" --name="$HOSTNAME" $OPTIONS --hostname "$HOSTNAME" brunoanjos/dummy_node:latest
}

docker system prune -f
docker network create --subnet=192.168.192.1/20 nodes-network

if [ $# -ne 1 ]; then
    echo "usage: deploy_dummy_stack.sh num_nodes"
    exit
fi



for (( c=1; c<=$1; c++ ))
do
	HOSTNAME="dummy$c"
	NODE=$((c % 255))
	CARRY=$((c / 255))
	echo "node: $NODE, carry: $CARRY"
	echo "STARTING dummy$c with ports $DEPLOYER_PORT_NUMBER $ARCHIMEDES_PORT_NUMBER"
	run &
done

wait

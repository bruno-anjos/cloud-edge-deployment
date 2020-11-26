#!/bin/bash

if [ $# -ne 1 ]; then
	echo "usage: deploy_dummy_stack.sh num_nodes"
	exit
fi

function del_everything() {
	nodes=$(docker ps -a -q)
	docker stop $nodes
	docker rm $nodes
	docker network rm dummies-network
}

del_everything

set -e

bash "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/build_dummy_node.sh

function run() {
	docker run -d --network=dummies-network --privileged --ip "192.168.19$((3 + CARRY)).$NODE" --name=$DUMMY_NAME \
		--hostname $DUMMY_NAME brunoanjos/dummy_node:latest
}

docker network create --subnet=192.168.192.1/20 dummies-network

for ((c = 1; c <= $1; c++)); do
	DUMMY_NAME="dummy$c"
	NODE=$((c % 255))
	CARRY=$((c / 255))
	echo "node: $NODE, carry: $CARRY"
	echo "STARTING dummy$c with ports $DEPLOYER_PORT_NUMBER $ARCHIMEDES_PORT_NUMBER"
	run &
done

wait

function start() {
	docker exec "$DUMMY_NAME" ./deploy_services.sh
}

for ((c = 1; c <= $1; c++)); do
	DUMMY_NAME="dummy$c"
	echo "LAUNCHING SERVICES in dummy$c"
	start &
done

wait

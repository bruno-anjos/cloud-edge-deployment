#!/bin/bash

REL_PATH="$(dirname "$0")"

bash "$REL_PATH"/remove_stack.sh

set -e

SERVICE_NAME=""
OPTIONS=""
PORT=""

bash "$REL_PATH"/build_images.sh

function run() {
	docker run -d --network=nodes-network --name=$SERVICE_NAME -p $PORT:$PORT \
		$OPTIONS --hostname "$HOSTNAME" brunoanjos/$SERVICE_NAME:latest
}

docker network create nodes-network

SERVICE_NAME="archimedes"
PORT="1500"
run &

SERVICE_NAME="scheduler"
PORT="1501"
OPTIONS="-v /var/run/docker.sock:/var/run/docker.sock"
run &

SERVICE_NAME="deployer"
PORT="1502"
ALTERNATIVES_DIR="$(pwd)/$REL_PATH/../build/deployer/alternatives"
OPTIONS="--mount type=bind,source=$ALTERNATIVES_DIR,target=/alternatives"
run &

SERVICE_NAME="autonomic"
PORT="1503"
OPTIONS=""
run &

wait

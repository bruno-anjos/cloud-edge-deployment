#!/bin/sh

SERVICE_NAME=""
OPTIONS=""
PORT=""

set -e

while :
do
	if ! docker info >/dev/null 2>&1; then
	    echo "Docker does not seem to be running, run it first and retry"
		sleep 2s
	else
		break
	fi
done


HOSTNAME=$(hostname)

./load_images.sh

run() {
	docker run -d --network=host --name=$SERVICE_NAME -p $PORT:$PORT \
		$OPTIONS --hostname "$HOSTNAME" brunoanjos/$SERVICE_NAME:latest
}

SERVICE_NAME="archimedes"
PORT="50000"
run &

SERVICE_NAME="scheduler"
PORT="50001"
OPTIONS="-v /var/run/docker.sock:/var/run/docker.sock"
run &

SERVICE_NAME="deployer"
PORT="50002"
run &

SERVICE_NAME="autonomic"
PORT="50003"
OPTIONS=""
run &

wait
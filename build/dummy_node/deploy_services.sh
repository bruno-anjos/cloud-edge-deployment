#!/bin/sh

SERVICE_NAME=""
OPTIONS=""
PORT=""
DOCKER_IMAGE="brunoanjos/demmon:latest"

set -e

while :; do
  if ! docker info >/dev/null 2>&1; then
    echo "Docker does not seem to be running, run it first and retry"
    sleep 2s
  else
    break
  fi
done

HOSTNAME=$(hostname)

chmod -R 777 /images

./load_images.sh

run() {
  docker run -d --network="bridge" --env NODE_IP="$NODE_IP" --env NODE_ID="$NODE_ID" --env LOCATION="$LOCATION" \
    --name=$SERVICE_NAME -p $PORT:$PORT $OPTIONS --hostname "$HOSTNAME" brunoanjos/$SERVICE_NAME:latest
}

SERVICE_NAME="demmon"
PORT="8090"
docker run -d --cap-add=NET_ADMIN --env NODE_IP="$NODE_IP" --env NODE_ID="$NODE_ID" --env LOCATION="$LOCATION" \
  --network="bridge" -p $PORT:$PORT -p 1200:1200 -p 1300:1300/udp --name=$SERVICE_NAME --hostname "$HOSTNAME" \
  --env LANDMARKS="$LANDMARKS" --env WAIT_FOR_START="$WAIT_FOR_START" "$DOCKER_IMAGE" "$NODE_NUM" "$@"

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

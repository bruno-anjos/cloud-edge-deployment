#!/bin/sh

SERVICE_NAME=""
OPTIONS=""
PORT=""
DOCKER_IMAGE="brunoanjos/demmon:latest"

set -e

START=$(date +%s)
while :; do
  if ! docker info >/dev/null 2>&1; then
    echo "Docker does not seem to be running, run it first and retry"
    sleep 5s
  else
    break
  fi
done
END=$(date +%s)

echo "time took waiting for dockerd: $(($END-$START))sec"

HOSTNAME=$(hostname)

chmod -R 777 /images

START=$(date +%s)
./load_images.sh
END=$(date +%s)
echo "time took loading images: $(($END-$START))sec"

run() {
  docker run -d --network="bridge" --env NODE_IP="$NODE_IP" --env NODE_ID="$NODE_ID" --env LOCATION="$LOCATION" \
    --env NODE_NUM="$NODE_NUM" --name=$SERVICE_NAME -p $PORT:$PORT $OPTIONS --hostname "$HOSTNAME" \
    brunoanjos/$SERVICE_NAME:latest
}

START=$(date +%s)
SERVICE_NAME="demmon"
PORT="8090"
docker run -d --cap-add=NET_ADMIN --env NODE_IP="$NODE_IP" --env NODE_ID="$NODE_ID" --env LOCATION="$LOCATION" \
  --network="bridge" -p $PORT:$PORT -p 1200:1200 -p 1300:1300/udp --name=$SERVICE_NAME --hostname "$HOSTNAME" \
  --env LANDMARKS="$LANDMARKS" --env WAIT_FOR_START="$WAIT_FOR_START" "$DOCKER_IMAGE" "$NODE_NUM" "$@"
END=$(date +%s)
echo "time took starting demmon: $(($END-$START))sec"

START=$(date +%s)
SERVICE_NAME="archimedes"
PORT="1500"
run
END=$(date +%s)
echo "time took starting archimedes: $(($END-$START))sec"

START=$(date +%s)
SERVICE_NAME="scheduler"
PORT="1501"
OPTIONS="-v /var/run/docker.sock:/var/run/docker.sock"
run
END=$(date +%s)
echo "time took starting scheduler: $(($END-$START))sec"

START=$(date +%s)
SERVICE_NAME="deployer"
PORT="1502"
OPTIONS="-v /tables:/tables"
run
END=$(date +%s)
echo "time took starting deployer: $(($END-$START))sec"

START=$(date +%s)
SERVICE_NAME="autonomic"
PORT="1503"
OPTIONS=""
run
END=$(date +%s)
echo "time took starting autonomic: $(($END-$START))sec"

START=$(date +%s)
wait
END=$(date +%s)
echo "time took waiting: $(($END-$START))sec"
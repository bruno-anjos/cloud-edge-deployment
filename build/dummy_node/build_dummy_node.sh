#!/bin/bash

export BUILD_DIR="/tmp/build"

if [[ -z "${CLOUD_EDGE_DEPLOYMENT}" ]]; then
  export CLOUD_EDGE_DEPLOYMENT="/home/b.anjos/go/src/github.com/bruno-anjos/cloud-edge-deployment"
fi

[ ! -d "$BUILD_DIR" ] && mkdir -p "$BUILD_DIR"
[ ! -d /tmp/images ] && mkdir -p /tmp/images
[ ! -d /tmp/bandwidth_stats ] && mkdir -p /tmp/bandwidth_stats

echo "Copying build to /tmp..."

cp -r $CLOUD_EDGE_DEPLOYMENT/build/* $BUILD_DIR/

echo "Removing garbage from previous runs..."

# Clear previous build directories and files
rm -f /tmp/images/*
[ -e "$BUILD_DIR"/dummy_node/fallback.json ] && rm "$BUILD_DIR"/dummy_node/fallback.json
rm -rf "$BUILD_DIR"/dummy_node/deployments
rm -rf "$BUILD_DIR"/dummy_node/images

set -e

echo "Build service binaries and client..."

bash "$BUILD_DIR"/build_binaries.sh &
bash "$BUILD_DIR"/build_client.sh

wait

echo "Build service images..."

bash "$BUILD_DIR"/dummy_node/build_images.sh

(
  export DOCKER_IMAGE="brunoanjos/demmon:latest"
  export LATENCY_MAP="config/latency_map.txt"
  export IPS_MAP="config/banjos_ips_config.txt"
  echo "Build demmon binary..."
  cd "$DEMMON_DIR"
  GO111MODULE='on' bash "$DEMMON_DIR"/scripts/buildImage.sh
  cp "$DEMMON_DIR"/"$LATENCY_MAP" $BUILD_DIR/dummy_node/latency_map
  cp "$DEMMON_DIR"/"$IPS_MAP" $BUILD_DIR/dummy_node/ips_map
  cp "$DEMMON_DIR"/scripts/setupTc.sh $BUILD_DIR/dummy_node/setupTc.sh
)

exit_code=$?

if [ "$exit_code" -ne 0 ]; then
  exit $exit_code
fi

# Deployer dependencies
cp "$BUILD_DIR"/deployer/fallback.json "$BUILD_DIR"/dummy_node/

# Client dependencies
cp -r "$CLOUD_EDGE_DEPLOYMENT"/deployments "$BUILD_DIR"/dummy_node/

cp $CLOUD_EDGE_DEPLOYMENT/scripts/clean_dummy.sh "$BUILD_DIR"/dummy_node/clean_dummy.sh

echo "Copying NOVAPokemon images to /tmp..."
cp /home/b.anjos/go/src/github.com/NOVAPokemon/images/* /tmp/images/

if [[ $1 == "--skip-final" ]]; then
  exit 0
fi

echo "Building final dummy node image..."
docker build -t brunoanjos/dummy_node:latest "$BUILD_DIR/dummy_node"

echo "Saving image to CLOUD_EDGE_DEPLOYMENT dir"
docker save brunoanjos/dummy_node:latest >"$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/dummy_node.tar

rm -rf "$BUILD_DIR"/dummy_node/deployments

#!/bin/bash

if [[ -z "${CLOUD_EDGE_DEPLOYMENT}" ]]; then
  export CLOUD_EDGE_DEPLOYMENT="/home/b.anjos/go/src/github.com/bruno-anjos/cloud-edge-deployment"
fi

export BUILD_DIR=/tmp/build



echo "Copying build to /tmp..."

cp -r $CLOUD_EDGE_DEPLOYMENT/build/* $BUILD_DIR/

echo "Removing garbage from previous runs..."

# Clear previous build directories and files
rm -rf /tmp/images
rm "$BUILD_DIR"/dummy_node/fallback.json
rm -rf "$BUILD_DIR"/dummy_node/metrics
rm -rf "$BUILD_DIR"/dummy_node/deployments
rm -rf "$BUILD_DIR"/dummy_node/images

set -e

echo "Build service binaries and client..."

bash "$BUILD_DIR"/build_binaries.sh &
bash "$BUILD_DIR"/build_client.sh

wait

mkdir /tmp/images

echo "Build service images..."

bash "$BUILD_DIR"/dummy_node/build_images.sh

# Deployer dependencies
cp "$BUILD_DIR"/deployer/fallback.json "$BUILD_DIR"/dummy_node/

# Autonomic dependencies
cp -r "$BUILD_DIR"/autonomic/metrics "$BUILD_DIR"/dummy_node/

# Client dependencies
cp -r "$CLOUD_EDGE_DEPLOYMENT"/deployments "$BUILD_DIR"/dummy_node/

(
  echo "Build demmon binary..."
  cd "$DEMMON_DIR"
  source config/swarmConfig.sh
  GO111MODULE='on' bash "$DEMMON_DIR"/scripts/buildImage.sh
  cp "$DEMMON_DIR"/"$LATENCY_MAP" $BUILD_DIR/dummy_node/latency_map
  cp "$DEMMON_DIR"/"$IPS_MAP" $BUILD_DIR/dummy_node/ips_map
  cp "$DEMMON_DIR"/scripts/setupTc.sh $BUILD_DIR/dummy_node/setupTc.sh
)

echo "Copying NOVAPokemon images to /tmp..."
cp /home/b.anjos/go/src/github.com/NOVAPokemon/images/* /tmp/images/

echo "Building final dummy node image..."
docker build -t brunoanjos/dummy_node:latest "$BUILD_DIR/dummy_node"

echo "Saving image to CLOUD_EDGE_DEPLOYMENT dir"
docker save brunoanjos/dummy_node:latest > "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/dummy_node.tar

rm -rf "$BUILD_DIR"/dummy_node/deployments

#!/bin/bash

BUILD_DIR="$CLOUD_EDGE_DEPLOYMENT"/build

# Clear previous build directories and files
rm "$BUILD_DIR"/dummy_node/fallback.txt
rm -rf "$BUILD_DIR"/dummy_node/metrics
rm -rf "$BUILD_DIR"/dummy_node/deployments
rm -rf "$BUILD_DIR"/dummy_node/images

set -e

bash "$BUILD_DIR"/build_binaries.sh &
bash "$BUILD_DIR"/build_client.sh &

wait

mkdir "$BUILD_DIR"/dummy_node/images
bash "$BUILD_DIR"/dummy_node/build_images.sh

# Deployer dependencies
cp "$BUILD_DIR"/deployer/fallback.txt "$BUILD_DIR"/dummy_node/

# Autonomic dependencies
cp -r "$BUILD_DIR"/autonomic/metrics "$BUILD_DIR"/dummy_node/

# Client dependencies
cp -r "$CLOUD_EDGE_DEPLOYMENT"/deployments "$BUILD_DIR"/dummy_node/

cp /home/b.anjos/go/src/github.com/NOVAPokemon/images/* "$BUILD_DIR"/dummy_node/images/

echo "Building final dummy node image..."
docker build -t brunoanjos/dummy_node:latest "$BUILD_DIR/dummy_node"

rm -rf "$BUILD_DIR"/dummy_node/deployments

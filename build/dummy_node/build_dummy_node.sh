#!/bin/bash

BUILD_DIR="$CLOUD_EDGE_DEPLOYMENT"/build

# Clear previous build directories and files
rm "$BUILD_DIR"/dummy_node/fallback.txt
rm -rf "$BUILD_DIR"/dummy_node/metrics
rm -rf "$BUILD_DIR"/dummy_node/deployments
rm -rf "$BUILD_DIR"/dummy_node/build_dirs
rm -rf "$BUILD_DIR"/dummy_node/images

set -e

bash "$BUILD_DIR"/build_binaries.sh &
bash "$BUILD_DIR"/build_client.sh &

wait

# Deployer dependencies
cp "$BUILD_DIR"/deployer/fallback.txt "$BUILD_DIR"/dummy_node/

# Autonomic dependencies
cp -r "$BUILD_DIR"/autonomic/metrics "$BUILD_DIR"/dummy_node/

# Client dependencies
cp -r "$CLOUD_EDGE_DEPLOYMENT"/deployments "$BUILD_DIR"/dummy_node/

mkdir "$BUILD_DIR"/dummy_node/build_dirs
cp -r "$BUILD_DIR"/archimedes/ "$BUILD_DIR"/dummy_node/build_dirs/
cp -r "$BUILD_DIR"/autonomic/ "$BUILD_DIR"/dummy_node/build_dirs/
cp -r "$BUILD_DIR"/deployer/ "$BUILD_DIR"/dummy_node/build_dirs/
cp -r "$BUILD_DIR"/scheduler/ "$BUILD_DIR"/dummy_node/build_dirs/

mkdir "$BUILD_DIR"/dummy_node/images
cp /home/b.anjos/go/src/github.com/NOVAPokemon/images/* "$BUILD_DIR"/dummy_node/images/

echo "Building final dummy node image..."
docker build -t brunoanjos/dummy_node:latest "$BUILD_DIR/dummy_node"

rm -rf "$BUILD_DIR"/dummy_node/deployments

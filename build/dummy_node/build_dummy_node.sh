#!/bin/bash

BUILD_DIR=$CLOUD_EDGE_DEPLOYMENT/build/dummy_node

bash "$CLOUD_EDGE_DEPLOYMENT"/scripts/build_binaries.sh

env CGO_ENABLED=1 go build -o "$CLOUD_EDGE_DEPLOYMENT/build/dummy_node/deployer-cli" \
	"${CLOUD_EDGE_DEPLOYMENT}/cmd/deployer-cli/main.go"

rm -rf "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/alternatives
mkdir "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/alternatives

function del_bin() {
	rm -f "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/"$SERVICE_NAME"
}

function cp_bin() {
	cp "$CLOUD_EDGE_DEPLOYMENT"/build/"$SERVICE_NAME"/"$SERVICE_NAME" "$BUILD_DIR"
}

SERVICE_NAME="archimedes"
del_bin
cp_bin

SERVICE_NAME="deployer"
del_bin
cp_bin

SERVICE_NAME="scheduler"
del_bin
cp_bin

SERVICE_NAME="autonomic"
del_bin
cp_bin

cp -r "$CLOUD_EDGE_DEPLOYMENT"/build/autonomic/metrics "$BUILD_DIR"/
cp "$CLOUD_EDGE_DEPLOYMENT"/build/deployer/fallback.txt "$BUILD_DIR"/
rm -rf "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/deployments
cp -r "$CLOUD_EDGE_DEPLOYMENT"/deployments "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/

docker build -t brunoanjos/dummy_node:latest "$BUILD_DIR"
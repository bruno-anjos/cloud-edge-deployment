#!/bin/bash

set -e

BUILD_DIR="$CLOUD_EDGE_DEPLOYMENT"/build
CMD_DIR="$CLOUD_EDGE_DEPLOYMENT"/cmd

function build() {
	echo "Building $SERVICE_NAME binary..."
	env CGO_ENABLED=1 go build -o "${BUILD_DIR}/${SERVICE_NAME}/${SERVICE_NAME}" \
		"${CMD_DIR}/${SERVICE_NAME}/main.go"
}

SERVICE_NAME="archimedes"
build &

SERVICE_NAME="deployer"
build &

SERVICE_NAME="scheduler"
build &

SERVICE_NAME="autonomic"
build &

wait

echo "Done building binaries!"

#!/bin/bash

set -e

REL_PATH="$(dirname "$0")"
BUILD_DIR="$REL_PATH/../build"
CMD_DIR="$REL_PATH/../cmd"

function build() {
	echo "running env CGO_ENABLED=1 go build -o \"${BUILD_DIR}/${SERVICE_NAME}/${SERVICE_NAME}\"\
	\"${CMD_DIR}/${SERVICE_NAME}/main.go\""

	env CGO_ENABLED=1 go build -o "${BUILD_DIR}/${SERVICE_NAME}/${SERVICE_NAME}" \
	"${CMD_DIR}/${SERVICE_NAME}/main.go"
}

SERVICE_NAME="archimedes"
build

SERVICE_NAME="deployer"
build

SERVICE_NAME="scheduler"
build

SERVICE_NAME="autonomic"
build

echo "Done building binaries"

wait
#!/bin/bash

REL_PATH="$(dirname "$0")"
BUILD_DIR="$REL_PATH/../build"
CMD_DIR="$REL_PATH/../cmd"
rm -rf "$BUILD_DIR/deployer/alternatives/*"

set -e

function build() {
	docker build -t brunoanjos/"$SERVICE_NAME":latest "$BUILD_DIR"/"$SERVICE_NAME"
}

SERVICE_NAME="archimedes"
build

SERVICE_NAME="deployer"
build

SERVICE_NAME="scheduler"
build

SERVICE_NAME="autonomic"
build

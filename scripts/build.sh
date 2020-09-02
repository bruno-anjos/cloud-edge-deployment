#!/bin/bash

REL_PATH="$(dirname "$0")"
BUILD_DIR="$REL_PATH/../build"
rm -rf "$BUILD_DIR/deployer/alternatives/*"

set -e

function build() {
	env CGO_ENABLED=0 GOOS=linux go build -o "$SERVICE_NAME" .
	docker build -t brunoanjos/"$SERVICE_NAME":latest "$BUILD_DIR"/"$SERVICE_NAME"
	rm "$SERVICE_NAME"
}

SERVICE_NAME="archimedes"
build

SERVICE_NAME="deployer"
build

SERVICE_NAME="scheduler"
build

SERVICE_NAME="autonomic"
build

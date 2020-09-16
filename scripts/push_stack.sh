#!/bin/bash

set -e

REL_PATH="$(dirname "$0")"
BUILD_DIR="$REL_PATH/../build"
set -e

bash "$REL_PATH"/build_images.sh

function push() {
	docker push -t brunoanjos/"$SERVICE_NAME":latest
}

SERVICE_NAME="archimedes"
push &

SERVICE_NAME="deployer"
push &

SERVICE_NAME="scheduler"
push &

SERVICE_NAME="autonomic"
push &

wait
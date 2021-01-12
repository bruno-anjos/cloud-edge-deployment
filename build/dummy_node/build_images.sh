#!/bin/bash

set -e

if [[ -z "${BUILD_DIR}" ]]; then
  echo "BUILD_DIR environment variable missing"
  exit 1
fi

build_image() {
  echo "Building $service image..."
  docker build -t brunoanjos/"$service":latest "$BUILD_DIR"/"$service"
  docker save brunoanjos/"$service":latest > /tmp/images/"$service".tar
}

service="archimedes"
build_image &

service="autonomic"
build_image &

service="deployer"
build_image &

service="scheduler"
build_image &

wait

echo "Done building images!"

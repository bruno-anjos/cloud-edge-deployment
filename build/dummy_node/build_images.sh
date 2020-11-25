#!/bin/sh

set -e

build_image() {
	echo "Building $service image..."
	docker build -t brunoanjos/"$service":latest /build_dirs/"$service"
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

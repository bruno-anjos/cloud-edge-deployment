#!/bin/sh

set -e

build_image() {
	echo "Building $service image..."
	docker build -t brunoanjos/"$service":latest "$CLOUD_EDGE_DEPLOYMENT"/build/"$service"
	docker save brunoanjos/"$service":latest > "$CLOUD_EDGE_DEPLOYMENT"/build/dummy_node/images/"$service".tar
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

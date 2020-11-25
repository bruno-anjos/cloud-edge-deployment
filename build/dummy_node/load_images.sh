#!/bin/sh

set -e

for image in images/*; do
	echo "Loading image $image..."
	docker load < $image
done
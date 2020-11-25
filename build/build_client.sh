#!/bin/bash

echo "Building client binary..."
env CGO_ENABLED=0 go build -o "$CLOUD_EDGE_DEPLOYMENT/build/dummy_node/deployer-cli" \
	"${CLOUD_EDGE_DEPLOYMENT}/cmd/deployer-cli/main.go"

echo "Done building client binary!"
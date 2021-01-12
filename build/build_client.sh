#!/bin/bash

if [[ -z "${BUILD_DIR}" ]]; then
  echo "BUILD_DIR environment variable missing"
  exit 1
fi


echo "Building client binary..."
env CGO_ENABLED=0 go build -o "$BUILD_DIR/dummy_node/deployer-cli" \
	"${CLOUD_EDGE_DEPLOYMENT}/cmd/deployer-cli/main.go"

echo "Done building client binary!"
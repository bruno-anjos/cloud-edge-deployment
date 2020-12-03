#!/bin/bash

set -e

if [ $# -ne 3 ]; then
	echo "usage: deploy_novapokemon_clients.sh num_clients region fallback"
	exit
fi

bash "$NOVAPOKEMON"/scripts/build_client.sh

num_clients=$1
region=$2
fallback=$3

docker run -d --env NUM_CLIENTS="$num_clients" --env REGION="$region" \
	--env FALLBACK_URL="$fallback" -v /tmp/client_logs:/logs brunoanjos/client:latest

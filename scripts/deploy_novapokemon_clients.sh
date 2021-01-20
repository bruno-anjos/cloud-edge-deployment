#!/bin/bash

set -e

if [ $# -ne 5 ]; then
	echo "usage: deploy_novapokemon_clients.sh <num_clients> <region> <logs_dir> <network> <timeout_string>"
	exit
fi

num_clients=$1
region=$2
logs_dir=$3
network=$4
timeout=$5

if [ -n "$(ls -A "$logs_dir")" ]; then
	read -p "There are logs in $logs_dir. Continue(y\\N)?" -n 1 -r
	echo
	if [[ ! $REPLY =~ ^[Yy]$ ]]; then
		exit 1
	fi
	docker run -v "$logs_dir":/logs debian:latest sh -c "rm -rf /logs/*"
fi

docker run -v /tmp/services:/services debian:latest sh -c "rm -rf /services/*"

bash "$NOVAPOKEMON"/scripts/build_client.sh

fallback=$(python3 -c 'import json; import os; '\
'fp = open(f"/tmp/build/deployer/fallback.json", "r"); fallback = json.load(fp); print(fallback["Addr"]); fp.close()')

docker run -d --env NUM_CLIENTS="$num_clients" --env REGION="$region" --env CLIENTS_TIMEOUT="$timeout" \
	--env FALLBACK_URL="$fallback" --env-file "$CLOUD_EDGE_DEPLOYMENT"/scripts/client-env.list \
	--network "$network" -v "$logs_dir":/logs -v /tmp/services:/services brunoanjos/client:latest
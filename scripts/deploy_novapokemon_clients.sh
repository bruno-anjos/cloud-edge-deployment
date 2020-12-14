#!/bin/bash

set -e

if [ $# -ne 4 ]; then
	echo "usage: deploy_novapokemon_clients.sh num_clients region fallback logs_dir"
	exit
fi

num_clients=$1
region=$2
fallback=$3
logs_dir=$4

if [ -n "$(ls -A "$logs_dir")" ]; then
	read -p "There are logs in /tmp/client_logs. Continue(y\\N)?" -n 1 -r
	echo
	if [[ ! $REPLY =~ ^[Yy]$ ]]; then
		exit 1
	fi
	docker run -v "$logs_dir":/logs debian:latest sh -c "rm -rf /logs/*"
fi

docker run -v /tmp/services:/services debian:latest sh -c "rm -rf /services/*"

bash "$NOVAPOKEMON"/scripts/build_client.sh

docker run -d --env NUM_CLIENTS="$num_clients" --env REGION="$region" \
	--env FALLBACK_URL="$fallback" --env-file "$CLOUD_EDGE_DEPLOYMENT"/scripts/client-env.list \
	--network dummies-network -v "$logs_dir":/logs -v /tmp/services:/services brunoanjos/client:latest
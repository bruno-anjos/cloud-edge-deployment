#!/bin/bash

set -e

if [ $# -ne 3 ]; then
	echo "usage: deploy_novapokemon_clients.sh num_clients region fallback"
	exit
fi

if [ -n "$(ls -A /tmp/client_logs)" ]; then
	read -p "There are logs in /tmp/client_logs. Continue(y\\N)?" -n 1 -r
	echo
	if [[ ! $REPLY =~ ^[Yy]$ ]]; then
		exit 1
	fi
	docker run -v /tmp/client_logs:/logs debian:latest sh -c "rm -rf /logs/*"
fi

docker run -v /tmp/services:/services debian:latest sh -c "rm -rf /services/*"

bash "$NOVAPOKEMON"/scripts/build_client.sh

num_clients=$1
region=$2
fallback=$3

docker run -d --env NUM_CLIENTS="$num_clients" --env REGION="$region" \
	--env FALLBACK_URL="$fallback" --env-file "$CLOUD_EDGE_DEPLOYMENT"/scripts/client-env.list \
	--network dummies-network -v /tmp/client_logs:/logs -v /tmp/services:/services brunoanjos/client:latest
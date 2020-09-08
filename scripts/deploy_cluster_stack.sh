#!/bin/bash

if [ "$CED_CLUSTER_NODES" = "" ]; then
	echo "CED_CLUSTER_NODES is not set"
	return
fi

IFS=';' read -ra NODES_ARRAY <<<"$CED_CLUSTER_NODES"

#Print the split string
for node in "${NODES_ARRAY[@]}"; do
	echo "-------------------------------------------------------"
	echo "Deploying stack in $node"
	if [ "$(hostname)" = "$node" ]; then
		echo "deploying locally in $node"
		cd $CLOUD_EDGE_DEPLOYMENT
		bash scripts/deploy_stack.sh
	else
		ssh "$node" "export GO111MODULE=on && export CLOUD_EDGE_DEPLOYMENT=$CLOUD_EDGE_DEPLOYMENT && cd \$CLOUD_EDGE_DEPLOYMENT && pwd && ./scripts/deploy_stack.sh"
	fi
done

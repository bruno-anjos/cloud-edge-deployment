#!/bin/bash

if [ "$CED_CLUSTER_NODES" = "" ]; then
	echo "CED_CLUSTER_NODES is not set"
	exit
fi

IFS=';' read -ra NODES_ARRAY <<<"$CED_CLUSTER_NODES"

echo "------------------------------------- REMOVING STACKS -------------------------------------"
for node in "${NODES_ARRAY[@]}"; do
	echo "Removing stack in $node"
	if [ "$(hostname)" = "$node" ]; then
		echo "removing locally in $node"
		cd $CLOUD_EDGE_DEPLOYMENT
		bash scripts/remove_stack.sh
	else
		ssh -o StrictHostKeyChecking=no "$node" "export GO111MODULE=on && export CLOUD_EDGE_DEPLOYMENT=$CLOUD_EDGE_DEPLOYMENT && cd \$CLOUD_EDGE_DEPLOYMENT && pwd && ./scripts/remove_stack.sh" &
	fi
	echo "---------------------------------------------------------------------------------------------"
done

wait

echo "------------------------------------- DEPLOYING STACKS -------------------------------------"
for node in "${NODES_ARRAY[@]}"; do
	echo "Deploying stack in $node"
	if [ "$(hostname)" = "$node" ]; then
		bash scripts/build_binaries.sh
		echo "deploying locally in $node"
		cd $CLOUD_EDGE_DEPLOYMENT
		bash scripts/deploy_stack.sh
	else
		ssh "$node" "export GO111MODULE=on && export CLOUD_EDGE_DEPLOYMENT=$CLOUD_EDGE_DEPLOYMENT && cd \$CLOUD_EDGE_DEPLOYMENT && pwd && ./scripts/deploy_stack.sh" &
	fi
	echo "---------------------------------------------------------------------------------------------"
done

wait

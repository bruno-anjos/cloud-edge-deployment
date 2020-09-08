#!/bin/bash

CED_CLUSTER_NODES=""

for var in "$@"; do
	echo "setting node$var"
	if [ "$CED_CLUSTER_NODES" = "" ]; then
		CED_CLUSTER_NODES="node$var"
	else
		CED_CLUSTER_NODES="${CED_CLUSTER_NODES};node$var"
	fi
done

echo "cluster nodes are $CED_CLUSTER_NODES"
export CED_CLUSTER_NODES=$CED_CLUSTER_NODES
#!/bin/bash

subnet=$1
name=$2

if [ -z $name ]; then
  name="swarm-network"
fi

host=$(hostname)

if [ -z $subnet ] || [ -z $name ]; then
  echo "setup takes 2 arguments (net_name defaults to swarm_network)"
  echo "setup.sh <subnet> <net_name>"
  exit
fi

docker swarm init
JOIN_TOKEN=$(docker swarm join-token manager -q)

for node in $(oarprint host); do
  if [ $node != $host ]; then
    oarsh $node "docker swarm join --token $JOIN_TOKEN $host:2377"
  fi
done

docker network create -d overlay --attachable --subnet $subnet $name

export DOCKER_NET="$name"
export DOCKER_MANAGER="$host"

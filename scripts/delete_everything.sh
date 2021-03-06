#!/usr/bin/bash

echo "Deleting everything on $(hostname)"

containers=$(docker ps -aq)
docker stop $containers
docker rm $containers
docker volume prune -f
docker system prune -f
docker network rm swarm-network

rm -rf /tmp/images/*
rm -rf /tmp/bandwidth_stats/*
rm -f ~/bandwidth_stats/*
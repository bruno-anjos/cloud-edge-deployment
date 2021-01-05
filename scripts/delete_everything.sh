#!/usr/bin/bash

echo "Deleting everything on $(hostname)"

containers=$(docker ps -aq)
docker stop $containers
docker rm $containers
docker network rm $DOCKER_NET
docker volume prune -f
docker system prune -f

#!/bin/sh

echo "Deleting everything on $(hostname)"

containers=$(docker ps -aq)
docker stop $containers
docker rm $containers
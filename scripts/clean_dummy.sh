#!/bin/sh

echo "Deleting everything on $(hostname)"

containers=$(docker ps -aq)
docker stop $containers
docker rm $containers
rm -rf /tmp/bandwidth_stats/"$(hostname)".csv
killall bwm-ng
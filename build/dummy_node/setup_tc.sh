#!/bin/sh

numNodes=$(cat /ips_map | wc -l)

echo "Bootstraping TC, args: /latency_map /ips_map $NODE_NUM"
sh setupTc.sh /latency_map /ips_map "$NODE_NUM" $numNodes
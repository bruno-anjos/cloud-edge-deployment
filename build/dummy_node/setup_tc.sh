#!/bin/sh

bandwidth_limit=$1
ip_prefix=$2
numNodes=$(cat /ips_map | wc -l)

echo "Bootstraping TC, args: /latency_map /ips_map $NODE_NUM $numNodes $bandwidth_limit"
./setupTc.sh /latency_map /ips_map "$NODE_NUM" "$numNodes" "$bandwidth_limit" "$ip_prefix"
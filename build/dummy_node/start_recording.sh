#!/bin/sh

timeout=$1
measurement_counts=$2

bwm-ng -t "$timeout" -o csv -c "$measurement_counts" -u bits -T rate -F /bandwidth_stats/"$NODE_ID".csv -D 1

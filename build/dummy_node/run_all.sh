#!/bin/bash

set -e

/archimedes -d &>/logs/archimedes_logs &
/scheduler -d &>/logs/scheduler_logs &
/autonomic -d &>/logs/autonomic_logs &
/deployer -d &>/logs/deployer_logs &

wait

cat /logs/archimedes_logs
cat /logs/scheduler_logs
cat /logs/autonomic_logs
cat /logs/deployer_logs

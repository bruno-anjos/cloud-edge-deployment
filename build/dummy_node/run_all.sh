#!/bin/bash

set -e

/archimedes -d &> /logs/archimedes_logs &
/scheduler -d -dummy &> /logs/scheduler_logs &
/deployer -d &> /logs/deployer_logs &
/autonomic -d  &> /logs/autonomic_logs &

wait
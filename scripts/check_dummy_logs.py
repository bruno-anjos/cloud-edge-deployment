#!/usr/bin/python3

import os
import subprocess

logsPrefix = "logs/"

archimedesLogs = "archimedes_logs"
autonomicLogs = "autonomic_logs"
deployerLogs = "deployer_logs"
schedulerLogs = "scheduler_logs"

logs = [archimedesLogs, autonomicLogs, deployerLogs, schedulerLogs]

dockerExecCmd = ["docker", "exec", "-it"]
filterSuffix = ["|", "grep", "\"level=error\""]

nodes = []
path = f"{os.path.dirname(os.path.realpath(__file__))}/../build/autonomic/metrics"
for f in os.listdir(path):
    if ".met" not in f:
        continue
    node = f.split(".met")[0]
    nodes.append(node)

print("nodes:", nodes)

for node in nodes:
    for log in logs:
        cmd = dockerExecCmd + [node] + ["cat" + logsPrefix + log] + filterSuffix
        output = subprocess.getoutput(" ".join(cmd))
        if output:
            print(f"[ERROR] {node} {log}:")
            print(output)
            exit(1)
        else:
            print(f"[OK] {node} {log}")

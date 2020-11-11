#!/usr/bin/python3

import os
import subprocess
import sys
from multiprocessing import Pool

logsPrefix = "logs/"

archimedesLogs = "archimedes_logs"
autonomicLogs = "autonomic_logs"
deployerLogs = "deployer_logs"
schedulerLogs = "scheduler_logs"

logs = [archimedesLogs, autonomicLogs, deployerLogs, schedulerLogs]

args = sys.argv
args = args[1:]

flag_all = False
flag_pattern = "level=error\\|panic\\|DATA RACE\\|race"
if len(args) > 0:
    error = False
    for idx, arg in enumerate(args):
        if arg == "-a":
            flag_all = True
        elif arg == "-p":
            if idx + 1 > len(args) - 1:
                error = True
                break
            flag_pattern = args[idx+1]
    if error:
        print("usage: python3 check_dummy_logs.py -a -p panic")


dockerExecCmd = ["docker", "exec"]
filterSuffix = ["|", "grep", f"\"{flag_pattern}\""]

logs_errors_path = "/home/b.anjos/dummy_logs_errors/"
files = os.listdir(logs_errors_path)
for f in files:
    os.remove(os.path.join(logs_errors_path, f))


def process_node_logs(data):
    node_to_process, log_to_process = data

    insideDockerCmd = ["cat", logsPrefix + log_to_process]
    insideDockerCmd.extend(filterSuffix)
    cmd = dockerExecCmd + [node_to_process] + [" ".join(insideDockerCmd)]
    out = subprocess.getoutput(" ".join(cmd))
    if out:
        return False, f"[ERROR] {node_to_process} {log_to_process}", out

    return True, f"[OK] {node_to_process} {log_to_process}", ""


nodes = []
path = f"{os.path.dirname(os.path.realpath(__file__))}/../build/autonomic/metrics"
for f in os.listdir(path):
    if ".met" not in f:
        continue
    node = f.split(".met")[0]
    nodes.append(node)

print("nodes:", nodes)

cpu_count = os.cpu_count()
print(f"using {cpu_count} cores")

logs_per_node = []
for node in nodes:
    for log in logs:
        logs_per_node.append((node, log))

pool = Pool(processes=cpu_count)
results = pool.map(process_node_logs, logs_per_node)
pool.close()

for idx, log_per_node in enumerate(logs_per_node):
    success, to_log, output = results[idx]
    node, log = log_per_node
    if flag_all or not success:
        print(to_log)
    if not success:
        with open(f"{logs_errors_path}/{node}_{log}", "w") as error_fp:
            error_fp.write(output)

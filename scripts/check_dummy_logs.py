#!/usr/bin/python3

import json
import os
import socket
import subprocess
import sys
from multiprocessing import Pool

archimedesLogs = "archimedes"
autonomicLogs = "autonomic"
deployerLogs = "deployer"
schedulerLogs = "scheduler"
demmonLogs = "demmon"

logs = [archimedesLogs, autonomicLogs, deployerLogs, schedulerLogs, demmonLogs]

host = socket.gethostname().strip()

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
            flag_pattern = args[idx + 1]
    if error:
        print("usage: python3 check_dummy_logs.py -a -p panic")

dockerExecCmd = ["docker", "exec"]
filterSuffix = ["|", "grep", f"\"{flag_pattern}\""]

logs_errors_path = "/home/b.anjos/dummy_logs_errors/"
files = os.listdir(logs_errors_path)
for f in files:
    os.remove(os.path.join(logs_errors_path, f))


def process_node_logs(data):
    node_to_process, log_to_process, cluster_node = data

    inside_docker_cmd = ["docker", "logs", "2>&1", log_to_process]
    inside_docker_cmd.extend(filterSuffix)
    cmd = dockerExecCmd + [node_to_process] + [" ".join(inside_docker_cmd)]

    if cluster_node == host:
        out = subprocess.getoutput(" ".join(cmd))
    else:
        out = subprocess.getoutput(f"oarsh {cluster_node} -- " + " ".join(cmd))

    if out:
        return False, f"[ERROR] {node_to_process} {log_to_process}", out

    return True, f"[OK] {node_to_process} {log_to_process}", ""


def process_other_logs(info):
    node_to_process, cluster_node = info

    inside_docker_cmd = ["docker", "ps", "--format", "{{.Names}}"]
    cmd = dockerExecCmd + [node_to_process] + [" ".join(inside_docker_cmd)]
    containers = subprocess.getoutput(" ".join(cmd))

    returns = []
    for line in containers.split("\n"):
        container_name = line.strip()
        if container_name in logs:
            continue

        inside_docker_cmd = ["docker", "logs", "2>&1", container_name]
        inside_docker_cmd.extend(filterSuffix)
        cmd = dockerExecCmd + [node_to_process] + [" ".join(inside_docker_cmd)]

        if cluster_node == host:
            out = subprocess.getoutput(" ".join(cmd))
        else:
            out = subprocess.getoutput(f"oarsh {cluster_node} -- " + " ".join(cmd))

        if out:
            returns.append((container_name, False, f"[ERROR] {node_to_process} {container_name}", out))
        else:
            returns.append((container_name, True, f"[OK] {node_to_process} {container_name}", ""))

    return returns


nodes = []
path = f"{os.path.dirname(os.path.realpath(__file__))}/../build/autonomic/metrics"
for f in os.listdir(path):
    if ".met" not in f:
        continue
    node = f.split(".met")[0]
    nodes.append(node)

cpu_count = os.cpu_count()
print(f"[INFO] using {cpu_count} cores")

nodes_with_cluster = []

with open("/tmp/dummy_infos.json", "r") as dummy_infos_fp:
    infos_list = json.load(dummy_infos_fp)

    dummy_infos = {}
    for info in infos_list:
        dummy_infos[info["name"]] = info

    logs_per_node = []
    for node in nodes:
        for log in logs:
            logs_per_node.append((node, log, dummy_infos[node]["node"]))
        nodes_with_cluster.append((node, dummy_infos[node]["node"]))

with open(f"{os.path.dirname(os.path.realpath(__file__))}/../deployments/clients_config.json") as services_config_fp:
    service_configs = json.load(services_config_fp)

print("[INFO] nodes:", nodes)

pool = Pool(processes=cpu_count)
results = pool.map(process_node_logs, logs_per_node)
other_results = pool.map(process_other_logs, nodes_with_cluster)
pool.close()

for idx, log_per_node in enumerate(logs_per_node):
    success, to_log, output = results[idx]
    node, log, cluster_node = log_per_node
    if flag_all or not success:
        print(to_log)
    if not success:
        with open(f"{logs_errors_path}/{node}_{log}.txt", "w") as error_fp:
            error_fp.write(output + "\n")

for idx, node in enumerate(nodes):
    other_logs_per_node = other_results[idx]
    for other_logs in other_logs_per_node:
        container, success, to_log, output = other_logs
        if flag_all or not success:
            print(to_log)
        if not success:
            with open(f"{logs_errors_path}/{node}_{container}.txt", "w") as error_fp:
                error_fp.write(output + "\n")

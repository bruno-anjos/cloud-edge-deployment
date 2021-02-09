#!/usr/bin/python
import json
import os
import subprocess
import sys
from datetime import datetime
from multiprocessing import Pool

clients_config_file = f"{os.path.expanduser('~')}" \
                      f"/go/src/github.com/bruno-anjos/cloud-edge-deployment/deployments/clients_config.json"

log_prefixes = ["archimedes", "autonomic", "deployer", "scheduler", "demmon"]


def get_client_logs(logs_dir_name):
    clients_dir_path = f"{logs_dir_name}/clients/"
    os.mkdir(clients_dir_path)
    subprocess.run(["cp", "/tmp/client_logs/*", clients_dir_path])


def get_specific_logs(logs_dir_name, dummy, cluster_node, logs_prefix):
    inside_docker_cmd = f"docker logs {logs_prefix}".split(" ")
    docker_logs_cmd = f"oarsh {cluster_node} -- docker exec {dummy}".split(" ")
    docker_logs_cmd.extend(inside_docker_cmd)
    log_path = f"{logs_dir_name}/{dummy}/{logs_prefix}"
    print(log_path)
    print(docker_logs_cmd)
    with open(log_path, "w") as log_file:
        subprocess.run(docker_logs_cmd, stdout=log_file, stderr=log_file)


def get_dummy_logs(logs_dir_name, dummy):
    os.mkdir(f"{logs_dir_name}/{dummy}")
    cluster_node = dummy_infos[dummy]["node"]
    for log_prefix in log_prefixes:
        get_specific_logs(logs_dir_name, dummy, cluster_node, log_prefix)


args = sys.argv[1:]
if len(args) > 1:
    print("ERROR: usage: python3 get_logs.py DIR_TO_WRITE_LOGS_FOLDER")
    exit(1)

date = datetime.now()
timestamp = date.strftime("%m-%d-%H-%M")

if len(args) == 1:
    logs_dir = args[0]
else:
    logs_dir = f'{os.path.expanduser("~")}/dummy_logs_{timestamp}'

with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/locations.json", 'r') as locations_fp:
    nodes = json.load(locations_fp)["nodes"].keys()

if os.path.isdir(logs_dir) or os.path.isfile(logs_dir):
    print(f"ERROR: dir {logs_dir} already exists")
    exit(1)

os.mkdir(logs_dir)

with open(f"/tmp/dummy_infos.json", "r") as dummy_infos_fp:
    infos = json.load(dummy_infos_fp)
    dummy_infos = {}
    for info in infos:
        dummy_infos[info["name"]] = info

pool = Pool(processes=os.cpu_count())
processess = []

for node in nodes:
    processess.append(pool.apply_async(get_dummy_logs, (logs_dir, node)))

get_client_logs(logs_dir)

for p in processess:
    p.wait()

pool.close()

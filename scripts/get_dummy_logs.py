#!/usr/bin/python
import json
import os
import subprocess
import sys
from datetime import datetime
from multiprocessing import Pool

log_prefixes = ["archimedes", "autonomic", "deployer", "scheduler"]


def get_specific_logs(logs_dir_name, dummy, logs_prefix):
    docker_logs_cmd = f"docker exec {dummy} cat logs/{logs_prefix}_logs".split(" ")
    log_path = f"{logs_dir_name}/{dummy}/{logs_prefix}_logs"
    print(log_path)
    with open(log_path, "w") as log_file:
        subprocess.run(docker_logs_cmd, stdout=log_file)


def get_dummy_logs(logs_dir_name, dummy):
    os.mkdir(f"{logs_dir_name}/{dummy}")
    for log_prefix in log_prefixes:
        get_specific_logs(logs_dir_name, dummy, log_prefix)


if len(sys.argv) > 2:
    print("ERROR: usage: python3 get_dummy_logs.py DIR_TO_WRITE_LOGS_FOLDER")
    exit(1)

date = datetime.now()
timestamp = date.strftime("%m-%d-%H-%M")

if len(sys.argv) == 2:
    logs_dir = sys.argv[1]
else:
    logs_dir = f'{os.path.expanduser("~")}/dummy_logs_{timestamp}'

with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/locations.json", 'r') as locations_fp:
    nodes = json.load(locations_fp)["nodes"].keys()

if os.path.isdir(logs_dir) or os.path.isfile(logs_dir):
    print(f"ERROR: dir {logs_dir} already exists")
    exit(1)

os.mkdir(logs_dir)

pool = Pool(processes=os.cpu_count())
processess = []

for node in nodes:
    processess.append(pool.apply_async(get_dummy_logs, (logs_dir, node)))

for p in processess:
    p.wait()

pool.close()

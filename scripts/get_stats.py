#!/usr/bin/python3
import json
import os
import socket
import subprocess
import sys

client_logs_dirname = "client_logs"

timetook_regex = r"took (\d+)"

TIMESTAMP = "TIMESTAMP"
BYTES_OUT = "BYTES_OUT"
BYTES_IN = "BYTES_IN"
BYTES_TOTAL = "BYTES_TOTAL"


def run_cmd_with_try(cmd):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=subprocess.DEVNULL)
    if cp.stderr is not None:
        raise Exception(cp.stderr)


def exec_cmd_on_node(node, cmd):
    path_var = os.environ["PATH"]
    remote_cmd = f"oarsh {node} -- 'PATH=\"{path_var}\" && {cmd}'"
    run_cmd_with_try(remote_cmd)


def sync_stats_to_NAS():
    hostname = socket.gethostname()

    nodes = {}
    for info in dummy_infos:
        node = info["node"]
        nodes[node] = None

    cp_to_nas_cmd = f"cp /tmp/bandwidth_stats/* {os.path.expanduser('~/bandwidth_stats/')}"
    for node in nodes:
        if node == hostname:
            run_cmd_with_try(cp_to_nas_cmd)
        else:
            exec_cmd_on_node(node, cp_to_nas_cmd)

    print("Synced stats to NAS")


def get_bandwidth_stats():
    bandwidth_dir = f"{os.path.expanduser('~/bandwidth_stats')}"
    node_results = {}

    for node in os.listdir(bandwidth_dir):
        node_results[node] = []

        with open(f"{bandwidth_dir}/{node}") as bandwidth_fp:
            measures = 0

            for line in bandwidth_fp.readlines():
                measures += 1
                splits = line.split(";")

                timestamp = splits[0]
                bytes_out_s = splits[2]
                bytes_in_s = splits[3]
                bytes_total_s = splits[4]

                node_results[node].append({
                    TIMESTAMP: timestamp,
                    BYTES_OUT: bytes_out_s,
                    BYTES_IN: bytes_in_s,
                    BYTES_TOTAL: bytes_total_s
                })

            print(f"{node} has {measures} measures")

    return node_results


args = sys.argv[1:]

if len(args) > 1:
    print("usage: python3 get_stats.py [output_log_dir]")
    exit(1)

if len(args) == 1:
    output_dir = args[0]
else:
    output_dir = os.path.expanduser("~")

with open("/tmp/dummy_infos.json", "r") as dummy_infos_fp:
    dummy_infos = json.load(dummy_infos_fp)

print("Will sync stats to NAS...")
sync_stats_to_NAS()

results = get_bandwidth_stats()
with open(f"{output_dir}/bandwidth_results.json", 'w') as results_fp:
    json.dump(results, results_fp, indent=4)

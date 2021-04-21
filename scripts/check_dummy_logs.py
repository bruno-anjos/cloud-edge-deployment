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

dockerExecCmd = ["docker", "exec"]
filterSuffix = ""


def process_default_logs(data):
    node_to_process, log_to_process, cluster_node = data

    inside_docker_cmd = ["docker", "logs", "2>&1", log_to_process]
    inside_docker_cmd.extend(filterSuffix)
    cmd = dockerExecCmd + [node_to_process] + [" ".join(inside_docker_cmd)]

    if cluster_node == host:
        out = subprocess.getoutput(" ".join(cmd))
    else:
        out = subprocess.getoutput(f"oarsh {cluster_node} -- " + " ".join(cmd))

    if out:
        return node_to_process, log_to_process, False, f"[ERROR] {node_to_process} {log_to_process}", out

    return node_to_process, log_to_process, True, f"[OK] {node_to_process} {log_to_process}", ""


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


def process_dummy_logs(info):
    node, logs_dir_path = info

    returns = []

    for log_name in os.listdir(logs_dir_path):
        log_path = os.path.join(logs_dir_path, log_name)
        cmd = ['grep', '\"{flag_pattern}\"', log_path]
        out = subprocess.getoutput(" ".join(cmd))

        if out:
            returns.append((node, log_name, False, f"[ERROR] {node} {log_name}", out))
        else:
            returns.append((node, log_name, True, f"[OK] {node} {log_name}", ""))

    return returns


def process_logs_dir(logs_dir, pool):
    dirs_with_logs = []
    for log_dir in os.listdir(logs_dir):
        full_path = os.path.join(logs_dir, log_dir)
        if not os.path.isdir(full_path):
            continue
        if "clients" in log_dir:
            for client_logs_dir in os.listdir(full_path):
                clients_path = os.path.join(full_path, client_logs_dir)
                dirs_with_logs.append((client_logs_dir, clients_path))
        elif "dummy" in log_dir:
            dirs_with_logs.append((log_dir, full_path))

    returns = []

    node_returns = pool.map(process_dummy_logs, dirs_with_logs)

    for node_return in node_returns:
        returns.extend(node_return)

    return returns


def build_infos():
    with open("/tmp/dummy_infos.json", "r") as dummy_infos_fp:
        infos_list = json.load(dummy_infos_fp)

        nodes = [node['name'] for node in infos_list]

        dummy_infos = {}
        for info in infos_list:
            dummy_infos[info["name"]] = info

        logs_per_node = []
        nodes_with_cluster = []

        for node in nodes:
            for log in logs:
                logs_per_node.append((node, log, dummy_infos[node]["node"]))
            nodes_with_cluster.append((node, dummy_infos[node]["node"]))

    return nodes, dummy_infos, logs_per_node, nodes_with_cluster


def show_docker_results(nodes, results, other_results, flag_all, logs_errors_path):
    for result in results:
        node, log,  success, to_log, output = result
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


def show_dir_results(results, flag_all, logs_errors_path):
    for result in results:
        node, log, success, to_log, output = result
        if flag_all or not success:
            print(to_log)
        if not success:
            with open(f"{logs_errors_path}/{node}_{log}.txt", "w") as error_fp:
                error_fp.write(output + "\n")


def main():
    args = sys.argv
    args = args[1:]

    if len(args) > 3:
        print("usage: python3 check_dummy_logs.py [-a] [-p panic] [-d logs_dir]")
        exit(1)

    flag_all = False
    flag_pattern = "level=error\\|panic\\|DATA RACE"
    logs_dir = ""
    skip = False

    for i, arg in enumerate(args):
        if skip:
            skip = False
            continue
        if arg == "-a":
            flag_all = True
        elif arg == "-p":
            flag_pattern = args[i + 1]
            skip = True
        elif arg == "-d":
            logs_dir = args[i + 1]
            skip = True

    global filterSuffix
    filterSuffix = ["|", "grep", f"\"{flag_pattern}\""]

    logs_errors_path = "/home/b.anjos/dummy_logs_errors/"
    if not os.path.exists(logs_errors_path):
        os.mkdir(logs_errors_path)

    files = os.listdir(logs_errors_path)
    for f in files:
        os.remove(os.path.join(logs_errors_path, f))

    cpu_count = os.cpu_count()
    print(f"[INFO] using {cpu_count} cores")

    nodes, dummy_infos, logs_per_node, nodes_with_cluster = build_infos()

    print("[INFO] nodes:", nodes)

    pool = Pool(processes=cpu_count)

    if logs_dir == "":
        results = pool.map(process_default_logs, logs_per_node)
        other_results = pool.map(process_other_logs, nodes_with_cluster)

        show_docker_results(nodes, results, other_results, flag_all, logs_errors_path)
    else:
        results = process_logs_dir(logs_dir, pool)
        show_dir_results(results, flag_all, logs_errors_path)

    pool.close()


if __name__ == '__main__':
    main()

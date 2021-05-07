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


def run_cmd_with_try(cmd):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=subprocess.DEVNULL)

    failed = False
    if cp.stderr is not None:
        failed = True
        print(f"StdErr: {cp.stderr}")
    if cp.returncode != 0:
        failed = True
        print(f"RetCode: {cp.returncode}")
    if failed:
        exit(1)


def exec_cmd_on_node(node, cmd):
    path_var = os.environ["PATH"]
    remote_cmd = f"oarsh {node} -- 'PATH=\"{path_var}\" && {cmd}'"
    run_cmd_with_try(remote_cmd)


def get_client_logs(client_logs_dir_name, clients_node):
    cmd = f'cp -R /tmp/client_logs/* {client_logs_dir_name}'
    exec_cmd_on_node(clients_node, cmd)


def get_specific_logs(logs_dir_name, dummy, cluster_node, logs_prefix):
    inside_docker_cmd = f"docker logs {logs_prefix}".split(" ")
    docker_logs_cmd = f"oarsh {cluster_node} -- docker exec {dummy}".split(" ")
    docker_logs_cmd.extend(inside_docker_cmd)
    log_path = f"{logs_dir_name}/{dummy}/{logs_prefix}"
    print(log_path)
    print(docker_logs_cmd)
    with open(log_path, "w") as log_file:
        subprocess.run(docker_logs_cmd, stdout=log_file, stderr=log_file)


def get_other_logs(logs_dir_name, dummy, cluster_node):
    inside_docker_cmd = f"docker ps -a --format {{{{.Names}}}}".split(" ")
    docker_logs_cmd = f"oarsh {cluster_node} -- docker exec {dummy}".split(" ")
    docker_logs_cmd.extend(inside_docker_cmd)

    output = subprocess.getoutput(" ".join(docker_logs_cmd))
    dummy_containers = [line.strip() for line in output.split(
        "\n") if line not in log_prefixes]
    for container in dummy_containers:
        get_specific_logs(logs_dir_name, dummy, cluster_node, container)


def get_dummy_logs(dummy_infos, server_logs_dir, dummy):
    os.mkdir(f"{server_logs_dir}/{dummy}")
    cluster_node = dummy_infos[dummy]["node"]
    for log_prefix in log_prefixes:
        get_specific_logs(server_logs_dir, dummy, cluster_node, log_prefix)

    get_other_logs(server_logs_dir, dummy, cluster_node)


def main():
    args = sys.argv[1:]
    if len(args) != 2:
        print("ERROR: usage: python3 get_logs.py <output_dir> <clients_node>")
        exit(1)

    logs_dir = ''
    clients_node = ''

    for arg in args:
        if logs_dir == '':
            logs_dir = arg
        elif clients_node == '':
            clients_node = arg

    with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/locations.json", 'r') as locations_fp:
        nodes = json.load(locations_fp)["nodes"].keys()

    if not os.path.exists(logs_dir):
        os.mkdir(logs_dir)

    with open(f"/tmp/dummy_infos.json", "r") as dummy_infos_fp:
        infos = json.load(dummy_infos_fp)
        dummy_infos = {}
        for info in infos:
            dummy_infos[info["name"]] = info

    server_logs_dir = f'{logs_dir}/servers'
    os.mkdir(server_logs_dir)
    client_logs_dir = f'{logs_dir}/clients'
    os.mkdir(client_logs_dir)

    pool = Pool(processes=os.cpu_count())
    processess = []

    for node in nodes:
        processess.append(
            pool.apply_async(
                get_dummy_logs, (dummy_infos, server_logs_dir, node)
            )
        )

    get_client_logs(client_logs_dir, clients_node)

    for p in processess:
        p.wait()

    pool.close()


if __name__ == '__main__':
    main()

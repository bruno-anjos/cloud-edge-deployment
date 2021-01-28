#!/usr/bin/python3
import json
import os
import socket
import subprocess
import sys

repoDir = "/home/b.anjos/go/src/github.com/bruno-anjos/cloud-edge-deployment"
cdToDir = f"cd {repoDir}"
runServiceFormat = "go run cmd/deployer-cli/main.go add static %s deployments/%s.yaml"
exportVars = "export GO111MODULE=on"

host = socket.gethostname().strip()

args = sys.argv[1:]

dummy = False
scenario = ""

min_args = 1
max_args = 2

args_count = 0

for arg in args:
    if "--dummy" == arg:
        dummy = True
    elif scenario == "":
        scenario = arg
        args_count += 1

if args_count != min_args or len(args) < min_args or len(args) > max_args:
    print("usage: python3 deploy_novapokemon.py <scenario.json> [--dummy]")
    exit(1)


def deploy_service_in_node(s, node):
    run_service = runServiceFormat % s
    try:
        subprocess.run(["ssh", node, f"{exportVars} && {cdToDir} && {run_service}"], check=True)
    except subprocess.CalledProcessError as e:
        print(f"ssh returned {e}")


def deploy_dummy(s_name, yaml, node, cluster_node):
    try:
        cmd = f"docker exec {node} /deployer-cli add {s_name} /deployments/{yaml}"
        if cluster_node == host:
            subprocess.run(cmd, shell=True, check=True)
        else:
            remote_cmd = f"oarsh {cluster_node} -- '{cmd}'"
            subprocess.run(remote_cmd, shell=True, check=True)
    except subprocess.CalledProcessError as e:
        print(f"docker exec returned {e}")


def remove_services():
    cmd = ["docker", "run", "-v", "/tmp/services:/services", "debian:latest", "sh", "-c", "rm -rf /services/*"]
    subprocess.run(cmd, check=True)


remove_services()
yaml_config_property = "deployment_yaml"
project_path = os.environ["CLOUD_EDGE_DEPLOYMENT"]
fallback_path = os.path.expanduser(f"{os.path.expanduser('~/ced-scenarios/')}/{scenario}")
client_config_path = os.path.expanduser(f"{project_path}/deployments/clients_config.json")
with open(fallback_path, 'r') as fallback_fp, open(client_config_path, 'r') as client_config_fp, \
        open("/tmp/dummy_infos.json", 'r') as dummy_infos_fp:
    dummy_infos = json.load(dummy_infos_fp)
    startNode = json.load(fallback_fp)["fallback"]
    services = json.load(client_config_fp)
    print(f"Start Node: {startNode}")
    for service_name, service in services.items():
        yaml_path = f"{service_name}.yaml"
        if yaml_config_property in service:
            yaml_path = service[yaml_config_property]
        print(f"deploying {service_name} in {startNode}")
        if dummy:
            for info in dummy_infos:
                if info["name"] == startNode:
                    cluster_node = info["node"]
            deploy_dummy(service_name, yaml_path, startNode, cluster_node)
        else:
            deploy_service_in_node(service_name, startNode)

#!/usr/bin/python3
import json
import os
import subprocess
import sys

repoDir = "/home/b.anjos/go/src/github.com/bruno-anjos/cloud-edge-deployment"
cdToDir = f"cd {repoDir}"
runServiceFormat = "go run cmd/deployer-cli/main.go add static %s deployments/%s.yaml"
exportVars = "export GO111MODULE=on"

dummy = False
if "--dummy" in sys.argv:
    dummy = True


def deploy_service_in_node(s, node):
    run_service = runServiceFormat % s
    try:
        subprocess.run(["ssh", node, f"{exportVars} && {cdToDir} && {run_service}"], check=True)
    except subprocess.CalledProcessError as e:
        print(f"ssh returned {e}")


def deploy_dummy(s_name, yaml, node):
    try:
        subprocess.run(
            ["docker", "exec", node, "/deployer-cli", "add", s_name, f"/deployments/{yaml}"],
            check=True)
    except subprocess.CalledProcessError as e:
        print(f"docker exec returned {e}")


yaml_config_property = "deployment_yaml"

project_path = os.environ["CLOUD_EDGE_DEPLOYMENT"]
fallback_path = os.path.expanduser(f"{project_path}/build/deployer/fallback.json")
client_config_path = os.path.expanduser(f"{project_path}/deployments/clients_config.json")
with open(fallback_path, 'r') as fallback_fp, open(client_config_path, 'r') as client_config_fp:
    startNode = json.load(fallback_fp)["Id"]
    services = json.load(client_config_fp)
    print(f"Start Node: {startNode}")
    for service_name, service in services.items():
        yaml_path = f"{service_name}.yaml"
        if yaml_config_property in service:
            yaml_path = service[yaml_config_property]
        print(f"deploying {service_name} in {startNode}")
        if dummy:
            deploy_dummy(service_name, yaml_path, startNode)
        else:
            deploy_service_in_node(service_name, startNode)

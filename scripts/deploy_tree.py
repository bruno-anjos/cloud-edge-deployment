#!/usr/bin/python3
import os
import subprocess

import sys

repoDir = os.path.expanduser("~/go/src/github.com/bruno-anjos/cloud-edge-deployment")
cdToDir = f"cd {repoDir}"
runServiceFormat = "go run cmd/deployer-cli/main.go add static %s deployments/%s.yaml"
exportVars = "export GO111MODULE=on"

dummy = False
if "--dummy" in sys.argv:
    dummy = True


def deploy_service_in_node(s, node):
    run_service = runServiceFormat % s
    try:
        subprocess.run(["oarsh", node, f"{exportVars} && {cdToDir} && {run_service}"], check=True)
    except subprocess.CalledProcessError as e:
        print(f"ssh returned {e}")


def deploy_dummy(s, node):
    try:
        subprocess.run(["docker", "exec", node, "/deployer-cli", "add", "static", s, f"/deployments/{s}.yaml"],
                       check=True)
    except subprocess.CalledProcessError as e:
        print(f"docker exec returned {e}")


pathToTrees = f"{repoDir}/build/autonomic/metrics/services.tree"
with open(pathToTrees, 'r') as file:
    line = file.readline()
    while line:
        splits = line.split(":")
        service = splits[0].strip()
        nodesJoined = splits[1]
        startNode = nodesJoined.split(" -> ")[0].strip()
        print(f"deploying {service} in {startNode}")
        if dummy:
            deploy_dummy(service, startNode)
        else:
            deploy_service_in_node(service, startNode)
        line = file.readline()

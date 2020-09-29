#!/usr/bin/python3

import subprocess
import sys

repoDir = "/home/b.anjos/go/src/github.com/bruno-anjos/cloud-edge-deployment"
cdToDir = f"cd {repoDir}"
runServiceFormat = "go run cmd/deployer-cli/main.go add static %s deployments/dummy.yaml"
exportVars = "export GO111MODULE=on"

dummy = False
if "--dummy" in sys.argv:
    dummy = True


def deploy_service_in_node(s, node):
    global runServiceFormat
    global cdToDir

    runService = runServiceFormat % s
    try:
        subprocess.run(["ssh", node, f"{exportVars} && {cdToDir} && {runService}"], check=True)
    except subprocess.CalledProcessError as e:
        print(f"ssh returned {e}")


def deploy_dummy(s, node):
    try:
        subprocess.run(["docker", "exec", node, "/deployer-cli", "add", "static", s, "/deployments/dummy.yaml"],
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

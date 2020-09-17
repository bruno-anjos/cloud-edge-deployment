#!/usr/bin/python3

import subprocess

repoDir = "/home/b.anjos/go/src/github.com/bruno-anjos/cloud-edge-deployment"
cdToDir = f"cd {repoDir}"
runServiceFormat = "go run cmd/deployer-cli/main.go add static %s deployments/dummy.yaml"
exportVars = "export GO111MODULE=on"


def deploy_service_in_node(s, node):
    global runServiceFormat
    global cdToDir

    runService = runServiceFormat % s
    try:
        subprocess.run(["ssh", node, f"{exportVars} && {cdToDir} && {runService}"], check=True)
    except subprocess.CalledProcessError as e:
        print(f"ssh returned {e}")


pathToTrees = f"{repoDir}/build/autonomic/metrics/services.tree"
with open(pathToTrees, 'r') as file:
    line = file.readline()
    while line:
        splits = line.split(":")
        service = splits[0].strip()
        nodesJoined = splits[1]
        startNode = nodesJoined.split(" -> ")[0].strip()
        print(f"deploying {service} in {startNode}")
        deploy_service_in_node(service, startNode)
        line = file.readline()

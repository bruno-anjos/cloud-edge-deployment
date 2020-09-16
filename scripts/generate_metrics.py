#!/usr/bin/python3

import json
import os
import random
import sys

"""
{
  "METRIC_NODE_ADDR": "node21",
  "METRIC_LOCATION": 5000.0,
  "METRIC_LOCATION_VICINITY": {
    "node21": 5000.0,
    "node23": 2000.0
  },
  "METRIC_NUMBER_OF_INSTANCES_PER_SERVICE_A": 0,
  "METRIC_LOAD_PER_SERVICE_A_IN_CHILD_node22": 0,
  "METRIC_LOAD_PER_SERVICE_A_IN_CHILD_node23": 0,
  "METRIC_LOAD_PER_SERVICE_A_IN_CHILDREN": {},
  "METRIC_AGG_LOAD_PER_SERVICE_A_IN_CHILDREN": 0,
  "METRIC_CLIENT_LATENCY_PER_SERVICE_A": 150,
  "METRIC_PROCESSING_TIME_PER_SERVICE_A": 10,
  "METRIC_AVERAGE_CLIENT_LOCATION_PER_SERVICE_A": 1000.0
}
"""

sortingService = ""


def sortChildren(n):
    global sortingService
    global clientLocations
    return abs(nodesLocations[n] - clientLocations[sortingService])


def generateServiceTree(startNode, sName):
    global nodesChildren
    global nodesLocations
    global clientLocations
    global sortingService

    sortingService = sName
    tree = f"{sName}: {startNode}"

    better = True
    currNode = startNode

    print(f"creating tree for {sName}, starting at {startNode}")

    while better:
        children = nodesChildren[currNode]
        if len(children) < 1:
            better = False
            continue
        candidates = []
        candidates.extend(children)
        candidates += [currNode]
        candidates.sort(key=sortChildren)
        best = candidates[0]
        print(f"from {candidates}, best is {best}")
        if best != currNode:
            tree += f" -> {best}"
            currNode = best
        else:
            better = False

    return tree


def generateDictsForServiceTree(sName):
    global serviceLatencies
    global processingTimes
    global clientLocations

    minLatency, maxLatency = 100, 1000
    serviceLatencies[sName] = random.randint(minLatency, maxLatency)

    minProcess, maxProcess = 0, 10
    processingTimes[sName] = random.randint(minProcess, maxProcess)

    minCLoc, maxCLoc = 1000, 5000
    clientLocations[sName] = random.randint(minCLoc, maxCLoc)


def generateNodeMetrics(nodeId, loc, visibleNodes, children):
    global services
    global nodesLocations
    visibleNodesLocation = ",\n".join([f"\"{nodeId}\":{nodesLocations[nodeId]}" for nodeId in visibleNodes])

    other = []
    for s in services:
        other += [f"\"METRIC_LOAD_PER_SERVICE_{s}_IN_CHILDREN\": {{}}",
                  f"\"METRIC_AGG_LOAD_PER_SERVICE_{s}_IN_CHILDREN\": 0",
                  f"\"METRIC_CLIENT_LATENCY_PER_SERVICE_{s}\": {serviceLatencies[s]}",
                  f"\"METRIC_PROCESSING_TIME_PER_SERVICE_{s}\": {processingTimes[s]}",
                  f"\"METRIC_NUMBER_OF_INSTANCES_PER_SERVICE_{s}\": 0",
                  f"\"METRIC_AVERAGE_CLIENT_LOCATION_PER_SERVICE_{s}\": {clientLocations[s]}"]
        for child in children:
            other += [f"\"METRIC_LOAD_PER_SERVICE_{s}_IN_CHILD_{child}\": 0"]

    otherStrings = ",\n".join(other)

    m = f"""{{
        "METRIC_NODE_ADDR": "{nodeId}",
        "METRIC_LOCATION": {loc},
        "METRIC_LOCATION_VICINITY": {{
            {visibleNodesLocation}
        }},
        {otherStrings}
    }}"""

    return m


sortingLocation = 0


def sortByDistance(n):
    return abs(nodesLocations[n] - sortingLocation)


def get_neighborhood(node):
    global nodes
    global sortingLocation
    nodesCopy = nodes[:]
    sortingLocation = nodesLocations[node]
    nodesCopy.sort(key=sortByDistance)
    neighSize = int(len(nodes)/2)
    return nodesCopy[:neighSize]


if len(sys.argv) < 4:
    print("usage: python3 generate_metrics.py output_dir number_of_services node1 [node2 node3 node4...]")

outputDir = sys.argv[1]
if not outputDir.endswith("/"):
    outputDir += "/"
numServices = int(sys.argv[2])
nodes = sys.argv[3:]
print("Number of services: ", numServices)
print("Nodes: ", nodes)

alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'

serviceLatencies = {}
processingTimes = {}
clientLocations = {}
services = []
nodesLocations = {}
nodesChildren = {}

minNodeLoc, maxNodeLoc = 1000, 10000

print("------------------------------------------ LOCATIONS ------------------------------------------")

for node in nodes:
    location = random.randint(minNodeLoc, maxNodeLoc)
    nodesLocations[node] = location
    print(f"{node} at {location}")

for idx in range(numServices):
    serviceName = alphabet[idx]
    services.append(serviceName)
    generateDictsForServiceTree(serviceName)

print("------------------------------------------ VISIBILITY ------------------------------------------")

for node in nodes:
    nodesVisible = get_neighborhood(node)
    print(f"{node} sees {nodesVisible}")
    nodeChildren = nodesVisible
    nodesChildren[node] = nodeChildren
    metrics = generateNodeMetrics(node, nodesLocations[node], nodesVisible, nodeChildren)
    with open(f"{outputDir}{node}.met", 'w') as nodeFp:
        parsed = json.loads(metrics)
        metrics = json.dumps(parsed, indent=4, sort_keys=False)
        nodeFp.write(metrics)

print("------------------------------------------ SERVICE TREES ------------------------------------------")

trees = []

for service in services:
    print(f"clients for {service} are at {clientLocations[service]}")
    randNode = nodes[random.randint(0, len(nodes) - 1)]
    trees.append(generateServiceTree(randNode, service))

treesString = "\n".join(trees)

print("------------------------------------------ FINAL TREES ------------------------------------------")

print(treesString)

with open(f"{outputDir}services.tree", 'w') as treeFp:
    treeFp.write(treesString)

locations = {"services": clientLocations, "nodes": nodesLocations}
with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/locations.txt", 'w') as locFp:
    locs = json.dumps(locations, indent=4, sort_keys=False)
    locFp.write(locs)

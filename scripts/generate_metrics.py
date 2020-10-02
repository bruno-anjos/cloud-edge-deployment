#!/usr/bin/python3

import json
import math
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


def calcDist(l1, l2):
    dX = l2["X"] - l1["X"]
    dY = l2["Y"] - l1["Y"]
    return math.sqrt(dX ** 2 + dY ** 2)


def sortChildren(n, sortingService, nodesLocations, clientLocations):
    return calcDist(nodesLocations[n], clientLocations[sortingService])


def generateServiceTree(startNode, sName, nodesChildren, nodesLocations, clientLocations):
    global sortingService

    sortingService = sName
    tree = f"{sName}: {startNode}"
    treeSize = 1

    better = True
    currNode = startNode

    while better:
        children = nodesChildren[currNode]
        if len(children) < 1:
            better = False
            continue
        candidates = []
        candidates.extend(children)
        candidates += [currNode]
        candidates.sort(key=lambda elem: sortChildren(elem, sName, nodesLocations, clientLocations))
        best = candidates[0]
        if best != currNode:
            tree += f" -> {best}"
            treeSize += 1
            currNode = best
        else:
            better = False

    return tree, treeSize


def generateDictsForServiceTree():
    minLatency, maxLatency = 100, 1000
    serviceLatency = random.randint(minLatency, maxLatency)

    minProcess, maxProcess = 0, 10
    processingTime = random.randint(minProcess, maxProcess)

    minCLoc, maxCLoc = 0, 10000
    serviceLocation = {"X": random.randint(minCLoc, maxCLoc), "Y": random.randint(minCLoc, maxCLoc)}

    return serviceLocation, processingTime, serviceLatency


def generateNodeMetrics(nodeId, loc, visibleNodes, children, nodesLocations, services, serviceLatencies,
                        processingTimes, clientLocations):
    visibleNodesLocation = ",\n".join([f""""{nodeId}": {{
        "X": {nodesLocations[nodeId]["X"]},
        "Y": {nodesLocations[nodeId]["Y"]}
    }}""" for nodeId in visibleNodes])

    other = []
    for s in services:
        other += [f"\"METRIC_LOAD_PER_SERVICE_{s}_IN_CHILDREN\": {{}}",
                  f"\"METRIC_AGG_LOAD_PER_SERVICE_{s}_IN_CHILDREN\": 0",
                  f"\"METRIC_CLIENT_LATENCY_PER_SERVICE_{s}\": {serviceLatencies[s]}",
                  f"\"METRIC_PROCESSING_TIME_PER_SERVICE_{s}\": {processingTimes[s]}",
                  f"\"METRIC_NUMBER_OF_INSTANCES_PER_SERVICE_{s}\": 0",
                  f"""\"METRIC_AVERAGE_CLIENT_LOCATION_PER_SERVICE_{s}\": {{
                        "X": {clientLocations[s]["X"]},
                        "Y": {clientLocations[s]["Y"]}
                    }}
                    """]
        for child in children:
            other += [f"\"METRIC_LOAD_PER_SERVICE_{s}_IN_CHILD_{child}\": 0"]

    otherStrings = ",\n".join(other)

    m = f"""{{
        "METRIC_NODE_ADDR": "{nodeId}",
        "METRIC_LOCATION": {{
            "X": {loc["X"]},
            "Y": {loc["Y"]}
        }},
        "METRIC_LOCATION_VICINITY": {{
            {visibleNodesLocation}
        }},
        {otherStrings}
    }}"""

    return m


sortingLocation = {"X": 0, "Y": 0}


def sortByDistance(n, nodesLocations):
    return calcDist(nodesLocations[n], sortingLocation)


def get_neighborhood(node, nodesLocations, neighSize):
    global nodes
    global sortingLocation
    nodesCopy = nodes[:]
    sortingLocation = nodesLocations[node]
    nodesCopy.sort(key=lambda elem: (sortByDistance(elem, nodesLocations)))
    return nodesCopy[:neighSize]


def gen_trees(neighSize):
    services = []

    nodesChildren = {}
    nodesLocations = {}

    minNodeLoc, maxNodeLoc = 0, 10000

    midNode = ""
    minDistToMid = -1
    midPoint = ((maxNodeLoc - minNodeLoc) / 2) + minNodeLoc
    for node in nodes:
        location = {"X": random.randint(minNodeLoc, maxNodeLoc), "Y": random.randint(minNodeLoc, maxNodeLoc)}
        nodesLocations[node] = location
        dist = calcDist(location, {"X":  midPoint, "Y": midPoint})
        if minDistToMid == -1 or dist < minDistToMid:
            midNode = node
            minDistToMid = dist

    clientLocations = {}
    processingTimes = {}
    serviceLatencies = {}
    for idx in range(numServices):
        serviceName = alphabet[idx]
        services.append(serviceName)
        serviceLocation, processingTime, serviceLatency = generateDictsForServiceTree()
        clientLocations[serviceName] = serviceLocation
        processingTimes[serviceName] = processingTime
        serviceLatencies[serviceName] = serviceLatency

    neighborhoods = {}
    for node in nodes:
        nodesVisible = get_neighborhood(node, nodesLocations, neighSize)
        neighborhoods[node] = nodesVisible
        nodeChildren = nodesVisible
        nodesChildren[node] = nodeChildren
        metrics = generateNodeMetrics(node, nodesLocations[node], nodesVisible, nodeChildren, nodesLocations,
                                      services, serviceLatencies, processingTimes, clientLocations)
        with open(f"{outputDir}{node}.met", 'w') as nodeFp:
            parsed = json.loads(metrics)
            metrics = json.dumps(parsed, indent=4, sort_keys=False)
            nodeFp.write(metrics)

    trees = []
    treeSizes = []

    fallback = midNode
    print(f"fallback is {fallback}")

    for service in services:
        tree, treeSize = generateServiceTree(fallback, service, nodesChildren, nodesLocations, clientLocations)
        trees.append(tree)
        treeSizes.append(treeSize)

    return trees, treeSizes, fallback, nodesLocations, nodesChildren, clientLocations, neighborhoods


if len(sys.argv) < 4:
    print("usage: python3 generate_metrics.py output_dir number_of_services prefix number_of_nodes")
    exit(1)

args = sys.argv[1:]

minTreeSize = 0
atLeastOneTreeSize = 0
hasOptions = True

idx = 0
idxsToIgnore = {}
for arg in args:
    if arg == "--min":
        minTreeSize = int(args[idx + 1])
        hasOptions = True
        idxsToIgnore[idx] = True
        idxsToIgnore[idx + 1] = True
    elif arg == "--at_least_one":
        atLeastOneTreeSize = int(args[idx + 1])
        hasOptions = True
        idxsToIgnore[idx] = True
        idxsToIgnore[idx + 1] = True
    idx += 1

newArgs = []
for i, arg in enumerate(args):
    if i not in idxsToIgnore:
        newArgs.append(arg)
args = newArgs

print("New args:", args)
print("Min tree size:", minTreeSize)
print("At least one tree size:", atLeastOneTreeSize)

outputDir = args[0]
if not outputDir.endswith("/"):
    outputDir += "/"
numServices = int(args[1])
prefix = args[2]
numberOfNodes = int(args[3])
print("Number of services: ", numServices)
print("Prefix: ", prefix)
print("Number of Nodes: ", numberOfNodes)

nodes = []
for i in range(numberOfNodes):
    nodes.append(prefix + str(i + 1))

print("Nodes: ", nodes)

filelist = os.listdir(outputDir)
for f in filelist:
    os.remove(os.path.join(outputDir, f))

alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'

done = False
trees = []

neighSize = int(len(nodes) / 10)
print(f"neighborhood size: {neighSize}")

while True:
    print("-------------------------------- TREE --------------------------------")

    trees, treeSizes, fallback, nodesLocations, nodesChildren, clientLocations, neighborhoods = gen_trees(neighSize)

    minMet = True
    atLeast = False

    maxSize = 0
    minSize = -1
    for treeSize in treeSizes:
        if treeSize < minTreeSize:
            minMet = False
        if treeSize >= atLeastOneTreeSize:
            atLeast = True
        if treeSize > maxSize:
            maxSize = treeSize
        if minSize == -1 or treeSize < minSize:
            minSize = treeSize

    print(f"min: {minSize}, max: {maxSize}")

    if minMet and atLeast:
        break

treesString = "\n".join(trees)

print("------------------------------------------ FINAL TREES ------------------------------------------")

print(treesString)

with open(f"{outputDir}services.tree", 'w') as treeFp:
    treeFp.write(treesString)

locations = {"services": clientLocations, "nodes": nodesLocations}
with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/locations.json", 'w') as locFp:
    locs = json.dumps(locations, indent=4, sort_keys=False)
    locFp.write(locs)

with open(f"{os.path.dirname(os.path.realpath(__file__))}/../build/deployer/fallback.txt", 'w') as fallbackFp:
    fallbackFp.write(fallback)

with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/neighborhoods.json", 'w') as neighsFp:
    for neighborhood in neighborhoods.values():
        del neighborhood[0]

    neighs = json.dumps(neighborhoods, indent=4, sort_keys=False)
    neighsFp.write(neighs)

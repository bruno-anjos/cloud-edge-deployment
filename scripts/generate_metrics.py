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
nodes_config_name = "NODES_CONFIG"
services_config_name = "SERVICES_CONFIG"
fallback_config_name = "FALLBACK_CONFIG"


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
    best = startNode

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

    print(f"closest node to {sName} is {best} at {nodesLocations[best]}")

    lastNode = best
    return tree, treeSize, lastNode


def generateServiceLatency():
    minLatency, maxLatency = 100, 1000
    return random.randint(minLatency, maxLatency)


def generateServiceProcessingTime():
    minProcess, maxProcess = 0, 10
    return random.randint(minProcess, maxProcess)


def generateDictsForServiceTree():
    processingTime = generateServiceProcessingTime()
    serviceLatency = generateServiceLatency()

    circle_r = 5000
    inner_circle_exclude = 3000
    circle_x = 10000 / 2
    circle_y = 10000 / 2

    alpha = 2 * math.pi * random.random()
    r = ((circle_r - inner_circle_exclude) * math.sqrt(random.random()) + inner_circle_exclude)
    x = int(r * math.cos(alpha) + circle_x)
    y = int(r * math.sin(alpha) + circle_y)

    serviceLocation = {"X": x, "Y": y}

    return serviceLocation, processingTime, serviceLatency


def generateNodeMetrics(nodeId, loc, visibleNodes, children, nodesLocations, services, serviceLatencies,
                        processingTimes, clientLocations, lastNode):
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
                  f"\"METRIC_LOAD_PER_SERVICE_{s}\": {2.7 if s in lastNode else 0.}",
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
    nodesCopy.remove(node)
    sortingLocation = nodesLocations[node]
    nodesCopy.sort(key=lambda elem: (sortByDistance(elem, nodesLocations)))

    return nodesCopy[:neighSize]


def loadChildrenFromConfig(node, config):
    global fromDummyToOriginal
    global fromOriginalToDummy

    originalNode = fromDummyToOriginal[node]
    originalChildren = config[originalNode]["neighbours"]
    children = []
    for child in originalChildren:
        children.append(fromOriginalToDummy[child])
    return children


def loadNodeLocationsFromConfig(config):
    global fromOriginalToDummy

    locations = {}
    for node, nodeConfig in config.items():
        originalNode = fromOriginalToDummy[node]
        x = nodeConfig["coords"][0]
        y = nodeConfig["coords"][1]
        locations[originalNode] = {"X": x, "Y": y}
    return locations


def calcMidNode(nodesLocations):
    midNode = ""
    minDistToMid = -1
    minNodeLoc, maxNodeLoc = 0, 10000

    midPoint = ((maxNodeLoc - minNodeLoc) / 2) + minNodeLoc

    for node, location in nodesLocations.items():
        dist = calcDist(location, {"X": midPoint, "Y": midPoint})
        if minDistToMid == -1 or dist < minDistToMid:
            midNode = node
            minDistToMid = dist
    return midNode


def generateLocations():
    nodesLocations = {}
    minNodeLoc, maxNodeLoc = 0, 10000

    midNode = ""
    minDistToMid = -1
    midPoint = ((maxNodeLoc - minNodeLoc) / 2) + minNodeLoc
    for node in nodes:
        location = {"X": random.randint(minNodeLoc, maxNodeLoc), "Y": random.randint(minNodeLoc, maxNodeLoc)}
        nodesLocations[node] = location
        dist = calcDist(location, {"X": midPoint, "Y": midPoint})
        if minDistToMid == -1 or dist < minDistToMid:
            midNode = node
            minDistToMid = dist

    return nodesLocations, midNode


def gen_services(numServices):
    services = []
    clientLocations = {}
    processingTimes = {}
    serviceLatencies = {}

    for idx in range(numServices):
        carry = idx // len(alphabet)
        alphaIdx = idx - carry * len(alphabet)
        serviceName = alphabet[alphaIdx] + alphabet[alphaIdx] * carry
        services.append(serviceName)
        serviceLocation, processingTime, serviceLatency = generateDictsForServiceTree()
        clientLocations[serviceName] = serviceLocation
        print(f"service {serviceName} is at {serviceLocation}")
        processingTimes[serviceName] = processingTime
        serviceLatencies[serviceName] = serviceLatency

    return services, clientLocations, processingTimes, serviceLatencies


def loadServicesFromConfig(servicesConfig):
    services = []
    clientLocations, processingTimes, serviceLatencies = {}, {}, {}

    for serviceName in servicesConfig:
        services.append(serviceName)
        serviceLocation = servicesConfig[serviceName]["location"]
        processingTime = generateServiceProcessingTime()
        serviceLatency = generateServiceLatency()
        clientLocations[serviceName] = serviceLocation
        processingTimes[serviceName] = processingTime
        serviceLatencies[serviceName] = serviceLatency

    return services, clientLocations, processingTimes, serviceLatencies


def gen_trees(numServices, neighSize, config):
    global nodes

    if config and config[nodes_config_name]:
        nodesLocations = loadNodeLocationsFromConfig(config[nodes_config_name])
        midNode = calcMidNode(nodesLocations)
    else:
        nodesLocations, midNode = generateLocations()

    if config and services_config_name in config:
        services, clientLocations, processingTimes, serviceLatencies = loadServicesFromConfig(
            config[services_config_name])
    else:
        services, clientLocations, processingTimes, serviceLatencies = gen_services(numServices)

    neighborhoods = {}
    nodesChildren = {}
    for node in nodes:
        if config:
            nodesVisible = loadChildrenFromConfig(node, config[nodes_config_name])
        else:
            nodesVisible = get_neighborhood(node, nodesLocations, neighSize)
        neighborhoods[node] = nodesVisible
        nodeChildren = nodesVisible
        nodesChildren[node] = nodeChildren

    trees = []
    treeSizes = []

    if fallback_config_name in config:
        fallback = fromOriginalToDummy[config[fallback_config_name]]
    else:
        fallback = midNode
    print(f"fallback is {fallback}")

    lastNodes = {}
    for service in services:
        tree, treeSize, lastNode = generateServiceTree(fallback, service, nodesChildren, nodesLocations,
                                                       clientLocations)
        print(f"{lastNode} is last node for service {service}")
        trees.append(tree)
        treeSizes.append(treeSize)
        if lastNode in lastNodes:
            lastNodes[lastNode].append(service)
        else:
            lastNodes[lastNode] = [service]

    for node in nodes:
        metrics = generateNodeMetrics(node, nodesLocations[node], neighborhoods[node], nodesChildren[node],
                                      nodesLocations, services, serviceLatencies, processingTimes, clientLocations,
                                      lastNodes[node] if node in lastNodes else [])
        with open(f"{outputDir}{node}.met", 'w') as nodeFp:
            parsed = json.loads(metrics)
            metrics = json.dumps(parsed, indent=4, sort_keys=False)
            nodeFp.write(metrics)

    return trees, treeSizes, fallback, nodesLocations, nodesChildren, clientLocations, neighborhoods


def loadConfig(nodes, nodesConfig, servicesConfig):
    fromOriginalToDummy = {}
    fromDummyToOriginal = {}
    loadedConfig = {}
    if nodesConfig:
        with open(nodesConfig, 'r') as configFp:
            loadedConfig[nodes_config_name] = json.load(configFp)
            print(f"config with {len(loadedConfig[nodes_config_name].keys())} nodes")
            for nodeId, dummy in zip(loadedConfig[nodes_config_name].keys(), nodes):
                print(f"{nodeId} -> {dummy}")
                fromOriginalToDummy[nodeId] = dummy
                fromDummyToOriginal[dummy] = nodeId
    if servicesConfig:
        with open(servicesConfig, 'r') as configFp:
            loadedConfig[services_config_name] = json.load(configFp)
            print(f"config with {len(loadedConfig[services_config_name].keys())} services")

    return fromOriginalToDummy, fromDummyToOriginal, loadedConfig


def writeFinalTree(trees, clientLocations, nodesLocations, outputDir, fallback, neighborhoods):
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
        neighs = json.dumps(neighborhoods, indent=4, sort_keys=False)
        neighsFp.write(neighs)


if len(sys.argv) < 4:
    print("usage: python3 generate_metrics.py output_dir number_of_services prefix number_of_nodes")
    exit(1)

args = sys.argv[1:]

minTreeSize = 0
atLeastOneTreeSize = 0
hasOptions = True

idx = 0
idxsToIgnore = {}
nodesConfig = ""
servicesConfig = ""
fallbackConfig = ""
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
    elif arg == "--nodes":
        nodesConfig = args[idx + 1]
        hasOptions = True
        idxsToIgnore[idx] = True
        idxsToIgnore[idx + 1] = True
    elif arg == "--services":
        servicesConfig = args[idx + 1]
        hasOptions = True
        idxsToIgnore[idx] = True
        idxsToIgnore[idx + 1] = True
    elif arg == "--fallback":
        fallbackConfig = args[idx + 1]
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
print("Number of services:", numServices)
print("Prefix:", prefix)
print("Number of Nodes:", numberOfNodes)

nodes = []
for i in range(numberOfNodes):
    nodes.append(prefix + str(i + 1))

fromOriginalToDummy, fromDummyToOriginal, loadedConfig = loadConfig(nodes, nodesConfig, servicesConfig)

if fallbackConfig:
    loadedConfig[fallback_config_name] = fallbackConfig

print("Nodes: ", nodes)

filelist = os.listdir(outputDir)
for f in filelist:
    os.remove(os.path.join(outputDir, f))

alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'

done = False
trees = []

neighSize = int(len(nodes) / 20)
print(f"neighborhood size: {neighSize}")

while True:
    print("-------------------------------- TREE --------------------------------")

    trees, treeSizes, fallback, nodesLocations, nodesChildren, \
    clientLocations, neighborhoods = gen_trees(numServices, neighSize, loadedConfig)

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

writeFinalTree(trees, clientLocations, nodesLocations, outputDir, fallback, neighborhoods)

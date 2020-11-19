#!../venv/bin/python3

import json
import os
import random
import sys

import s2sphere

"""
{
  "METRIC_NODE_ADDR": "node21",
  "METRIC_LOCATION": 5000.0,
  "METRIC_LOCATION_VICINITY": {
    "node21": 5000.0,
    "node23": 2000.0
  },
  "METRIC_NUMBER_OF_INSTANCES_PER_DEPLOYMENT_A": 0,
  "METRIC_LOAD_PER_DEPLOYMENT_A_IN_CHILD_node22": 0,
  "METRIC_LOAD_PER_DEPLOYMENT_A_IN_CHILD_node23": 0,
  "METRIC_LOAD_PER_DEPLOYMENT_A_IN_CHILDREN": {},
  "METRIC_AGG_LOAD_PER_DEPLOYMENT_A_IN_CHILDREN": 0,
  "METRIC_CLIENT_LATENCY_PER_DEPLOYMENT_A": 150,
  "METRIC_PROCESSING_TIME_PER_DEPLOYMENT_A": 10,
  "METRIC_AVERAGE_CLIENT_LOCATION_PER_DEPLOYMENT_A": 1000.0
}
"""

sortingService = ""
nodes_config_name = "NODES_CONFIG"
services_config_name = "DEPLOYMENTS_CONFIG"
fallback_config_name = "FALLBACK_CONFIG"


def sort_children(n, sorting_service, nodes_locations, client_locations):
    return calc_dist_in_lat_lng(nodes_locations[n], client_locations[sorting_service])


def generate_service_tree(start_node, service_name, nodes_children, nodes_locations, client_locations):
    global sortingService

    sortingService = service_name
    tree = f"{service_name}: {start_node}"
    tree_size = 1

    better = True
    curr_node = start_node
    best = start_node

    while better:
        children = nodes_children[curr_node]
        if len(children) < 1:
            better = False
            continue
        candidates = []
        candidates.extend(children)
        candidates += [curr_node]
        candidates.sort(key=lambda elem: sort_children(elem, service_name, nodes_locations, client_locations))
        best = candidates[0]
        if best != curr_node:
            tree += f" -> {best}"
            tree_size += 1
            curr_node = best
        else:
            better = False

    print(f"closest node to {service_name} is {best} at {nodes_locations[best]}")

    last_node = best
    return tree, tree_size, last_node


def generate_service_latency():
    min_latency, max_latency = 100, 1000
    return random.randint(min_latency, max_latency)


def generate_service_processing_time():
    min_process, max_process = 0, 10
    return random.randint(min_process, max_process)


def generate_dicts_for_service_tree():
    processing_time = generate_service_processing_time()
    service_latency = generate_service_latency()

    min_lat, max_lat, min_long, max_long = -90, 90, -180, 180

    service_location = {"lat": random.randrange(min_lat, max_lat), "lng": random.randint(min_long, max_long)}

    return service_location, processing_time, service_latency


def generate_node_metrics(node_id, loc, visible_nodes, children, nodes_locations, services, service_latencies,
                          processing_times, client_locations):
    visible_nodes_location = ",\n".join([
        f""""{nodeId}": "{s2sphere.CellId.from_lat_lng(s2sphere.LatLng.from_degrees(nodes_locations[nodeId]["lat"], nodes_locations[nodeId]["lng"])).to_token()}" """
        for nodeId in visible_nodes])

    other = []
    for s in services:
        cell_token = s2sphere.CellId.from_lat_lng(s2sphere.LatLng.from_degrees(client_locations[s]["lat"],
                                                                               client_locations[s]["lng"])).to_token()
        other += [f"\"METRIC_LOAD_PER_DEPLOYMENT_{s}_IN_CHILDREN\": {{}}",
                  f"\"METRIC_AGG_LOAD_PER_DEPLOYMENT_{s}_IN_CHILDREN\": 0",
                  f"\"METRIC_CLIENT_LATENCY_PER_DEPLOYMENT_{s}\": {service_latencies[s]}",
                  f"\"METRIC_PROCESSING_TIME_PER_DEPLOYMENT_{s}\": {processing_times[s]}",
                  f"\"METRIC_NUMBER_OF_INSTANCES_PER_DEPLOYMENT_{s}\": 0",
                  f"\"METRIC_LOAD_PER_DEPLOYMENT_{s}\": 0",
                  f"""\"METRIC_AVERAGE_CLIENT_LOCATION_PER_DEPLOYMENT_{s}\": "{cell_token}" """]
        for child in children:
            other += [f"\"METRIC_LOAD_PER_DEPLOYMENT_{s}_IN_CHILD_{child}\": 0"]

    other_strings = ",\n".join(other)

    m = f"""{{
        "METRIC_NODE_ADDR": "{node_id}",
        "METRIC_LOCATION": "{s2sphere.CellId.from_lat_lng(s2sphere.LatLng.from_degrees(loc["lat"], loc["lng"])).to_token()}",
        "METRIC_LOCATION_VICINITY": {{
            {visible_nodes_location}
        }},
        {other_strings}
    }}"""

    return m


sortingLocation = {"lat": 0, "lng": 0}


def calc_dist_in_lat_lng(loc1, loc2):
    l1_lat_lng = s2sphere.LatLng.from_degrees(loc1["lat"], loc1["lng"])
    l2_lat_lng = s2sphere.LatLng.from_degrees(loc2["lat"], loc2["lng"])

    return l1_lat_lng.get_distance(l2_lat_lng)


def sort_by_distance(n, nodes_locations):
    return calc_dist_in_lat_lng(nodes_locations[n], sortingLocation)


def get_neighborhood(node, nodes_locations, neigh_size):
    global nodes
    global sortingLocation

    nodes_copy = nodes[:]
    nodes_copy.remove(node)
    sortingLocation = nodes_locations[node]
    nodes_copy.sort(key=lambda elem: (sort_by_distance(elem, nodes_locations)))

    return nodes_copy[:neigh_size]


def load_children_from_config(node, config):
    global fromDummyToOriginal
    global fromOriginalToDummy

    original_node = fromDummyToOriginal[node]
    original_children = config[original_node]["neighbours"]
    children = []
    for child in original_children:
        children.append(fromOriginalToDummy[child])
    return children


def load_node_locations_from_config(config):
    global fromOriginalToDummy

    locations = {}
    for node, nodeConfig in config.items():
        original_node = fromOriginalToDummy[node]
        x = nodeConfig["coords"][0]
        y = nodeConfig["coords"][1]
        locations[original_node] = {"lat": x, "lng": y}
    return locations


def calc_mid_node(nodes_locations):
    mid_node = ""
    min_dist_to_mid = -1

    mid_point = {"lat": 0, "lng": 0}

    for node, location in nodes_locations.items():
        dist = calc_dist_in_lat_lng(location, mid_point)
        if min_dist_to_mid == -1 or dist < min_dist_to_mid:
            mid_node = node
            min_dist_to_mid = dist
    return mid_node


def generate_locations():
    nodes_locations = {}
    min_lat, max_lat, min_long, max_long = -90, 90, -180, 180

    mid_node = ""
    min_dist_to_mid = -1
    mid_point = {"lat": 0, "lng": 0}
    for node in nodes:
        location = {"lat": random.randint(min_lat, max_lat), "lng": random.randint(min_long, max_long)}
        nodes_locations[node] = location
        dist = calc_dist_in_lat_lng(location, mid_point)
        if min_dist_to_mid == -1 or dist < min_dist_to_mid:
            mid_node = node
            min_dist_to_mid = dist

    return nodes_locations, mid_node


def gen_services():
    client_locations = {}
    processing_times = {}
    service_latencies = {}

    for service_name in services:
        service_location, processing_time, service_latency = generate_dicts_for_service_tree()
        client_locations[service_name] = service_location
        print(f"service {service_name} is at {service_location}")
        processing_times[service_name] = processing_time
        service_latencies[service_name] = service_latency

    return client_locations, processing_times, service_latencies


def load_services_from_config(services_config):
    client_locations, processing_times, service_latencies = {}, {}, {}

    for serviceName in services_config:
        services.append(serviceName)
        service_location = services_config[serviceName]["location"]
        processing_time = generate_service_processing_time()
        service_latency = generate_service_latency()
        client_locations[serviceName] = service_location
        processing_times[serviceName] = processing_time
        service_latencies[serviceName] = service_latency

    return client_locations, processing_times, service_latencies


def gen_trees(neigh_size, loaded_config):
    if loaded_config and loaded_config[nodes_config_name]:
        nodes_locations = load_node_locations_from_config(loaded_config[nodes_config_name])
        mid_node = calc_mid_node(nodes_locations)
    else:
        nodes_locations, mid_node = generate_locations()

    if loaded_config and services_config_name in loaded_config:
        client_locations, processing_times, service_latencies = load_services_from_config(
            loaded_config[services_config_name])
    else:
        client_locations, processing_times, service_latencies = gen_services()

    new_neighborhoods = {}
    nodes_children = {}
    for node in nodes:
        if loaded_config:
            nodes_visible = load_children_from_config(node, loaded_config[nodes_config_name])
        else:
            nodes_visible = get_neighborhood(node, nodes_locations, neigh_size)
        new_neighborhoods[node] = nodes_visible
        node_children = nodes_visible
        nodes_children[node] = node_children

    new_trees = []
    tree_sizes = []

    if fallback_config_name in loaded_config:
        new_fallback = fromOriginalToDummy[loaded_config[fallback_config_name]]
    else:
        new_fallback = mid_node
    print(f"fallback is {new_fallback}")

    last_nodes = {}
    for service in services:
        tree, tree_size, last_node = generate_service_tree(new_fallback, service, nodes_children, nodes_locations,
                                                           client_locations)
        print(f"{last_node} is last node for service {service}")
        new_trees.append(tree)
        tree_sizes.append(tree_size)
        if last_node in last_nodes:
            last_nodes[last_node].append(service)
        else:
            last_nodes[last_node] = [service]

    for node in nodes:
        metrics = generate_node_metrics(node, nodes_locations[node], new_neighborhoods[node], nodes_children[node],
                                        nodes_locations, services, service_latencies, processing_times,
                                        client_locations)
        with open(f"{outputDir}{node}.met", 'w') as nodeFp:
            parsed = json.loads(metrics)
            metrics = json.dumps(parsed, indent=4, sort_keys=False)
            nodeFp.write(metrics)

    return new_trees, tree_sizes, new_fallback, nodes_locations, nodes_children, client_locations, new_neighborhoods


def load_config(nodes_config, services_config):
    from_original_to_dummy = {}
    from_dummy_to_original = {}
    loaded_config = {}
    if nodes_config:
        with open(nodes_config, 'r') as configFp:
            loaded_config[nodes_config_name] = json.load(configFp)
            print(f"config with {len(loaded_config[nodes_config_name].keys())} nodes")
            for nodeId, dummy in zip(loaded_config[nodes_config_name].keys(), nodes):
                print(f"{nodeId} -> {dummy}")
                from_original_to_dummy[nodeId] = dummy
                from_dummy_to_original[dummy] = nodeId
    if services_config:
        with open(services_config, 'r') as configFp:
            loaded_config[services_config_name] = json.load(configFp)
            print(f"config with {len(loaded_config[services_config_name].keys())} services")

    return from_original_to_dummy, from_dummy_to_original, loaded_config


def write_final_tree(nodes_locations, output_dir):
    trees_string = "\n".join(trees)

    print("------------------------------------------ FINAL TREES ------------------------------------------")

    print(trees_string)

    with open(f"{output_dir}services.tree", 'w') as treeFp:
        treeFp.write(trees_string)

    locations = {"services": clientLocations, "nodes": nodes_locations}
    with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/locations.json", 'w') as locFp:
        locs = json.dumps(locations, indent=4, sort_keys=False)
        locFp.write(locs)

    with open(f"{os.path.dirname(os.path.realpath(__file__))}/../build/deployer/fallback.txt", 'w') as fallbackFp:
        fallbackFp.write(fallback)

    with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/neighborhoods.json", 'w') as neighsFp:
        neighs = json.dumps(neighborhoods, indent=4, sort_keys=False)
        neighsFp.write(neighs)


if len(sys.argv) < 4:
    print("usage: python3 generate_metrics.py output_dir prefix number_of_nodes")
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
prefix = args[1]
numberOfNodes = int(args[2])
print("Prefix:", prefix)
print("Number of Nodes:", numberOfNodes)

nodes = []
for i in range(numberOfNodes):
    nodes.append(prefix + str(i + 1))

fromOriginalToDummy, fromDummyToOriginal, loadedConfig = load_config(nodesConfig, servicesConfig)

if fallbackConfig:
    loadedConfig[fallback_config_name] = fallbackConfig

print("Nodes: ", nodes)

filelist = os.listdir(outputDir)
for f in filelist:
    os.remove(os.path.join(outputDir, f))

done = False
trees = []

neighSize = int(len(nodes) / 20)
print(f"neighborhood size: {neighSize}")

services = []
with open(f"{os.path.dirname(os.path.realpath(__file__))}/launch_config.json", "r") as config_fp:
    configs = json.load(config_fp)
    for config in configs:
        service_name = config
        services.append(service_name)

while True:
    print("-------------------------------- TREE --------------------------------")

    trees, treeSizes, fallback, nodesLocations, nodesChildren, \
        clientLocations, neighborhoods = gen_trees(neighSize, loadedConfig)

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

write_final_tree(nodesLocations, outputDir)

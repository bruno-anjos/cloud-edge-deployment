#!/usr/bin/python3

import json
import os
import random

import s2sphere
import sys

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


def get_vicinity_metric(visible_nodes, nodes_locations):
    print(f"Vicinity: {visible_nodes}")

    vicinity_nodes = {}
    for nodeid in visible_nodes:
        vicinity_nodes[nodeid] = ids_to_nodes[nodeid]

    locations = {}
    for nodeid in visible_nodes:
        location = s2sphere.CellId.from_lat_lng(
            s2sphere.LatLng.from_degrees(
                nodes_locations[nodeid]["lat"],
                nodes_locations[nodeid]["lng"]
            )
        ).to_token()
        locations[nodeid] = location

    vicinity = {"Nodes": vicinity_nodes, "Locations": locations}
    return vicinity


def generate_node_metrics(node_id, loc, visible_nodes, nodes_locations):
    print(f"Generating metrics for {node_id}")

    vicinity = get_vicinity_metric(visible_nodes, nodes_locations)
    print(vicinity)

    m = f"""{{
        "METRIC_NODE_ADDR": "{node_id}",
        "METRIC_LOCATION": "{s2sphere.CellId.from_lat_lng(s2sphere.LatLng.from_degrees(loc["lat"], loc["lng"]))
        .to_token()}",
        "METRIC_LOCATION_VICINITY":
            {json.dumps(vicinity)}
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


def load_services_from_config(services_config):
    services = []
    client_locations, processing_times, service_latencies = {}, {}, {}

    for serviceName in services_config:
        services.append(serviceName)
        service_location = services_config[serviceName]["location"]
        processing_time = generate_service_processing_time()
        service_latency = generate_service_latency()
        client_locations[serviceName] = service_location
        processing_times[serviceName] = processing_time
        service_latencies[serviceName] = service_latency

    return services, client_locations, processing_times, service_latencies


def gen_trees(neigh_size, config):
    global nodes

    if config and config[nodes_config_name]:
        nodes_locations = load_node_locations_from_config(config[nodes_config_name])
        mid_node = calc_mid_node(nodes_locations)
    else:
        nodes_locations, mid_node = generate_locations()

    new_neighborhoods = {}
    nodes_children = {}
    for node in nodes:
        if config:
            nodes_visible = load_children_from_config(node, config[nodes_config_name])
        else:
            nodes_visible = get_neighborhood(node, nodes_locations, neigh_size)
        new_neighborhoods[node] = nodes_visible
        node_children = nodes_visible
        nodes_children[node] = node_children

    new_trees = []
    tree_sizes = []

    if fallback_config_name in config:
        new_fallback = ids_to_nodes[fromOriginalToDummy[config[fallback_config_name]]]
    else:
        new_fallback = ids_to_nodes[mid_node]
    print(f"fallback is {new_fallback}")

    for node in nodes:
        metrics = generate_node_metrics(node, nodes_locations[node], new_neighborhoods[node], nodes_locations)
        with open(f"{outputDir}{node}.met", 'w') as node_fp:
            parsed = json.loads(metrics)
            metrics = json.dumps(parsed, indent=4, sort_keys=False)
            node_fp.write(metrics)

    return new_trees, tree_sizes, new_fallback, nodes_locations, nodes_children, new_neighborhoods


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


def write_final_tree(trees_to_write, nodes_locations, output_dir):
    trees_string = "\n".join(trees_to_write)

    print("------------------------------------------ FINAL TREES ------------------------------------------")

    print(trees_string)

    with open(f"{output_dir}services.tree", 'w') as treeFp:
        treeFp.write(trees_string)

    locations = {"services": {}, "nodes": nodes_locations}
    with open(f"{os.path.dirname(os.path.realpath(__file__))}/visualizer/locations.json", 'w') as locFp:
        locs = json.dumps(locations, indent=4, sort_keys=False)
        locFp.write(locs)

    with open(f"{os.path.dirname(os.path.realpath(__file__))}/../build/deployer/fallback.json", 'w') as fallbackFp:
        json.dump(fallback, fallbackFp)

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
neighSize = 1
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
    elif arg == "--neigh":
        neighSize = int(args[idx + 1])
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

fromOriginalToDummy, fromDummyToOriginal, loadedConfig = load_config(nodesConfig, servicesConfig)

if fallbackConfig:
    loadedConfig[fallback_config_name] = fallbackConfig

print("Nodes: ", nodes)

ids_to_nodes = {}
for aux_node_id in nodes:
    aux_node = {"Id": aux_node_id}
    ids_to_nodes[aux_node_id] = aux_node

filelist = os.listdir(outputDir)
for f in filelist:
    os.remove(os.path.join(outputDir, f))

done = False

print(f"neighborhood size: {neighSize}")

print("-------------------------------- TREE --------------------------------")

trees, treeSizes, fallback, nodesLocations, nodesChildren, \
neighborhoods = gen_trees(neighSize, loadedConfig)

write_final_tree(trees, nodesLocations, outputDir)

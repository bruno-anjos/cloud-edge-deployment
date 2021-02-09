#!/usr/bin/python3
import json
import os
import random
import sys

import s2sphere

project_path = os.environ["CLOUD_EDGE_DEPLOYMENT"]


def calc_dist_in_lat_lng(loc1, loc2):
    l1_lat_lng = s2sphere.LatLng.from_degrees(loc1["lat"], loc1["lng"])
    l2_lat_lng = s2sphere.LatLng.from_degrees(loc2["lat"], loc2["lng"])

    return l1_lat_lng.get_distance(l2_lat_lng)


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


def load_locations():
    with open(f"{project_path}/latency_maps/{map_name}/nodes.txt", "r") as node_locations_fp:
        node_locations = {}
        lines = node_locations_fp.readlines()
        for aux_i, line in enumerate(lines):
            if aux_i >= number_nodes:
                break

            splitted = line.strip().split(" ")
            node_name = f"dummy{splitted[0]}"
            lat = splitted[1]
            lng = splitted[2]
            node_locations[node_name] = {
                "lat": float(lat),
                "lng": float(lng),
            }

    return node_locations, f"dummy{fallback_id}"


def gen_scenario():
    if map_name == "":
        locations, mid_node = generate_locations()
    else:
        locations, mid_node = load_locations()

    scenario = {
        "locations": locations,
        "fallback": mid_node,
        "duration": duration
    }

    scen_path = f"{os.path.expanduser('~/ced-scenarios')}/{output_filename}.json"

    print(f"Writing to {scen_path} scenario: {scenario}")

    with open(scen_path, "w") as scenario_fp:
        json.dump(scenario, scenario_fp)


args = sys.argv[1:]

if len(args) < 2:
    print("usage: python3 generate_scenario.py <number_of_nodes> <output_file> [--duration 10m] [--mapname map2] "
          "[--fallback node10]")
    exit(1)

number_nodes_string = ""
output_filename = ""
map_name = ""
fallback_id = ""
duration = ""

skip = False

for i, arg in enumerate(args):
    if skip:
        skip = False
        continue
    if arg == "--mapname":
        map_name = args[i + 1]
        skip = True
    elif arg == "--fallback":
        fallback_id = args[i + 1]
        skip = True
    elif arg == "--duration":
        duration = args[i + 1]
        skip = True
    elif number_nodes_string == "":
        number_nodes_string = arg
    elif output_filename == "":
        output_filename = arg
    else:
        print(f"Unrecognized option {arg}")
        exit(1)

number_nodes = int(number_nodes_string)

print("Number of Nodes:", number_nodes)
print("Output filename:", output_filename)
if map_name != "":
    if fallback_id == "":
        print("When providing --mapname please also provide the number for the fallback node. e.g. --fallback 2")
    print(f"Map name: {map_name}")

nodes = []
for i in range(number_nodes):
    nodes.append("dummy" + str(i + 1))

print("Nodes: ", nodes)

gen_scenario()

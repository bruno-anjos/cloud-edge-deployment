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


def gen_locations(nodes):
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


def load_locations(map_name, number_nodes, fallback_id):
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


def gen_scenario(map_name, nodes, number_nodes, fallback_id, output_filename, cells_to_region_path, region_delays_path):
    if map_name == "":
        locations, mid_node = gen_locations(nodes)
    else:
        locations, mid_node = load_locations(map_name, number_nodes, fallback_id)

    scenario = {
        "locations": locations,
        "fallback": mid_node,
    }

    if cells_to_region_path != "":
        latencies = gen_latencies(nodes, locations, cells_to_region_path, region_delays_path)
        scenario["latencies"] = latencies

    scen_path = f"{os.path.expanduser('~/ced-scenarios')}/{output_filename}.json"

    print(f"Writing to {scen_path} scenario: {scenario}")

    with open(scen_path, "w") as scenario_fp:
        json.dump(scenario, scenario_fp)


def gen_latencies(nodes, locations, cells_to_region_path, region_delays_path):
    with open(os.path.expanduser(cells_to_region_path), 'r') as f:
        cells_to_region = json.load(f)

    node_regions = {}

    for node in nodes:
        loc = locations[node]
        cell_id = s2sphere.CellId.from_lat_lng(s2sphere.LatLng.from_degrees(loc["lat"], loc["lng"]))
        top_cell = cell_id.parent(1)
        node_regions[node] = cells_to_region[top_cell.to_token()]

    with open(os.path.expanduser(region_delays_path), 'r') as f:
        region_delays = json.load(f)

    latencies = []

    for node in nodes:
        region = node_regions[node]
        node_latencies = []

        for other_node in nodes:
            if other_node == node:
                node_latencies.append(0)
                continue

            other_region = node_regions[other_node]

            delay = region_delays[region][other_region]
            node_latencies.append(delay)

        latencies.append(node_latencies)

    return latencies


def main():
    args = sys.argv[1:]

    if len(args) < 2:
        print("usage: python3 generate_scenario.py <number_of_nodes> <output_file> [--mapname map2] "
              "[--fallback node10] [--cells_to_region cells_to_region.json] [--region_delays region_delays.json]")
        exit(1)

    number_nodes_string = ""
    output_filename = ""
    map_name = ""
    fallback_id = ""
    cells_to_region_path = ""
    region_delays_path = ""

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
        elif arg == "--cells_to_region":
            cells_to_region_path = args[i + 1]
            skip = True
        elif arg == "--region_delays":
            region_delays_path = args[i + 1]
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
            exit(1)
        print(f"Map name: {map_name}")

    if (cells_to_region_path != "" and region_delays_path == "") or (
            cells_to_region_path == "" and region_delays_path != ""):
        print("When using one of --cells_to_region or --region_delays both have to be provided")
        exit(1)

    nodes = []
    for i in range(number_nodes):
        nodes.append("dummy" + str(i + 1))

    print("Nodes: ", nodes)

    gen_scenario(map_name, nodes, number_nodes, fallback_id, output_filename, cells_to_region_path,
                 region_delays_path)


if __name__ == '__main__':
    main()

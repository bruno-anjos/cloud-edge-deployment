#!/usr/bin/python3
import json
import os
import subprocess
import sys
from multiprocessing import Pool

import s2sphere

dummy_network = "dummies-network"
project_path = os.path.expanduser("~/go/src/github.com/bruno-anjos/cloud-edge-deployment")


def del_everything():
    print("Deleting everything...")
    list_containers_cmd = "docker ps -a -q"
    containers = " ".join(subprocess.getoutput(list_containers_cmd).split("\n"))
    remove_cmd = f"docker stop {containers} ; docker rm {containers} ; docker network rm {dummy_network} ; docker " \
                 f"volume prune -f ; docker system prune -f"
    subprocess.run(remove_cmd, shell=True)


def build_dummy_node_image():
    print("Building dummy node image...")
    build_cmd = f"bash {project_path}/build/dummy_node/build_dummy_node.sh"
    subprocess.run(build_cmd, shell=True)


def create_network():
    print("Creating network...")
    create_network_cmd = f"docker network create --subnet=192.168.192.1/20 {dummy_network}"
    subprocess.run(create_network_cmd, shell=True)


NAME = "name"
NODE_IP = "ip"
LOCATION = "location"


def launch_dummy(info):
    launch_cmd = f'docker run -d --network=dummies-network --privileged --ip {info[NODE_IP]} --name=' \
                 f'{info[NAME]}  --hostname {info[NAME]} --env NODE_IP="{info[NODE_IP]}" --env ' \
                 f'NODE_ID="{info[NAME]}" --env LOCATION="{info[LOCATION]}" brunoanjos/dummy_node:latest'
    subprocess.run(launch_cmd, shell=True)


def start_services_in_dummy(info):
    start_services_cmd = f"docker exec {info[NAME]} ./deploy_services.sh"
    subprocess.run(start_services_cmd, shell=True)


def build_dummy_infos(num, s2_locs):
    print("Building dummy nodes infos...")

    infos = []
    for i in range(1, num + 1):
        carry = i // 255
        remainder = i % 255
        name = f"dummy{i}"
        ip = f"192.168.19{3 + carry}.{remainder}"
        location = s2_locs[name]

        dummy_info = {
            NAME: name,
            NODE_IP: ip,
            LOCATION: location,
        }

        print(f"{name}: {ip} | {location}")
        infos.append(dummy_info)

    return infos


def load_s2_locations():
    print("Loading s2 locations...")
    with open(f"{project_path}/scripts/visualizer/locations.json", 'r') as locations_fp:
        locations = json.load(locations_fp)["nodes"]

    s2_locs = {}
    for dummy_name, location in locations.items():
        s2_locs[dummy_name] = s2sphere.CellId.from_lat_lng(
            s2sphere.LatLng.from_degrees(
                location["lat"],
                location["lng"]
            )
        ).to_token()

    print(s2_locs)

    return s2_locs


del_everything()
build_dummy_node_image()
create_network()

args = sys.argv[1:]
if len(args) != 1:
    print("usage: deploy_dummy_stack.sh num_nodes")

num_nodes = int(args[0])

s2_locations = load_s2_locations()
dummy_infos = build_dummy_infos(num_nodes, s2_locations)

pool = Pool(processes=os.cpu_count())
pool.map(launch_dummy, dummy_infos)
pool.map(start_services_in_dummy, dummy_infos)

pool.close()
pool.terminate()

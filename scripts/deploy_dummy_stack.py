#!/usr/bin/python3
import json
import os
import socket
import subprocess
import sys
from multiprocessing import Pool

import s2sphere

dummy_network = "swarm-network"
project_path = os.path.expanduser("~/go/src/github.com/bruno-anjos/cloud-edge-deployment")
nodes = subprocess.getoutput("oarprint host").strip().split("\n")
host = socket.gethostname().strip()


def exec_cmd_on_nodes(cmd):
    for node in nodes:
        if node == host:
            subprocess.run(cmd, shell=True)
        else:
            exec_cmd_on_node(cmd, node)


def exec_cmd_on_node(cmd, node):
    remote_cmd = f"oarsh {node} -- {cmd}"
    subprocess.run(remote_cmd)


def del_everything():
    print("Deleting everything...")
    list_containers_cmd = "docker service ls -q -a"
    containers = " ".join(subprocess.getoutput(list_containers_cmd).split("\n"))
    remove_cmd = f"docker service rm {containers} ; docker network rm {dummy_network}"
    subprocess.run(remove_cmd, shell=True)
    exec_cmd_on_nodes("docker volume prune -f ; docker system prune -f")


def build_dummy_node_image():
    print("Building dummy node image...")
    build_cmd = f"bash {project_path}/build/dummy_node/build_dummy_node.sh"
    subprocess.run(build_cmd, shell=True)


NAME = "name"
NODE_IP = "ip"
LOCATION = "location"


def launch_dummy(info):
    launch_cmd = f'docker service create --replicas 1 --name {info[NAME]} --cap-add CAP_NET_ADMIN ' \
                 f'--cap-add CAT_SYS_ADMIN ' \
                 f'--network=swarm-network --hostname {info[NAME]} --env NODE_IP="{info[NODE_IP]}" ' \
                 f'--env NODE_ID="{info[NAME]}" --env LOCATION="{info[LOCATION]}" brunoanjos/dummy_node:latest'
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

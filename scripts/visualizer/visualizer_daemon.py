#!/usr/bin/python3
import json
import os
import subprocess
import sys
import time
from multiprocessing import Pool


def run_cmd_with_try(cmd):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=subprocess.DEVNULL)

    failed = False
    if cp.stderr is not None:
        failed = True
        print(f"StdErr: {cp.stderr}")
    if cp.returncode != 0:
        failed = True
        print(f"RetCode: {cp.returncode}")
    if failed:
        exit(1)


def exec_cmd_on_host(node, cmd):
    path_var = os.environ["PATH"]
    remote_cmd = f"oarsh {node} -- 'PATH=\"{path_var}\" && {cmd}'"
    run_cmd_with_try(remote_cmd)


def get_all_hierarchy_tables(count):
    tables = {}
    for node in nodes:
        table_path = f"{recordings_path}/{node}/{count}.json"
        with open(table_path, 'r') as table_fp:
            tables[node] = json.load(table_fp)

    return tables


def collect_recordings(aux_output_dir, aux_duration, timeout):
    cycles = int(aux_duration[:-1]) // int(timeout[:-1])
    snapshots_path = aux_output_dir

    for count in range(cycles):
        target_path = f"{snapshots_path}/graph_{count}.json"
        node_tables = get_all_hierarchy_tables(count+1)

        print("Got all hierarchy tables!")

        # add all connections
        deployment_colors = {}
        i = 0

        # graphs = {}

        deployments = set()
        for node in node_tables:
            for deployment_id in node_tables[node]:
                if deployment_id == "dead":
                    continue
                deployments.add(deployment_id)
                # graphs[deployment_id] = Graph(directed=True)
                if deployment_id not in deployment_colors:
                    color = colors[i % len(colors)]
                    deployment_colors[deployment_id] = color
                    i += 1

        aux_locations = {"nodes": {}, "services": {}}

        services = [file_name for file_name in os.listdir(services_path)]
        for service in services:
            with open(f"{services_path}/{service}", 'r') as service_fp:
                service_loc = json.load(service_fp)
                if service not in aux_locations["services"]:
                    aux_locations["services"][service] = service_loc

        for node in node_tables:
            loc = get_location(node)
            aux_locations["nodes"][node] = loc

        graph_json = {
            "node_tables": node_tables,
            "colors": deployment_colors,
            "services": services,
            "locations": aux_locations
        }

        with open(target_path, 'w', encoding='utf-8') as graph_json_fp:
            json.dump(graph_json, graph_json_fp, ensure_ascii=False, indent=4)

        print(f"saved graph.json to {target_path}")


def get_location(name):
    if name in locations["services"]:
        return locations["services"][name]
    elif name in locations["nodes"]:
        return locations["nodes"][name]
    else:
        print(f"{name} has no location in {locations}")


def start_recording(aux_duration, timeout):
    infos = []
    for node in nodes:
        url = (deployerURLf % dummy_infos[node]["ip"]) + startRecordingPath
        infos.append((url, f"Duration={aux_duration} Timeout={timeout}"))

    pool.map(exec_req, infos)


def exec_req(info):
    (url, content) = info

    docker_cmd = ['docker', 'exec', '-t', 'vis_entry', 'http', '-b', url]
    docker_cmd.extend(content.split(" "))
    subprocess.run(docker_cmd)
    return


def remove_visualizer_entrypoint():
    cmd = "docker stop vis_entry"
    subprocess.run(cmd, shell=True)
    cmd = "docker rm vis_entry"
    subprocess.run(cmd, shell=True)


def setup_visualizer_entrypoint():
    cmd = f'docker run -itd --entrypoint /bin/sh --network="swarm-network" --rm --name="vis_entry" alpine/httpie'
    subprocess.run(cmd, shell=True)


def process_time_string(time_string):
    time_in_seconds = int(time_string[:-1])

    time_suffix = time_string[-1]
    if time_suffix == "m" or time_suffix == "M":
        time_in_seconds = time_in_seconds * 60
    elif time_suffix == "h" or time_suffix == "H":
        time_in_seconds = time_in_seconds * 60 * 60

    return time_in_seconds


print("Removing old entrypoint...")
remove_visualizer_entrypoint()
print("Done!")

print("Setting up entrypoint...")
setup_visualizer_entrypoint()
print("Done!")

deployerURLf = 'http://%s:1502/deployer'
archimedesURLf = 'http://%s:1500/archimedes'
tablePath = '/table'
startRecordingPath = '/start_recording'
services_path = "/tmp/services"

args = sys.argv[1:]

if len(args) > 5:
    print("usage: python3 visualizer_daemon.py <scenario_filename> <time_between_snapshots> <duration_in_seconds> ["
          "--output_dir=] [--collect_only]")
    exit(1)

scenario_filename = ""
time_between = ""
duration = ""
output_dir = os.path.expanduser('~/snapshots')
collect_only = False

for arg in args:
    if "--output_dir=" in arg:
        output_dir = arg.split("--output_dir=")[1]
    elif "--collect_only" == arg:
        collect_only = True
    elif scenario_filename == "":
        scenario_filename = arg
    elif time_between == "":
        time_between = arg
    elif duration == "":
        duration = arg

with open(f"{os.path.expanduser('~/ced-scenarios')}/{scenario_filename}", 'r') as scenario_fp:
    scenario = json.load(scenario_fp)

nodes = scenario["locations"].keys()

print("Got nodes: ", nodes)

if os.path.exists("/home/b.anjos/results/results.json"):
    os.remove("/home/b.anjos/results/results.json")

for f in os.listdir("/home/b.anjos/deployer_pngs/"):
    os.remove(os.path.join("/home/b.anjos/deployer_pngs/", f))

fallback = scenario["fallback"]
locations = {"services": {}, "nodes": scenario["locations"]}

with open(f"/tmp/dummy_infos.json", "r") as dummy_infos_fp:
    infos_list = json.load(dummy_infos_fp)

    dummy_infos = {}
    for info in infos_list:
        dummy_infos[info["name"]] = info

# CONSTS
attr_child = "child"
attr_parent = "parent"
attr_grandparent = "grandparent"
attr_neigh = "neigh"

parent_field_id = "Parent"
grandparent_field_id = "Grandparent"
children_field_id = "Children"
node_id_field_id = "Id"
orphan_field_id = "IsOrphan"

# GRAPH PROPERTIES
colors = ["blue", "pink", "green", "orange", "dark blue", "brown", "dark green"]
arrow_width_dict = {attr_grandparent: 3, attr_parent: 1, attr_child: 1, attr_neigh: 0.5}
edge_width_dict = {attr_grandparent: 1, attr_parent: 1, attr_child: 3, attr_neigh: 0.5}

pool = Pool(os.cpu_count())

if not collect_only:
    start_recording(duration, time_between)

    time.sleep(process_time_string(duration))

recordings_path = os.path.expanduser("~/tables/")

hosts = subprocess.getoutput("oarprint host").strip().split("\n")
for host in hosts:
    exec_cmd_on_host(host, f'cp -R /tmp/tables/* {recordings_path}')

collect_recordings(output_dir, duration, time_between)

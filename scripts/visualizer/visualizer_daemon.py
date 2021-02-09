#!/usr/bin/python3
import json
import os
import subprocess
import sys
import time
from multiprocessing import Pool

from tabulate import tabulate


def exec_req(url):
    cmd = f"docker exec -t vis_entry http -b {url}"
    result = subprocess.getoutput(cmd).strip()
    return json.loads(result)


def get_services_table(node):
    url = (archimedesURLf % dummy_infos[node]["ip"]) + tablePath

    try:
        table_resp = exec_req(url)
        return table_resp
    except json.JSONDecodeError:
        return {}


def get_hierarchy_table(node):
    url = (deployerURLf % dummy_infos[node]["ip"]) + tablePath

    try:
        table_resp = exec_req(url)
        return node, table_resp
    except json.JSONDecodeError:
        return node, {"dead": True}


def get_all_hierarchy_tables():
    tables_aux = {}

    results = pool.map(get_hierarchy_table, nodes)
    for res in results:
        node, table = res
        tables_aux[node] = table

    return tables_aux


def get_all_services_tables():
    tables_aux = {}
    for nodeAux in nodes:
        table_aux = get_services_table(nodeAux)
        if table_aux:
            tables_aux[nodeAux] = table_aux
    return tables_aux


def get_load(deployment_id, node):
    autonomic_ulr_f = "http://%s:50000/archimedes"
    load_path = "/deployments/%s/load"

    url = (autonomic_ulr_f % dummy_infos[node]["ip"]) + (load_path % deployment_id)

    try:
        table_resp = exec_req(url)
        return table_resp
    except json.JSONDecodeError:
        return 0.


"""
table json schema:

{
    "deploymentId": {
        "parent": {
            "id": "",
            "addr": ""
        },
        "grandparent": {
            "id": "",
            "addr": ""
        },
        "children": {
            "childId": {
                "id": "",
                "addr": ""
            }
        },
        "isStatic": False,
        "isOrphan": False
    }
}
"""


def add_if_missing(graph, table, node):
    nodes_aux = [name for name in graph.vs['name']]
    if node in nodes_aux:
        return
    if "dead" in table:
        graph.add_vertex(node, color="black", service=False)


def get_load_for_given_node(node_table, node, aux_pool):
    processes = {}
    loads = []

    for deployment_id in node_table.keys():
        processes[deployment_id] = aux_pool.apply_async(get_load, (deployment_id, node))

    for deployment_id, p in processes.items():
        load = p.get()
        load_string = f"{deployment_id}: {load}"
        loads.append(load_string)

    return loads


def graph_deployer(target_path):
    print("creating graph for deployers")

    node_tables = get_all_hierarchy_tables()

    print("Got all hierarchy tables!")

    loads = {}
    for node in node_tables:
        aux_loads = get_load_for_given_node(node_tables[node], node, pool)
        loads[node] = aux_loads

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
        with open(f"{services_path}/{service}") as service_fp:
            service_loc = json.load(service_fp)
            if service not in aux_locations["services"]:
                aux_locations["services"][service] = service_loc

    for node in node_tables:
        loc = get_location(node)
        aux_locations["nodes"][node] = loc

    graph_json = {
        "node_tables": node_tables,
        "colors": deployment_colors,
        "loads": loads,
        "services": services,
        "locations": aux_locations
    }

    with open(target_path, 'w') as graph_json_fp:
        json.dump(graph_json, graph_json_fp)

    # deployer_processes = {}
    # for deployment_id in deployments:
    #     deployer_processes[deployment_id] = pool.apply_async(graph_deployment, (
    #         deployment_id, graphs[deployment_id], node_tables, deployment_colors[deployment_id], loads))

    # resulting_trees = {}
    # for deployment_id, dp in deployer_processes.items():
    #     res = dp.get()
    #     if not dp.successful():
    #         print(f"error with {deployment_id}: {res}")
    #         return
    #     resulting_trees[deployment_id] = res

    # with open(f"/home/b.anjos/results/results.json", "w") as results_fp:
    #     print("writing results.json")
    #     results = json.dumps(resulting_trees, indent=4, sort_keys=False)
    #     results_fp.write(results)

    # if not has_tables:
    #     mypath = "/home/b.anjos/deployer_pngs/"
    #     onlyfiles = [os.path.join(mypath, f) for f in os.listdir(mypath) if os.path.isfile(os.path.join(mypath, f))]
    #     print(f"deleting {onlyfiles}")
    #     for file in onlyfiles:
    #         os.remove(file)

    print(f"saved graph.json to {target_path}")


def get_location(name):
    if name in locations["services"]:
        return locations["services"][name]
    elif name in locations["nodes"]:
        return locations["nodes"][name]
    else:
        print(f"{name} has no location in {locations}")


def graph_archimedes():
    s_tables = get_all_services_tables()

    entries_field_id = "Entries"
    instances_field_id = "Instances"
    initialized_field_id = "Initialized"
    static_field_id = "Static"
    local_field_id = "Local"
    max_hops_field_id = "MaxHops"

    tab_headers = ["ServiceId", "Hops", "MaxHops", "Version", "InstanceId", "Initialized", "Static", "Local"]

    latex_filename = "/home/b.anjos/archimedes_tables.tex"
    latex_file = open(latex_filename, "w")
    latex_file.write("\\documentclass{article}\n"
                     "\\nonstopmode\n"
                     "\\begin{document}\n"
                     "\\title{Archimedes tables}\n"
                     "\\maketitle\n")

    for node, sTable in s_tables.items():
        table = [tab_headers]
        if sTable[entries_field_id]:
            for serviceId, entry in sTable[entries_field_id].items():
                first = True
                for instanceId, instance in entry[instances_field_id].items():
                    row = ["", "", "", ""]
                    if first:
                        row = [serviceId, entry[max_hops_field_id]]
                        first = False
                    row.extend([instanceId, instance[initialized_field_id], instance[static_field_id],
                                instance[local_field_id]])
                    table.append(row)
            latex_file.write("\\begin{center}\n")
            latex_file.write("NODE: %s\n\n" % node)
            latex_file.write(tabulate(table, headers="firstrow", tablefmt="latex"))
            latex_file.write("\\end{center}\n\n")

    latex_file.write("\n\\end{document}")
    latex_file.close()
    print("wrote archimedes latex file")


def setup_visualizer_entrypoint():
    cmd = f'docker run -itd --entrypoint /bin/sh --network="swarm-network" --rm --name="vis_entry" alpine/httpie'
    subprocess.run(cmd, shell=True)


deployerURLf = 'http://%s:50002/deployer'
archimedesURLf = 'http://%s:50000/archimedes'
tablePath = '/table'
services_path = "/tmp/services"

args = sys.argv[1:]

if len(args) != 2:
    print("usage: python3 visualizer_daemon.py <scenario_filename> <time_between_snapshots>")

scenario_filename = args[0]
time_between = args[1]

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


def remove_visualizer_entrypoint():
    cmd = "docker stop vis_entry"
    subprocess.run(cmd, shell=True)
    cmd = "docker rm vis_entry"
    subprocess.run(cmd, shell=True)


print("Removing old entrypoint...")
remove_visualizer_entrypoint()
print("Done!")

print("Setting up entrypoint...")
setup_visualizer_entrypoint()
print("Done!")

time_between_in_seconds = int(time_between[:-1])

time_suffix = time_between[-1]
if time_suffix == "m" or time_suffix == "M":
    time_between_in_seconds = time_between_in_seconds * 60
elif time_suffix == "h" or time_suffix == "H":
    time_between_in_seconds = time_between_in_seconds * 60 * 60

pool = Pool(processes=os.cpu_count())
while True:
    start = time.time()
    timestamp = int(start)
    path = f"{os.path.expanduser('~/snapshots')}/{timestamp}"
    print(f"saving snapshot to {path}")
    graph_deployer(path)
    time_took = time.time() - start

    remaining_time = time_between_in_seconds - time_took
    if remaining_time > 0:
        time.sleep(remaining_time)

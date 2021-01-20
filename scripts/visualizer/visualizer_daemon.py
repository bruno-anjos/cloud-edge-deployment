#!/usr/bin/python3
import json
import logging
import os
import subprocess
import sys
import time
from multiprocessing import Pool

from igraph import Graph, plot
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


def graph_combined_deployments(graph, node_tables, deployment_colors, loads):
    try:
        for node, table in node_tables.items():
            if "dead" in table:
                graph.add_vertex(node, color="black", service=False)
            else:
                graph.add_vertex(node, color="red", service=False)

        services = [file_name for file_name in os.listdir(services_path)]
        for service in services:
            with open(f"{services_path}/{service}") as service_fp:
                service_loc = json.load(service_fp)
                graph.add_vertex(service, color="yellow", service=True)
                if service not in locations["services"]:
                    locations["services"][service] = service_loc

        for node, auxTable in node_tables.items():
            if not auxTable or "dead" in auxTable:
                continue
            for deployment_id, entry in auxTable.items():
                for child_id in entry[children_field_id].keys():
                    print(f"edge from {node} to {child_id} for {deployment_id}")
                    add_if_missing(graph, node_tables[child_id], child_id)
                    graph.add_edge(node, child_id, deployment_id=deployment_id, relation=attr_child)

        visual_style = {}
        labels = []
        colors = []

        for vertex in graph.vs:
            if not vertex["service"]:
                label = vertex["name"]
                if vertex["name"] in loads:
                    label = label + f"\n{', '.join(loads[label])}"
            else:
                label = ""

            labels.append(label)
            colors.append(vertex["color"])

        graph.vs["label"] = labels
        visual_style["vertex_size"] = 10
        visual_style["vertex_color"] = colors
        visual_style["vertex_label"] = graph.vs["label"]
        visual_style["vertex_label_dist"] = 2
        visual_style["vertex_label_size"] = 10
        visual_style["vertex_shape"] = ["triangle-up" if service else "circle" for service in
                                        graph.vs["service"]]
        if len(graph.es) > 0:
            visual_style["edge_color"] = [deployment_colors[deployment_id]
                                          for deployment_id in graph.es['deployment_id']]
            visual_style["edge_width"] = 3

        layout = []
        for node in graph.vs["name"]:
            loc = get_location(node)
            layout.append((loc["lng"], loc["lat"]))
        visual_style["layout"] = layout
        visual_style["bbox"] = (4000, 4000)
        visual_style["margin"] = 200
        print("plotting combined")
        plot(graph, f"/home/b.anjos/deployer_pngs/combined_plot.png", **visual_style, autocurve=True)
    except Exception as e:
        logging.exception(e)


def graph_deployer():
    print("creating graph for deployers")

    node_tables = get_all_hierarchy_tables()
    loads = {}

    for node in node_tables:
        loads[node] = []
        for deployment_id in node_tables[node].keys():
            load = get_load(deployment_id, node)
            load_string = f"{deployment_id}: {load}"
            loads[node].append(load_string)

    # add all connections
    deployment_colors = {}
    i = 0

    graphs = {}
    combined_graph = Graph(directed=True)

    deployments = set()
    for node in node_tables:
        for deployment_id in node_tables[node]:
            if deployment_id == "dead":
                continue
            deployments.add(deployment_id)
            graphs[deployment_id] = Graph(directed=True)
            if deployment_id not in deployment_colors:
                color = colors[i % len(colors)]
                deployment_colors[deployment_id] = color
                i += 1

    # deployer_processes = {}
    # for deployment_id in deployments:
    #     deployer_processes[deployment_id] = pool.apply_async(graph_deployment, (
    #         deployment_id, graphs[deployment_id], node_tables, deployment_colors[deployment_id], loads))

    combined = pool.apply_async(graph_combined_deployments,
                                (combined_graph, node_tables, deployment_colors, loads))

    # resulting_trees = {}
    # for deployment_id, dp in deployer_processes.items():
    #     res = dp.get()
    #     if not dp.successful():
    #         print(f"error with {deployment_id}: {res}")
    #         return
    #     resulting_trees[deployment_id] = res

    combined.wait()

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

    print("finished creating graphs")
    sys.stdout.flush()
    sys.stderr.flush()


def get_location(name):
    if name in locations["services"]:
        return transform_loc_to_range(locations["services"][name])
    elif name in locations["nodes"]:
        return transform_loc_to_range(locations["nodes"][name])
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


def transform_loc_to_range(loc):
    new_loc = {"lat": 4000 - (((loc["lat"] + 90) * 4000) / 180), "lng": (((loc["lng"] + 180) * 4000) / 360)}
    return new_loc


def setup_visualizer_entrypoint():
    cmd = f'docker run -itd --entrypoint /bin/sh --network="swarm-network" --rm --name="vis_entry" alpine/httpie'
    subprocess.run(cmd, shell=True)


deployerURLf = 'http://%s:50002/deployer'
archimedesURLf = 'http://%s:50000/archimedes'
tablePath = '/table'
services_path = "/tmp/services"

args = sys.argv[1:]

if len(args) < 1:
    print("usage: python3 visualizer_daemon.py scenario_filename")

scenario_filename = args[0]

with open(f"{os.path.expanduser('~/ced-scenarios')}/{scenario_filename}.json", 'r') as scenario_fp:
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


remove_visualizer_entrypoint()
setup_visualizer_entrypoint()

pool = Pool(processes=os.cpu_count())
while True:
    graph_deployer()
    graph_archimedes()

    time.sleep(5)

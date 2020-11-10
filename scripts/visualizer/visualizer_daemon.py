#!/usr/bin/python3
import json
import os
import sys
import time
from multiprocessing import Pool

import requests
from igraph import Graph, plot
from tabulate import tabulate


def check_resp_and_get_json(resp, url):
    status = resp.status_code
    if status != 200:
        print("ERROR: got status %d for url %s" % (status, url))
        exit(1)
    table_resp = resp.json()
    return table_resp


def get_services_table(node_arg, number):
    global dummy

    if dummy:
        node_num = number % 255
        carry = number // 255
        url = (archimedesURLf % (dummyContainerFormatf % (3 + carry, node_num))) + tablePath
    else:
        url = (archimedesURLf % node_arg) + tablePath

    try:
        table_resp = check_resp_and_get_json(requests.get(url), url)
        return table_resp
    except requests.ConnectionError:
        return {}


def get_hierarchy_table(node_arg, number):
    if dummy:
        node_num = number % 255
        carry = number // 255
        url = (deployerURLf % (dummyContainerFormatf % (3 + carry, node_num))) + tablePath
    else:
        url = (deployerURLf % node_arg) + tablePath

    try:
        table_resp = check_resp_and_get_json(requests.get(url), url)
        return table_resp
    except requests.ConnectionError:
        return {"dead": True}


def get_all_hierarchy_tables():
    tables_aux = {}
    for idx, nodeAux in enumerate(nodes):
        table_aux = get_hierarchy_table(nodeAux, idx + 1)
        tables_aux[nodeAux] = table_aux
    return tables_aux


def get_all_services_tables():
    tables_aux = {}
    for idx, nodeAux in enumerate(nodes):
        table_aux = get_services_table(nodeAux, idx + 1)
        if table_aux:
            tables_aux[nodeAux] = table_aux
    return tables_aux


def get_load(deployment_id, node_arg, number):
    autonomic_ulr_f = "http://%s:50000/archimedes"
    load_path = "/deployments/%s/load"

    if dummy:
        node_num = number % 255
        carry = number // 255
        url = (autonomic_ulr_f % (dummyContainerFormatf % (3 + carry, node_num))) + (load_path % deployment_id)
    else:
        url = (autonomic_ulr_f % node_arg) + (load_path % deployment_id)

    try:
        table_resp = check_resp_and_get_json(requests.get(url), url)
        return table_resp
    except requests.ConnectionError:
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


def graph_combined_deployments(graph, deployments, node_tables, deployment_colors, loads):
    for node, table in node_tables.items():
        if "dead" in table:
            graph.add_vertex(node, color="black", service=False)
        else:
            graph.add_vertex(node, color="red", service=False)

    for deployment_id in deployments:
        graph.add_vertex(deployment_id, color=deployment_colors[deployment_id], service=True)

    for node, auxTable in node_tables.items():
        if not auxTable or "dead" in auxTable:
            continue
        for deployment_id, entry in auxTable.items():
            for child_id in entry[children_field_id].keys():
                add_if_missing(graph, node_tables[child_id], child_id)
                graph.add_edge(node, child_id, deployment_id=deployment_id, relation=attr_child)

    visual_style = {}
    graph.vs["label"] = [name + f"\n{', '.join(loads[name])}" if name in loads else name for name in
                         graph.vs["name"]]
    visual_style["vertex_size"] = 10
    visual_style["vertex_color"] = [color for color in graph.vs["color"]]
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
    plot(graph, f"/home/b.anjos/deployer_pngs/combined_plot.png", **visual_style, autocurve=True)


def graph_deployment(deployment_id, graph, node_tables, deployment_color, loads):
    for node, table in node_tables.items():
        if "dead" in table:
            graph.add_vertex(node, color="black", service=False)
        else:
            graph.add_vertex(node, color="red", service=False)

    graph.add_vertex(deployment_id, color=deployment_color, service=True)

    resulting_tree = {}

    for node, node_table in node_tables.items():
        if not node_table or "dead" in node_table or deployment_id not in node_table:
            continue

        entry = node_table[deployment_id]

        for neigh in neighborhoods[node]:
            add_if_missing(graph, node_tables[neigh], neigh)
            graph.add_edge(node, neigh, relation=attr_neigh, deployment_id=attr_neigh)

        if entry[parent_field_id] is not None:
            parent = entry[parent_field_id]
            parent_id = parent[node_id_field_id]
            add_if_missing(graph, node_tables[parent_id], parent_id)
            graph.add_edge(node, parent_id, relation=attr_parent,
                           deployment_id=deployment_id)
        if entry[grandparent_field_id] is not None:
            grandparent = entry[grandparent_field_id]
            grandparent_id = grandparent[node_id_field_id]
            if grandparent_id != "":
                add_if_missing(graph, node_tables[grandparent_id], grandparent_id)
                graph.add_edge(node, grandparent_id, relation=attr_grandparent,
                               deployment_id=deployment_id)
        for childId in entry[children_field_id].keys():
            add_if_missing(graph, node_tables[childId], childId)
            graph.add_edge(node, childId, relation=attr_child, deployment_id=deployment_id)
            if node in resulting_tree:
                resulting_tree[node].append(childId)
            else:
                resulting_tree[node] = [childId]

    visual_style = {}
    graph.vs["label"] = [name + f"\n{', '.join(loads[name])}" if name in loads else name for name in graph.vs["name"]]
    visual_style["vertex_size"] = 10
    visual_style["vertex_color"] = [color for color in graph.vs["color"]]
    visual_style["vertex_label"] = graph.vs["label"]
    visual_style["vertex_label_dist"] = 2
    visual_style["vertex_label_size"] = 10
    visual_style["vertex_shape"] = ["triangle-up" if service else "circle" for service in
                                    graph.vs["service"]]
    if len(graph.es) > 0:
        visual_style["edge_color"] = ["black" if deployment_id == attr_neigh else deployment_color for deployment_id in
                                      graph.es['deployment_id']]
        visual_style["edge_arrow_width"] = [arrow_width_dict[relation] for relation in graph.es["relation"]]
        visual_style["edge_width"] = [edge_width_dict[relation] for relation in graph.es["relation"]]

    layout = []
    for node in graph.vs["name"]:
        loc = get_location(node)
        layout.append((loc["lng"], loc["lat"]))

    visual_style["bbox"] = (4000, 4000)
    visual_style["margin"] = 200
    visual_style["layout"] = layout
    graph_filename = f"/home/b.anjos/deployer_pngs/deployer_plot_{deployment_id}.png"
    plot(graph, graph_filename, **visual_style, autocurve=True)
    print(f"plotted {graph_filename}")
    sys.stdout.flush()

    return resulting_tree


def graph_deployer():
    print("creating graph for deployers")

    node_tables = get_all_hierarchy_tables()
    loads = {}

    for node in node_tables:
        loads[node] = []
        node_number = int(node.split("dummy")[1])
        for deployment_id in node_tables[node].keys():
            load = get_load(deployment_id, node, node_number)
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

    deployer_processes = {}
    for deployment_id in deployments:
        deployer_processes[deployment_id] = pool.apply_async(graph_deployment, (
            deployment_id, graphs[deployment_id], node_tables, deployment_colors[deployment_id], loads))

    pool.apply_async(graph_combined_deployments, (combined_graph, deployments, node_tables, deployment_colors, loads))

    resulting_trees = {}
    for deployment_id, dp in deployer_processes.items():
        dp.wait()
        if not dp.successful():
            print(f"error with {deployment_id}: {dp.get()}")
            return
        resulting_trees[deployment_id] = dp.get()

    with open(f"/home/b.anjos/results/results.json", "w") as resultsFp:
        print("writing results.json")
        results = json.dumps(resulting_trees, indent=4, sort_keys=False)
        resultsFp.write(results)

    # if not has_tables:
    #     mypath = "/home/b.anjos/deployer_pngs/"
    #     onlyfiles = [os.path.join(mypath, f) for f in os.listdir(mypath) if os.path.isfile(os.path.join(mypath, f))]
    #     print(f"deleting {onlyfiles}")
    #     for file in onlyfiles:
    #         os.remove(file)

    print("finished creating graphs")
    sys.stdout.flush()


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
    hops_field_id = "NumberOfHops"
    max_hops_field_id = "MaxHops"
    version_field_id = "Version"

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
                        row = [serviceId, entry[hops_field_id], entry[max_hops_field_id],
                               entry[version_field_id]]
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


deployerURLf = 'http://%s:50002/deployer'
archimedesURLf = 'http://%s:50000/archimedes'
dummyContainerFormatf = "192.168.19%d.%d"
tablePath = '/table'

args = sys.argv[1:]

if len(args) < 2:
    print("usage: python3 visualizer_daemon.py prefix number_of_nodes")

dummy = False

prefix = ""
numNodes = 0

nodes = []
for arg in args:
    if arg == "--dummy":
        print("running in dummy mode")
        dummy = True
    elif prefix == "":
        prefix = arg
    else:
        numNodes = int(arg)

for num in range(numNodes):
    nodes.append(prefix + str(num + 1))

print("Got nodes: ", nodes)

if os.path.exists("/home/b.anjos/results/results.json"):
    os.remove("/home/b.anjos/results/results.json")

for f in os.listdir("/home/b.anjos/deployer_pngs/"):
    os.remove(os.path.join("/home/b.anjos/deployer_pngs/", f))

with open(f"{os.path.dirname(os.path.realpath(__file__))}/../../build/deployer/fallback.txt", 'r') as fallbackFp:
    fallback = fallbackFp.readline()

with open(f"{os.path.dirname(os.path.realpath(__file__))}/neighborhoods.json", 'r') as neighsFp:
    neighborhoods = json.load(neighsFp)

with open(f"{os.path.dirname(os.path.realpath(__file__))}/locations.json", 'r') as f:
    locations = json.load(f)

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

while True:
    pool = Pool(processes=os.cpu_count())

    pool.apply_async(graph_archimedes, ())
    graph_deployer()

    pool.close()
    pool.join()
    time.sleep(5)

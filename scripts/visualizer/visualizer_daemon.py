#!/usr/bin/python3
import json
import os
import sys
import time

import igraph
import requests
from tabulate import tabulate

deployerURLf = 'http://%s:50002/deployer'
archimedesURLf = 'http://%s:50000/archimedes'
tablePath = '/table'

nodes = sys.argv[1:]
print("Got nodes: ", nodes)


def check_resp_and_get_json(resp):
    status = resp.status_code
    if status != 200:
        print("ERROR: got status %d" % status)
        exit(1)
    table_resp = resp.json()
    # print("got ", table_resp)
    return table_resp


def get_services_table(node_arg):
    print("----------------------------------- %s -----------------------------------" % node_arg)
    print("requesting %s services table" % node_arg)

    try:
        table_resp = check_resp_and_get_json(requests.get((archimedesURLf % node_arg) + tablePath))
        return table_resp
    except requests.ConnectionError:
        return {}


def get_hierarchy_table(node_arg):
    print("----------------------------------- %s -----------------------------------" % node_arg)
    print("requesting %s hierarchy table" % node_arg)
    try:
        table_resp = check_resp_and_get_json(requests.get((deployerURLf % node_arg) + tablePath))
        return table_resp
    except requests.ConnectionError:
        return {}


def get_all_hierarchy_tables():
    tables_aux = {}
    for nodeAux in nodes:
        table_aux = get_hierarchy_table(nodeAux)
        tables_aux[nodeAux] = table_aux
    return tables_aux


def get_all_services_tables():
    tables_aux = {}
    for nodeAux in nodes:
        table_aux = get_services_table(nodeAux)
        if table_aux:
            tables_aux[nodeAux] = table_aux
    return tables_aux


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


def add_if_missing(graph, nodes, node):
    if node not in nodes:
        graph.add_vertex(node, color="black")


def graph_deployer():
    print("creating graph for deployers")
    attr_child = "child"
    attr_parent = "parent"
    attr_grandparent = "grandparent"

    parent_field_id = "Parent"
    grandparent_field_id = "Grandparent"
    children_field_id = "Children"
    node_id_field_id = "Id"
    orphan_field_id = "IsOrphan"

    colors = ["blue", "pink", "red", "black", "green", "gray", "orange"]
    arrow_width_dict = {attr_grandparent: 3, attr_parent: 1, attr_child: 1}
    edge_width_dict = {attr_grandparent: 1, attr_parent: 1, attr_child: 3}
    orphan_dict = {}

    tables = get_all_hierarchy_tables()

    # add all connections
    deployment_colors = {}
    i = 0
    has_edges = False
    has_tables = False

    graphs = {}
    inited_deployments = {}

    for node, auxTable in tables.items():
        if not auxTable:
            continue
        has_tables = True
        print("—--------------------------------------------")
        for deploymentId, entry in auxTable.items():
            if deploymentId in graphs:
                g = graphs[deploymentId]
            else:
                g = igraph.Graph(directed=True)
                graphs[deploymentId] = g

            if deploymentId not in inited_deployments:
                # first add all nodes
                for auxNode in tables.keys():
                    g.add_vertex(auxNode, color="red")

                with open(f"{os.path.dirname(os.path.realpath(__file__))}/locations.txt", 'r') as f:
                    locations = json.load(f)
                    g.add_vertex(deploymentId, color="blue")
                inited_deployments[deploymentId] = True

            if entry[orphan_field_id]:
                if node in orphan_dict:
                    orphan_dict[node].append(deploymentId)
                else:
                    orphan_dict[node] = [deploymentId]

            if deploymentId not in deployment_colors:
                color = colors[i % len(colors)]
                deployment_colors[deploymentId] = color
                i += 1
            if entry[parent_field_id] is not None:
                parent = entry[parent_field_id]
                parentId = parent[node_id_field_id]
                print(f"({deploymentId}) {node} has parent {parentId}")
                add_if_missing(g, tables.keys(), parentId)
                g.add_edge(node, parentId, relation=attr_parent,
                           deploymentId=deploymentId)
                has_edges = True
            if entry[grandparent_field_id] is not None:
                grandparent = entry[grandparent_field_id]
                grandparentId = grandparent[node_id_field_id]
                print(f"({deploymentId}) {node} has grandparent {grandparentId}")
                add_if_missing(g, tables.keys(), grandparentId)
                g.add_edge(node, grandparentId, relation=attr_grandparent,
                           deploymentId=deploymentId)
                has_edges = True
            for childId in entry[children_field_id].keys():
                print(f"({deploymentId}) {node} has child {childId}")
                add_if_missing(g, tables.keys(), childId)
                g.add_edge(node, childId, relation=attr_child, deploymentId=deploymentId)
                has_edges = True

    for deploymentId, g in graphs.items():
        visual_style = {}
        layout = g.layout_auto()
        g.vs["label"] = [name + f"\n({get_location(name, locations)})\n(orphan): " + ",".join(orphan_dict[name])
                         if name in orphan_dict
                         else name + f"\n({get_location(name, locations)})" for name in g.vs["name"]]
        visual_style["vertex_size"] = 30
        visual_style["vertex_color"] = [color for color in g.vs["color"]]
        visual_style["vertex_label"] = g.vs["label"]
        visual_style["vertex_label_dist"] = 4
        visual_style["vertex_label_size"] = 16
        visual_style["vertex_shape"] = ["triangle-up" if color == "blue" else "circle" for color in
                                        g.vs["color"]]
        if has_edges:
            visual_style["edge_color"] = deployment_colors[deploymentId]
            visual_style["edge_arrow_width"] = [arrow_width_dict[relation] for relation in g.es["relation"]]
            visual_style["edge_width"] = [edge_width_dict[relation] for relation in g.es["relation"]]
        visual_style["layout"] = layout
        visual_style["bbox"] = (1000, 1000)
        visual_style["margin"] = 200
        igraph.plot(g, f"/home/b.anjos/deployer_pngs/deployer_plot_{deploymentId}.png", **visual_style, autocurve=True)

    if not has_tables:
        mypath = "/home/b.anjos/deployer_pngs/"
        onlyfiles = [os.path.join(mypath, f) for f in os.listdir(mypath) if os.path.isfile(os.path.join(mypath, f))]
        print(f"deleting {onlyfiles}")
        for file in onlyfiles:
            os.remove(file)


def get_location(name, locations):
    if name in locations["services"]:
        return locations["services"][name]
    elif name in locations["nodes"]:
        return locations["nodes"][name]


def graph_archimedes():
    sTables = get_all_services_tables()
    entries_field_id = "Entries"
    instances_field_id = "Instances"
    initialized_field_id = "Initialized"
    static_field_id = "Static"
    local_field_id = "Local"
    hops_field_id = "NumberOfHops"
    max_hops_field_id = "MaxHops"
    version_field_id = "Version"

    tabHeaders = ["ServiceId", "Hops", "MaxHops", "Version", "InstanceId", "Initialized", "Static", "Local"]

    latex_filename = "/home/b.anjos/archimedes_tables.tex"
    latex_file = open(latex_filename, "w")
    latex_file.write("\\documentclass{article}\n"
                     "\\nonstopmode\n"
                     "\\begin{document}\n"
                     "\\title{Archimedes tables}\n"
                     "\\maketitle\n")

    for node, sTable in sTables.items():
        table = [tabHeaders]
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


while True:
    graph_deployer()
    graph_archimedes()
    time.sleep(5)

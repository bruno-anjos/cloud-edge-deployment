#!/usr/bin/python3

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
        if table_aux:
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

    colors = ["blue", "pink", "red", "black", "yellow", "green", "gray", "orange"]
    arrow_width_dict = {"grandparent": 2, "parent": 1, "child": 1}
    edge_width_dict = {"grandparent": 1, "parent": 1, "child": 3}
    edge_curved_dict = {"grandparent": 0.5, "parent": False, "child": 0.5}
    orphan_dict = {}

    plot_filename = "/home/b.anjos/deployer_plot.png"

    tables = get_all_hierarchy_tables()
    g = igraph.Graph(directed=True)

    # first add all nodes
    for node in tables.keys():
        g.add_vertex(node, absent=False)

    print("tables ", tables)

    # add all connections
    deployment_colors = {}
    i = 0
    has_edges = False
    empty_nodes = {}
    for node, table in tables.items():
        for deploymentId, entry in table.items():
            if entry[orphan_field_id]:
                if node in orphan_dict:
                    orphan_dict[node].append(deploymentId)
                else:
                    orphan_dict[node] = [deploymentId]

            if deploymentId not in deployment_colors:
                color = colors[i]
                deployment_colors[deploymentId] = color
                i += 1
            if entry[parent_field_id] is not None:
                parent = entry[parent_field_id]
                print("%s has parent %s" % (node, parent))
                parentId = parent[node_id_field_id]
                if not (parentId in tables.keys() or parentId in empty_nodes):
                    empty_nodes[parentId] = True
                    g.add_vertex(parentId, absent=True)
                g.add_edge(node, parentId, relation=attr_parent,
                           deploymentId=deploymentId)
                has_edges = True
            if entry[grandparent_field_id] is not None:
                grandparent = entry[grandparent_field_id]
                print("%s has grandparent %s" % (node, grandparent))
                grandparentId = grandparent[node_id_field_id]
                if not (grandparentId in tables.keys() or grandparentId in empty_nodes):
                    empty_nodes[grandparentId] = True
                    g.add_vertex(grandparentId, absent=True)
                g.add_edge(node, grandparentId, relation=attr_grandparent,
                           deploymentId=deploymentId)
                has_edges = True
            for childId in entry[children_field_id].keys():
                print("%s has child %s" % (node, childId))
                if not (childId in tables.keys() or childId in empty_nodes):
                    empty_nodes[childId] = True
                    g.add_vertex(childId, absent=True)
                g.add_edge(node, childId, relation=attr_child, deploymentId=deploymentId)
                has_edges = True

    layout = g.layout("tree")
    visual_style = {}

    if len(tables) > 0:
        g.vs["label"] = [name + " (orphan): " + ",".join(orphan_dict[name]) if name in orphan_dict else name for name in
                         g.vs["name"]]
        visual_style["vertex_size"] = 20
        visual_style["vertex_color"] = ["black" if absent else "red" for absent in g.vs["absent"]]
        visual_style["vertex_label"] = g.vs["name"]
        visual_style["vertex_label_dist"] = 2
        if has_edges:
            visual_style["edge_color"] = [deployment_colors[deploymentId] for deploymentId in g.es["deploymentId"]]
            visual_style["edge_arrow_width"] = [arrow_width_dict[relation] for relation in g.es["relation"]]
            visual_style["edge_width"] = [edge_width_dict[relation] for relation in g.es["relation"]]
            visual_style["edge_curved"] = [edge_curved_dict[relation] for relation in g.es["relation"]]
        visual_style["layout"] = layout
        visual_style["bbox"] = (1000, 1000)
        visual_style["margin"] = 50

    igraph.plot(g, plot_filename, **visual_style)


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
        table = []
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
            latex_file.write(tabulate(table, headers=tabHeaders, tablefmt="latex"))
            latex_file.write("\\end{center}\n\n")

    latex_file.write("\n\\end{document}")
    latex_file.close()
    print("wrote archimedes latex file")


while True:
    graph_deployer()
    graph_archimedes()
    time.sleep(5)

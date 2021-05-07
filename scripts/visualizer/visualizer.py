import json
import os

import sys
from graph_tool.all import Graph, graph_draw, find_vertex

archimedes_tex_filename = os.path.expanduser("~/archimedes_tables.tex")
archimedes_tex_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.tex"
archimedes_pdf_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.pdf"
archimedes_out_local_path = "/Users/banjos/Desktop/archimedes_tables/"
archimedes_png_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.png"

results_tree_filename = "dicluster:~/results/results.json"
results_tree_local_path = "/Users/banjos/Desktop/deployer_pngs/results.json"
graph_json_local_path = "/Users/banjos/Desktop/deployer_pngs/graph.json"

wait = 5

blue = [0., 0., 1., 1.]
pink = [1., 0., 0.5, 1.]
green = [0., 1., 0., 1.]
orange = [1., 0.5, 0., 1.]
dark_blue = [0., 0., 0.5, 1.]
brown = [0.25, 0.25, 0., 1.]
dark_green = [0., 0., 0.4, 1.]

colors = [blue, pink, green, orange, dark_blue, brown, dark_green]


def add_if_missing(g, node_to_v, vprop_text, vprop_color, vprop_fill_color, table, node):
    res = find_vertex(g, vprop_text, node)
    if len(res) > 0:
        return
    if "dead" in table:
        v = g.add_vertex(node)
        node_to_v[node] = v
        vprop_color[v] = [0., 0., 0., 1.]
        vprop_fill_color[v] = [0., 0., 0., 1.]


def get_location(name, locations):
    if name in locations["services"]:
        return transform_loc_to_range(locations["services"][name])
    elif name in locations["nodes"]:
        return transform_loc_to_range(locations["nodes"][name])
    else:
        print(f"{name} has no location in {locations}")


def transform_loc_to_range(loc):
    new_loc = {"lat": 4000 - (((loc["lat"] + 90) * 4000) / 180),
               "lng": (((loc["lng"] + 180) * 4000) / 360)}
    return new_loc


def graph_combined_deployments(dir):
    stats_dir = f'{dir}/stats'
    files = [file for file in os.listdir(stats_dir) if "graph" in file]

    for file in files:
        graph = Graph(directed=True)

        with open(f"{stats_dir}/{file}", 'r') as graph_fp:
            graph_json = json.load(graph_fp)

        node_tables = graph_json["node_tables"]

        node_to_vertices = {}

        vprop_text = graph.new_vertex_property("string")
        vprop_color = graph.new_vertex_property("vector<double>")
        vprop_fill_color = graph.new_vertex_property("vector<double>")
        vprop_shape = graph.new_vertex_property("string")

        eprop_color = graph.new_edge_property("vector<double>")

        for node, table in node_tables.items():
            v = graph.add_vertex()

            node_to_vertices[node] = v
            vprop_text[v] = node
            vprop_shape[v] = "circle"
            if "dead" in table:
                vprop_color[v] = [0., 0., 0., 1.]
                vprop_fill_color[v] = [0., 0., 0., 1.]
            else:
                vprop_color[v] = [0., 0., 0., 0.25]
                vprop_fill_color[v] = [1., 0., 0., 0.5]

        services_to_v = {}
        services = graph_json["services"]
        for service in services:
            v = graph.add_vertex()
            vprop_color[v] = [0., 0., 0., 0.25]
            vprop_fill_color[v] = [1., 1., 0., 1.]
            vprop_shape[v] = "triangle"
            services_to_v[service] = v

        deployment_colors = {}
        i = 0

        for node, auxTable in node_tables.items():
            if not auxTable or "dead" in auxTable:
                continue
            for deployment_id, entry in auxTable.items():
                for child_id in entry["Children"].keys():
                    add_if_missing(graph, node_to_vertices, vprop_text, vprop_color, vprop_fill_color, node_tables[
                        child_id], child_id)
                    s, t = node_to_vertices[node], node_to_vertices[child_id]

                    aux_e = graph.add_edge(s, t)
                    if deployment_id not in deployment_colors:
                        color = colors[i % len(colors)]
                        deployment_colors[deployment_id] = color
                        i += 1

                    color = deployment_colors[deployment_id]
                    eprop_color[aux_e] = color

        locations = graph_json["locations"]
        positions = graph.new_vertex_property("vector<double>")
        for node, v in node_to_vertices.items():
            loc = get_location(node, locations)
            positions[v] = [loc["lng"], loc["lat"]]

        for service, v in services_to_v.items():
            loc = get_location(service, locations)
            positions[v] = [loc["lng"], loc["lat"]]

        print(f"Plotting {file} combined graph with {len(graph.get_vertices())} nodes and {len(graph.get_edges())} "
              f"edges")

        vprops = {
            "text": vprop_text,
            "color": vprop_color,
            "fill_color": vprop_fill_color,
            "shape": vprop_shape,
        }

        out_filename = file.split(".")[0]

        graph_draw(graph, pos=positions, output_size=(4000, 4000), vertex_size=10,
                   output=f"{dir}/plots/{out_filename}.png",
                   bg_color=[1., 1., 1., 1.], vertex_fill_color=vprop_fill_color, edge_color=eprop_color,
                   fit_view=True, adjust_aspect=True, vprops=vprops, vertex_text_color=[0., 0., 0., 1.],
                   vertex_font_size=14)


def main():
    args = sys.argv[1:]
    if len(args) != 1:
        print("usage: python3 visualizer.py <experiment_dir>")
        exit(1)

    experiment_dir = args[0]
    graph_combined_deployments(experiment_dir)


if __name__ == '__main__':
    main()

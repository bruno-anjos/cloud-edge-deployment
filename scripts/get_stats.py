#!/usr/bin/python3
import json
import os
import socket
import subprocess
import sys

import matplotlib.colors as colors
import matplotlib.pyplot as plt

client_logs_dirname = "client_logs"

timetook_regex = r"took (\d+)"

TIMESTAMP = "TIMESTAMP"
BYTES_OUT = "BYTES_OUT"
BYTES_IN = "BYTES_IN"
BYTES_TOTAL = "BYTES_TOTAL"


def run_cmd_with_try(cmd):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=subprocess.DEVNULL)
    if cp.stderr is not None:
        raise Exception(cp.stderr)


def exec_cmd_on_node(node, cmd):
    path_var = os.environ["PATH"]
    remote_cmd = f"oarsh {node} -- 'PATH=\"{path_var}\" && {cmd}'"
    run_cmd_with_try(remote_cmd)


def clean_folder_nas():
    cmd = f"rm -f {os.path.expanduser('~/bandwidth_stats/*')}"
    subprocess.run(cmd.split(" "))


def sync_stats_to_nas(dummy_infos):
    hostname = socket.gethostname()

    nodes = {}
    for info in dummy_infos:
        node = info["node"]
        nodes[node] = None

    cp_to_nas_cmd = f"cp /tmp/bandwidth_stats/* {os.path.expanduser('~/bandwidth_stats/')}"
    for node in nodes:
        if node == hostname:
            run_cmd_with_try(cp_to_nas_cmd)
        else:
            exec_cmd_on_node(node, cp_to_nas_cmd)

    print("Synced stats to NAS")


def get_bandwidth_stats():
    bandwidth_dir = f"{os.path.expanduser('~/bandwidth_stats')}"
    node_results = {}

    for node in os.listdir(bandwidth_dir):
        node_results[node] = []

        with open(f"{bandwidth_dir}/{node}") as bandwidth_fp:
            measures = 0

            for line in bandwidth_fp.readlines():
                measures += 1
                splits = line.split(";")

                timestamp = splits[0]
                bytes_out_s = splits[2]
                bytes_in_s = splits[3]
                bytes_total_s = splits[4]

                node_results[node].append({
                    TIMESTAMP: timestamp,
                    BYTES_OUT: bytes_out_s,
                    BYTES_IN: bytes_in_s,
                    BYTES_TOTAL: bytes_total_s
                })

            print(f"{node} has {measures} measures")

    return node_results


def find_lowest_timestamp(node_results):
    lowest_timestamp = -1

    for _, results in node_results.items():
        for result in results:
            if lowest_timestamp == -1 or lowest_timestamp > float(result[TIMESTAMP]):
                lowest_timestamp = float(result[TIMESTAMP])

    return lowest_timestamp


def main():
    args = sys.argv[1:]

    if len(args) > 2:
        print("usage: python3 get_stats.py [output_log_dir] [--plot-only=results.json]")
        exit(1)

    plot_only = False
    results_json = ""
    output_dir = os.path.expanduser("~")
    for arg in args:
        if '--plot-only' in arg:
            plot_only = True
            results_json = arg.split('=')[1]
        else:
            output_dir = arg

    if not plot_only:
        with open("/tmp/dummy_infos.json", "r") as dummy_infos_fp:
            dummy_infos = json.load(dummy_infos_fp)

        print("Cleaning folder in NAS...")

        clean_folder_nas()

        print("Will sync stats to NAS...")
        sync_stats_to_nas(dummy_infos)

        node_results = get_bandwidth_stats()
        with open(f"{output_dir}/bandwidth_results.json", 'w') as results_fp:
            json.dump(node_results, results_fp, indent=4)
    else:
        with open(results_json, 'r') as results_fp:
            node_results = json.load(results_fp)

    lowest_timestamp = find_lowest_timestamp(node_results)

    plt.figure(figsize=(25, 15))

    pcolors = list(colors.TABLEAU_COLORS.keys())
    pline_types = ['-', '--', '-.', ':']
    markers = ['o', '^', '<', '>']

    print(pcolors)

    color_i = 0
    line_i = 0
    marker_i = 0
    for node, results in node_results.items():
        x_axis = []
        y_axis = []

        for result in results:
            x_axis.append(float(result[TIMESTAMP]) - float(lowest_timestamp))
            y_axis.append(float(result[BYTES_TOTAL]) / 1000)

        color_i = color_i % len(pcolors)
        line_i = line_i % len(pline_types)
        marker_i = marker_i % len(markers)

        color = pcolors[color_i]
        line_type = pline_types[line_i]
        marker = markers[marker_i]

        plt.plot(x_axis, y_axis, label=node.split('.csv')[0], color=color, linestyle=line_type, marker=marker)

        color_i += 1

        if color_i == len(pcolors):
            line_i += 1

        if line_i == len(pline_types):
            marker_i += 1

    plt.legend()
    plt.savefig(f'{output_dir}/bandwidths.png')


if __name__ == '__main__':
    main()

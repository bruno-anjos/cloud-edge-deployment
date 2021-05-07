#!/usr/bin/python3
import json
import os
import socket
import subprocess

import matplotlib.pyplot as plt
import pandas
import sys

client_logs_dirname = "client_logs"

TIMESTAMP = "TIMESTAMP"
BITS_OUT = "BYTES_OUT"
BITS_IN = "BYTES_IN"
BITS_TOTAL = "BYTES_TOTAL"


def run_cmd_with_try(cmd):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=subprocess.DEVNULL)
    if cp.stderr is not None:
        raise Exception(cp.stderr)


def exec_cmd_on_node(node, cmd):
    path_var = os.environ["PATH"]
    remote_cmd = f"oarsh {node} -- 'PATH=\"{path_var}\" && {cmd}'"
    run_cmd_with_try(remote_cmd)


def sync_stats_to_nas(dummy_infos, output_dir):
    hostname = socket.gethostname()

    nodes = {}
    for info in dummy_infos:
        node = info["node"]
        nodes[node] = None

    stats_dir = f'{output_dir}/stats/bandwidths'
    os.mkdir(stats_dir)

    cp_to_nas_cmd = f"cp /tmp/bandwidth_stats/* {stats_dir}"
    for node in nodes:
        if node == hostname:
            run_cmd_with_try(cp_to_nas_cmd)
        else:
            exec_cmd_on_node(node, cp_to_nas_cmd)

    print("Synced stats to NAS")


def read_csvs(stats_dir):
    headers = ['timestamp',
               'iface_name',
               'bytes_out/s',
               'bytes_in/s',
               'bytes_total/s',
               'bytes_in',
               'bytes_out',
               'packets_out/s',
               'packets_in/s',
               'packets_total/s',
               'packets_in',
               'packets_out',
               'errors_out/s',
               'errors_in/s',
               'errors_in',
               'errors_out',
               'bits_out/s',
               'bits_in/s',
               'bits_total/s',
               'bits_in',
               'bits_out']
    values_list = []
    for dummy in os.listdir(stats_dir):
        full_file_path = os.path.join(stats_dir, dummy)
        values = pandas.read_csv(full_file_path, delimiter=';', names=headers)
        values['dummy'] = dummy.split('.')[0]
        values_list.append(values)

    return pandas.concat(values_list)


def get_nodes():
    cmd = f'oarprint host'
    return subprocess.getoutput(cmd).strip().split('\n')


def plot_bandwidths(output_dir, dummy_infos):
    values = read_csvs(f'{output_dir}/stats/bandwidths/')
    print(values)

    timestamp_header = 'timestamp'
    dummy_header = 'dummy'
    bits_total_header = 'bits_total/s'
    iface_header = 'iface_name'
    bits_out_s = 'bits_out/s'
    bits_in_s = 'bits_in/s'

    min_timestamp = values[timestamp_header].min()
    to_keep = values[iface_header].str.match(r'eth0')
    values = values[to_keep]
    to_keep = values[bits_total_header] < 100_000_000
    values = values[to_keep]
    dummies = values.groupby(dummy_header)

    bandwidths_dir = f'{output_dir}/plots/bandwidths'
    if not os.path.exists(bandwidths_dir):
        os.mkdir(bandwidths_dir)

    for dummy in dummy_infos:
        fig = plt.figure(figsize=(25, 15))

        name = dummy['name']
        try:
            dummy_stats = dummies.get_group(name)
        except KeyError:
            print(f'Missing group for {name}')
            continue

        ifaces_stats = dummy_stats.groupby(iface_header)

        plt.ylabel("Mb/s")
        for iface_name, stats in ifaces_stats:
            plt.plot(stats[timestamp_header] - min_timestamp,
                     stats[bits_total_header] / 1_000_000, label=f'{iface_name} {bits_total_header}')
            plt.plot(stats[timestamp_header] - min_timestamp,
                     stats[bits_out_s] / 1_000_000, label=f'{iface_name} {bits_out_s}')
            plt.plot(stats[timestamp_header] - min_timestamp,
                     stats[bits_in_s] / 1_000_000, label=f'{iface_name} {bits_in_s}')

        plt.grid()
        plt.legend()
        plt.savefig(f'{bandwidths_dir}/{name}.png')
        plt.close(fig)


def plot_cpu_mem_stats(output_dir, nodes, prefix):
    server_results = {}
    for node in nodes:
        results = pandas.read_csv(
            f'{output_dir}/stats/{node}_cpu_mem.csv', delimiter=';')
        server_results[node] = results

    plt.figure(figsize=(25, 15))
    plt.xlabel('seconds')
    plt.ylabel('CPU (%)')

    min_timestamp = 0
    for node, results in server_results.items():
        timestamp = results['timestamp'].min()
        if timestamp < min_timestamp or min_timestamp == 0:
            min_timestamp = timestamp

    for node, results in server_results.items():
        plt.plot(results['timestamp'].apply(lambda x: x -
                                                      min_timestamp), results['cpu'], label=node)

    prefix_dir = f'{output_dir}/plots/{prefix}'
    if not os.path.exists(prefix_dir):
        os.mkdir(prefix_dir)

    plt.ylim([0, 100])
    plt.legend()
    plt.grid()
    plt.savefig(f'{prefix_dir}/cpu_stats.png')

    plt.figure(figsize=(25, 15))
    plt.xlabel('seconds')
    plt.ylabel('MEMORY (%)')

    for node, results in server_results.items():
        plt.plot(results['timestamp'].apply(lambda x: x -
                                                      min_timestamp), results['mem'], label=node)

    plt.ylim([0, 100])
    plt.legend()
    plt.grid()
    plt.savefig(f'{prefix_dir}/mem_stats.png')


def main():
    args = sys.argv[1:]

    if len(args) > 2:
        print(
            "usage: python3 get_stats.py [output_log_dir] [--plot-only]")
        exit(1)

    plot_only = False
    output_dir = os.path.expanduser("~")
    for arg in args:
        if '--plot-only' in arg:
            plot_only = True
        else:
            output_dir = arg

    with open(f"{output_dir}/dummy_infos.json", "r") as dummy_infos_fp:
        dummy_infos = json.load(dummy_infos_fp)
    if not plot_only:
        print("Cleaning folder in NAS...")

        print("Will sync stats to NAS...")
        sync_stats_to_nas(dummy_infos, output_dir)

    plot_bandwidths(output_dir, dummy_infos)
    plot_cpu_mem_stats(output_dir, get_nodes(), '')


if __name__ == '__main__':
    main()

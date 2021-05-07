import json
import os
import re
import subprocess
import threading
from datetime import datetime

import sys
import time

CLOUD_EDGE_DIR = os.getenv("CLOUD_EDGE_DEPLOYMENT")
SCENARIOS_DIR = os.path.expanduser('~/ced-scenarios/')
CLIENT_SCENARIOS_DIR = os.path.expanduser('~/ced-client-scenarios/')
MAPS_DIR = f'{CLOUD_EDGE_DIR}/latency_maps'
NOVAPOKEMON_DIR = os.getenv('NOVAPOKEMON')
NOVAPOKEMON_SCRIPTS_DIR = f'{NOVAPOKEMON_DIR}/scripts'
CED_SCRIPTS_DIR = f'{CLOUD_EDGE_DIR}/scripts'


def run_with_log_and_exit(cmd):
    print(f"RUNNING | {cmd}")
    ret = subprocess.run(cmd, shell=True)
    if ret.returncode != 0:
        exit(ret.returncode)


def exec_cmd_on_node(node, cmd):
    remote_cmd = f'oarsh {node} "{cmd}"'
    run_with_log_and_exit(remote_cmd)


def launch_visualizer_thread(scenario, time_between_snapshots, target_dir, duration, client_node):
    path = f'{CLOUD_EDGE_DIR}/scripts/visualizer/visualizer_daemon.py'
    cmd = f'python3 {path} {scenario} {time_between_snapshots} --output_dir={target_dir}/stats {duration} {client_node}'

    print("Launching visualizer thread...")

    run_with_log_and_exit(cmd)

    return 0


def save_stats(target_dir):
    path = f"{CLOUD_EDGE_DIR}/scripts/get_stats.py"
    cmd = f'python3 {path} {target_dir}'

    print("Saving stats...")

    run_with_log_and_exit(cmd)


def save_logs(logs_dir, client_node):
    path = f"{CLOUD_EDGE_DIR}/scripts/get_logs.py"
    cmd = f'python3 {path} {logs_dir} {client_node}'

    print("Saving logs...")

    run_with_log_and_exit(cmd)


def deploy_stack(scenario, deploy_opts, server_nodes, build_only, edge_bw_multiplier):
    path = f"{CLOUD_EDGE_DIR}/scripts/deploy_dummy_stack.py"
    cmd = f'python3 {path} {scenario} {deploy_opts} {",".join(server_nodes)} {edge_bw_multiplier} {" --build-only" if build_only else ""}'

    print("Deploying stack...")

    run_with_log_and_exit(cmd)


def deploy_novapokemon(nova_poke_opts):
    path = f"{CLOUD_EDGE_DIR}/scripts/deploy_novapokemon.py"
    cmd = f'python3 {path} {nova_poke_opts}'

    print("Deploying novapokemon...")

    run_with_log_and_exit(cmd)


def record_cpu_mem_stats_node(node, timeout, duration, experiment_dir):
    cmd = f'python3 {CED_SCRIPTS_DIR}/record_stats.py {timeout} {duration} {experiment_dir}'
    exec_cmd_on_node(node, cmd)


def start_recording(duration, timeout, experiment_dir, nodes):
    path = f"{CLOUD_EDGE_DIR}/scripts/start_recording.py"
    cmd = f'python3 {path} {duration} {timeout}'

    print("Starting bandwidth recordings...")

    run_with_log_and_exit(cmd)

    threads = []
    for node in nodes:
        t = threading.Thread(
            target=record_cpu_mem_stats_node,
            daemon=True,
            args=(
                node, timeout, duration, experiment_dir
            )
        )
        t.start()
        threads.append(t)

    return threads


def deploy_clients(scenario, fallback, timeout, clients, client_node, duration):
    path = f"{CLOUD_EDGE_DIR}/scripts/deploy_novapokemon_clients.py"
    cmd = f'python3 {path} {scenario} {fallback} {timeout} {clients} {duration}'

    print("Deploying clients...")

    exec_cmd_on_node(client_node, cmd)


def process_time_string(time_string):
    time_in_seconds = int(time_string[:-1])

    time_suffix = time_string[-1]
    if time_suffix == "m" or time_suffix == "M":
        time_in_seconds = time_in_seconds * 60
    elif time_suffix == "h" or time_suffix == "H":
        time_in_seconds = time_in_seconds * 60 * 60

    return time_in_seconds


def save_dummy_infos(save_dir):
    cmd = ['cp', '/tmp/dummy_infos.json', save_dir]
    subprocess.run(cmd)


def get_lats(deploy_opts):
    splitted_opts = deploy_opts.split(" ")

    found = False
    mapname = ""

    for i, opt in enumerate(splitted_opts):
        if opt == "--mapname":
            mapname = splitted_opts[i + 1]
            found = True

    if not found:
        sys.exit("missing mapname in deploy opts")

    print(f"Getting map {mapname} lats...")

    cmd = f'cp {MAPS_DIR}/{mapname}/lat.txt {NOVAPOKEMON_DIR}/lat.txt'
    run_with_log_and_exit(cmd)

    print("Done!")


def build_novapokemon(scenario, deploy_opts):
    print("Extracting locations...")

    cmd = f'python3 {NOVAPOKEMON_SCRIPTS_DIR}/extract_locations_from_scenario.py {scenario}'
    run_with_log_and_exit(cmd)

    get_lats(deploy_opts)

    print("Done!")

    cmd = f'cd {NOVAPOKEMON_DIR} && bash scripts/build_client.sh && ' \
          'bash scripts/build_service_images.sh'
    run_with_log_and_exit(cmd)


SCENARIO = "scenario"
CLIENTS_SCENARIO = "clients_scenario"
TIME_BETW_SNAP = "time_between_snapshots"
TIME_AFTER_STACK = "time_after_stack"
TIME_AFTER_DEPLOY = "time_after_deploy"
DURATION = "duration"
DEPLOY_OPTS = "deploy_opts"
NOVA_POKE_CMD = "novapokemon_cmd"
EDGE_BW_MULTI = 'edge_bw_multi'


def atoi(text):
    return int(text) if text.isdigit() else text


def natural_keys(text):
    return [atoi(c) for c in re.split(r'(\d+)', text)]


def main():
    args = sys.argv[1:]
    if len(args) > 2:
        sys.exit("usage: python3 run_exeperiment.py <experiment.json> [--build-only]")

    experiment_json = ''
    build_only = False
    for arg in args:
        if arg == '--build-only':
            build_only = True
        elif experiment_json == '':
            experiment_json = arg

    with open(experiment_json, 'r') as experiment_fp:
        experiment = json.load(experiment_fp)

    build_novapokemon(experiment[SCENARIO], experiment[DEPLOY_OPTS])

    date = datetime.now()
    timestamp = date.strftime("%m-%d-%H-%M")
    experiment_dir = os.path.expanduser(f'~/experiment_{timestamp}/')
    os.mkdir(experiment_dir)
    plots_dir = f'{experiment_dir}/plots'
    os.mkdir(plots_dir)
    logs_dir = f'{experiment_dir}/logs'
    os.mkdir(logs_dir)
    stats_dir = f'{experiment_dir}/stats'
    os.mkdir(stats_dir)

    nodes = subprocess.getoutput('oarprint host').strip().split('\n')
    nodes.sort(key=natural_keys)
    server_nodes = nodes[:-1]
    client_nodes = nodes[-1:]

    if len(client_nodes) != 1:
        sys.exit(f'For now we only support 1 client node: {client_nodes}')

    print(f'ServerNodes: {server_nodes}\nClientNodes: {client_nodes}')

    with open(f'{experiment_dir}/info.json', 'w') as f:
        with open(f'{CLIENT_SCENARIOS_DIR}/{experiment[CLIENTS_SCENARIO]}') as f_in:
            cli_scenario = json.load(f_in)
        with open(f'{SCENARIOS_DIR}/{experiment[SCENARIO]}') as f_in:
            scenario = json.load(f_in)
        info = {
            'nodes': {
                'client': client_nodes,
                'server': server_nodes
            },
            'experiment': experiment,
            'client_scenario': cli_scenario,
            'scenario': scenario,
        }
        json.dump(info, f, indent=4)

    edge_bw_multiplier = experiment[EDGE_BW_MULTI]
    deploy_stack(experiment[SCENARIO], experiment[DEPLOY_OPTS],
                 server_nodes, build_only, edge_bw_multiplier)

    if build_only:
        sys.exit('BUILD_ONLY: exiting...')

    after_stack_time = process_time_string(experiment[TIME_AFTER_STACK])
    print(f"Sleeping {after_stack_time} after deploying stack...")
    time.sleep(after_stack_time)

    deploy_novapokemon(experiment[NOVA_POKE_CMD])
    threads = start_recording(
        experiment[DURATION], experiment[TIME_BETW_SNAP], experiment_dir, nodes)

    after_deploy_time = process_time_string(experiment[TIME_AFTER_DEPLOY])
    print(f"Sleeping {after_deploy_time} after deploying novapokemon...")
    time.sleep(after_deploy_time)

    with open(f'{SCENARIOS_DIR}/{experiment[SCENARIO]}', 'r') as scenario_fp:
        stack_scenario = json.load(scenario_fp)
        fallback = stack_scenario['fallback']

    duration = experiment[DURATION]

    time_between_in_secs = process_time_string(experiment[TIME_BETW_SNAP])
    cli_timeout = time_between_in_secs * 1000
    cli_counts = process_time_string(experiment[DURATION]) / time_between_in_secs
    cli_thread = threading.Thread(
        target=deploy_clients,
        daemon=True,
        args=(
            experiment[CLIENTS_SCENARIO],
            fallback,
            cli_timeout,
            cli_counts,
            client_nodes[0],
            duration
        )
    )
    cli_thread.start()

    save_dummy_infos(experiment_dir)

    duration_in_seconds = process_time_string(duration)

    vs = threading.Thread(
        target=launch_visualizer_thread,
        daemon=True,
        args=(
            experiment[SCENARIO],
            experiment[TIME_BETW_SNAP],
            experiment_dir, duration,
            client_nodes[0]
        )
    )
    vs.start()

    time.sleep(duration_in_seconds)

    save_logs(logs_dir, client_nodes[0])

    print('Finished saving logs, will now wait for stats recording threads to finish')

    for t in threads:
        t.join()

    print('Stats recording threads finished')

    save_stats(experiment_dir)

    cli_thread.join()
    vs.join()


if __name__ == '__main__':
    main()

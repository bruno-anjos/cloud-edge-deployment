import json
import os
import subprocess
import sys
import threading
import time
from datetime import datetime

from matplotlib.pyplot import sca

CLOUD_EDGE_DIR = os.getenv("CLOUD_EDGE_DEPLOYMENT")
SCENARIOS_DIR = os.path.expanduser('~/ced-scenarios/')
NOVAPOKEMON_DIR = os.getenv('NOVAPOKEMON')
MAPS_DIR = f'{CLOUD_EDGE_DIR}/latency_maps'
SCRIPTS_DIR = f'{NOVAPOKEMON_DIR}/scripts'


def run_with_log_and_exit(cmd):
    print(f"RUNNING | {cmd}")
    ret = subprocess.run(cmd, shell=True)
    if ret.returncode != 0:
        exit(ret.returncode)


def launch_visualizer_thread(scenario, time_between_snapshots, target_dir, duration):
    path = f'{CLOUD_EDGE_DIR}/scripts/visualizer/visualizer_daemon.py'
    cmd = f'python3 {path} {scenario} {time_between_snapshots} --output_dir={target_dir} {duration}'

    print("Launching visualizer thread...")

    run_with_log_and_exit(cmd)

    return 0


def save_stats(target_dir):
    path = f"{CLOUD_EDGE_DIR}/scripts/get_stats.py"
    cmd = f'python3 {path} {target_dir}'

    print("Saving stats...")

    run_with_log_and_exit(cmd)


def save_logs(logs_dir):
    path = f"{CLOUD_EDGE_DIR}/scripts/get_logs.py"
    cmd = f'python3 {path} {logs_dir}'

    print("Saving logs...")

    run_with_log_and_exit(cmd)


def deploy_stack(scenario, deploy_opts):
    path = f"{CLOUD_EDGE_DIR}/scripts/deploy_dummy_stack.py"
    cmd = f'python3 {path} {scenario} {deploy_opts}'

    print("Deploying stack...")

    run_with_log_and_exit(cmd)


def deploy_novapokemon(nova_poke_opts):
    path = f"{CLOUD_EDGE_DIR}/scripts/deploy_novapokemon.py"
    cmd = f'python3 {path} {nova_poke_opts}'

    print("Deploying novapokemon...")

    run_with_log_and_exit(cmd)


def start_recording(aux_duration, timeout):
    path = f"{CLOUD_EDGE_DIR}/scripts/start_recording.py"
    cmd = f'python3 {path} {aux_duration} {timeout}'

    print("Starting bandwidth recordings...")

    run_with_log_and_exit(cmd)


def deploy_clients(scenario, fallback, timeout, clients):
    path = f"{CLOUD_EDGE_DIR}/scripts/deploy_novapokemon_clients.py"
    cmd = f'python3 {path} {scenario} {fallback} {timeout} {clients}'

    print("Deploying clients...")

    run_with_log_and_exit(cmd)


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
        print("missing mapname in deploy opts")
        exit(1)

    print(f"Getting map {mapname} lats...")

    cmd = f'cp {MAPS_DIR}/{mapname}/lat.txt {NOVAPOKEMON_DIR}/lat.txt'
    run_with_log_and_exit(cmd)

    print("Done!")


def build_novapokemon(scenario, deploy_opts):
    print("Extracting locations...")

    cmd = f'python3 {SCRIPTS_DIR}/extract_locations_from_scenario.py {scenario}'
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


def main():
    args = sys.argv[1:]
    if len(args) != 1:
        print("usage: python3 run_exeperiment.py <experiment.json>")
        exit(1)

    with open(args[0], 'r') as experiment_fp:
        experiment = json.load(experiment_fp)

    build_novapokemon(experiment[SCENARIO], experiment[DEPLOY_OPTS])

    deploy_stack(experiment[SCENARIO], experiment[DEPLOY_OPTS])

    after_stack_time = process_time_string(experiment[TIME_AFTER_STACK])
    print(f"Sleeping {after_stack_time} after deploying stack...")
    time.sleep(after_stack_time)

    deploy_novapokemon(experiment[NOVA_POKE_CMD])
    start_recording(experiment[DURATION], experiment[TIME_BETW_SNAP])

    after_deploy_time = process_time_string(experiment[TIME_AFTER_DEPLOY])
    print(f"Sleeping {after_deploy_time} after deploying novapokemon...")
    time.sleep(after_deploy_time)

    with open(f'{SCENARIOS_DIR}/{experiment[SCENARIO]}', 'r') as scenario_fp:
        stack_scenario = json.load(scenario_fp)
        fallback = stack_scenario['fallback']

    time_between_in_secs = process_time_string(experiment[TIME_BETW_SNAP])
    cli_timeout = time_between_in_secs*1000
    cli_counts = process_time_string(experiment[DURATION])/time_between_in_secs
    cli_thread = threading.Thread(target=deploy_clients, daemon=True, args=(
        experiment[CLIENTS_SCENARIO], fallback, cli_timeout, cli_counts))
    cli_thread.start()

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

    save_dummy_infos(experiment_dir)

    duration = experiment[DURATION]
    duration_in_seconds = process_time_string(duration)

    vs = threading.Thread(
        target=launch_visualizer_thread,
        daemon=True,
        args=(
            experiment[SCENARIO],
            experiment[TIME_BETW_SNAP],
            experiment_dir, duration
        )
    )
    vs.start()

    time.sleep(duration_in_seconds)

    save_stats(experiment_dir)

    save_logs(logs_dir)

    cli_thread.join()
    vs.join()


if __name__ == '__main__':
    main()

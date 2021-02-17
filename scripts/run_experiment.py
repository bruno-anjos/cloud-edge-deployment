import json
import os
import subprocess
import sys
import threading
import time
from datetime import datetime

project_path = os.getenv("CLOUD_EDGE_DEPLOYMENT")
scenarios_path = os.path.expanduser('~/ced-scenarios/')


def run_with_log_and_exit(cmd):
    print(f"RUNNING | {cmd}")
    ret = subprocess.run(cmd)
    if ret.returncode != 0:
        exit(ret.returncode)


def launch_visualizer_thread(scenario, time_between_snapshots, target_dir, duration):
    path = f"{project_path}/scripts/visualizer/visualizer_daemon.py"
    cmd = ["python3", path, scenario, time_between_snapshots, f"--output_dir={target_dir}", duration]

    print("Launching visualizer thread...")

    run_with_log_and_exit(cmd)

    return 0


def save_stats(target_dir):
    path = f"{project_path}/scripts/get_stats.py"
    cmd = ["python3", path, target_dir]

    print("Saving stats...")

    run_with_log_and_exit(cmd)


def save_logs(target_dir):
    path = f"{project_path}/scripts/get_logs.py"
    cmd = ["python3", path, target_dir]

    print("Saving logs...")

    run_with_log_and_exit(cmd)


def deploy_stack(scenario, deploy_opts):
    path = f"{project_path}/scripts/deploy_dummy_stack.py"
    cmd = ["python3", path, scenario]
    cmd.extend(deploy_opts.split(" "))

    print("Deploying stack...")

    run_with_log_and_exit(cmd)


def deploy_novapokemon(nova_poke_opts):
    path = f"{project_path}/scripts/deploy_novapokemon.py"
    cmd = ["python3", path]
    cmd.extend(nova_poke_opts.split(" "))

    print("Deploying novapokemon...")

    run_with_log_and_exit(cmd)


def start_recording(aux_duration, timeout):
    path = f"{project_path}/scripts/start_recording.py"
    cmd = ["python3", path, aux_duration, timeout]

    print("Starting bandwidth recordings...")

    run_with_log_and_exit(cmd)


def deploy_clients(scenario, fallback):
    path = f"{project_path}/scripts/deploy_novapokemon_clients.py"
    cmd = ["python3", path, scenario, fallback]

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

    deploy_stack(experiment[SCENARIO], experiment[DEPLOY_OPTS])

    after_stack_time = process_time_string(experiment[TIME_AFTER_STACK])
    print(f"Sleeping {after_stack_time} after deploying stack...")
    time.sleep(after_stack_time)

    deploy_novapokemon(experiment[NOVA_POKE_CMD])
    start_recording(experiment[DURATION], experiment[TIME_BETW_SNAP])

    after_deploy_time = process_time_string(experiment[TIME_AFTER_DEPLOY])
    print(f"Sleeping {after_deploy_time} after deploying novapokemon...")
    time.sleep(after_deploy_time)

    with open(f'{scenarios_path}/{experiment[SCENARIO]}', 'r') as scenario_fp:
        stack_scenario = json.load(scenario_fp)
        fallback = stack_scenario['fallback']

    cli_thread = threading.Thread(target=deploy_clients, daemon=True, args=(experiment[CLIENTS_SCENARIO], fallback))
    cli_thread.start()

    date = datetime.now()
    timestamp = date.strftime("%m-%d-%H-%M")
    target_path = os.path.expanduser(f'~/experiment_{timestamp}/')
    os.mkdir(target_path)

    save_dummy_infos(target_path)

    duration = experiment[DURATION]
    duration_in_seconds = process_time_string(duration)

    vs = threading.Thread(target=launch_visualizer_thread, daemon=True,
                          args=(experiment[SCENARIO], experiment[TIME_BETW_SNAP], target_path, duration))
    vs.start()

    time.sleep(duration_in_seconds)

    save_stats(target_path)

    save_logs(target_path)

    cli_thread.join()
    vs.join()


if __name__ == '__main__':
    main()

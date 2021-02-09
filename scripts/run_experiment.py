import json
import os
import subprocess
import sys
import threading
import time
from datetime import datetime

project_path = os.getenv("CLOUD_EDGE_DEPLOYMENT")


def run_with_log(cmd):
    print(f"RUNNING | {cmd}")
    subprocess.run(cmd)


def launch_visualizer_thread(scenario, time_between_snapshots):
    path = f"{project_path}/scripts/visualizer/visualizer_daemon.py"
    cmd = ["python3", path, scenario, time_between_snapshots]

    print("Launching visualizer thread...")

    run_with_log(cmd)


def save_stats(target_dir):
    path = f"{project_path}/scripts/get_stats.py {target_dir}"
    cmd = ["python3", path]

    print("Saving stats...")

    run_with_log(cmd)


def save_logs(target_dir):
    path = f"{project_path}/scripts/get_logs.py"
    cmd = ["python3", path, target_dir]

    print("Saving logs...")

    run_with_log(cmd)


def deploy_stack(scenario, deploy_opts):
    path = f"{project_path}/scripts/deploy_dummy_stack.py"
    cmd = ["python3", path, scenario]
    cmd.extend(deploy_opts.split(" "))

    print("Deploying stack...")

    run_with_log(cmd)


def deploy_novapokemon(nova_poke_opts):
    path = f"{project_path}/scripts/deploy_novapokemon.py"
    cmd = ["python3", path]
    cmd.extend(nova_poke_opts.split(" "))

    print("Deploying novapokemon...")

    run_with_log(cmd)


def deploy_clients(scenario):
    path = f"{project_path}/scripts/deploy_novapokemon_clients.py"
    cmd = ["python3", path, scenario]

    print("Deploying clients...")

    run_with_log(cmd)


def process_time_string(time_string):
    time_in_seconds = int(time_string[:-1])

    time_suffix = time_string[-1]
    if time_suffix == "m" or time_suffix == "M":
        time_in_seconds = time_in_seconds * 60
    elif time_suffix == "h" or time_suffix == "H":
        time_in_seconds = time_in_seconds * 60 * 60

    return time_in_seconds


args = sys.argv[1:]
if len(args) != 1:
    print("usage: python3 run_exeperiment.py <experiment.json>")
    exit(1)

with open(args[0], 'r') as experiment_fp:
    experiment = json.load(experiment_fp)

SCENARIO = "scenario"
CLIENTS_SCENARIO = "clients_scenario"
TIME_BETW_SNAP = "time_between_snapshots"
TIME_AFTER_STACK = "time_after_stack"
TIME_AFTER_DEPLOY = "time_after_deploy"
DURATION = "duration"
DEPLOY_OPTS = "deploy_opts"
NOVA_POKE_CMD = "novapokemon_cmd"

deploy_stack(experiment[SCENARIO], experiment[DEPLOY_OPTS])

after_stack_time = process_time_string(experiment[TIME_AFTER_STACK])
print(f"Sleeping {after_stack_time} after deploying stack...")
time.sleep(after_stack_time)

deploy_novapokemon(experiment[NOVA_POKE_CMD])

after_deploy_time = process_time_string(experiment[TIME_AFTER_DEPLOY])
print(f"Sleeping {after_deploy_time} after deploying novapokemon...")
time.sleep(after_deploy_time)

deploy_clients(experiment[CLIENTS_SCENARIO])

date = datetime.now()
timestamp = date.strftime("%m-%d-%H-%M")
target_path = os.path.expanduser(f'~/experiment_{timestamp}')
os.mkdir(target_path)

vs = threading.Thread(target=launch_visualizer_thread, args=(experiment[SCENARIO], experiment[TIME_BETW_SNAP]),
                      daemon=True)
vs.start()

duration = experiment[DURATION]
duration_in_seconds = process_time_string(duration)

time.sleep(duration_in_seconds)

save_stats(target_path)

save_logs(target_path)

vs.join()

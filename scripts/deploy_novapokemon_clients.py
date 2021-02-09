import json
import os
import subprocess
import sys
import time
from multiprocessing import Pool

project_path = os.getenv("CLOUD_EDGE_DEPLOYMENT")
script_path = f"{project_path}/scripts/deploy_novapokemon_clients.sh"

NUM = "number_of_clients"
REGION = "region"
DURATION = "duration"
WAIT_TIME = "wait_time"

NUM_ARGS = 1


def deploy_clients(wait_time, num, region, logs_dir, network, duration):
    time.sleep(wait_time)

    cmd = ["bash", script_path, str(num), region, logs_dir, network, duration]
    subprocess.run(cmd)


def process_time_string(time_string):
    time_in_seconds = int(time_string[:-1])

    time_suffix = time_string[-1]
    if time_suffix == "m" or time_suffix == "M":
        time_in_seconds = time_in_seconds * 60
    elif time_suffix == "h" or time_suffix == "H":
        time_in_seconds = time_in_seconds * 60 * 60

    return time_in_seconds


def main():
    pool = Pool(processes=os.cpu_count())

    args = sys.argv[1:]
    if len(args) != NUM_ARGS:
        print("usage: python3 scripts/deploy_novapokemon_clients.py <clients_scenario.json>")
        exit(1)

    clients_scenarios = f"{os.path.expanduser('~')}/ced-client-scenarios/"
    path = f"{clients_scenarios}/{args[0]}"

    with open(path, 'r') as clients_scenario_fp:
        clients_scenario = json.load(clients_scenario_fp)

        async_waits = []

        for clients_id, clients_config in clients_scenario.items():
            num = clients_config[NUM]
            region = clients_config[REGION]
            duration = clients_config[DURATION]
            wait_time = process_time_string(clients_config[WAIT_TIME])

            print(f"Launching {clients_id}")
            async_waits.append(pool.apply_async(deploy_clients, (
                int(wait_time), num, region, f"/tmp/client_logs/{clients_id}", "swarm-network", duration)))

    for w in async_waits:
        w.get()

    pool.terminate()
    pool.close()

    print("Done!")


if __name__ == '__main__':
    main()

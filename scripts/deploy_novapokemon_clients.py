import json
import os
import subprocess
import sys
import time
from multiprocessing import Pool

project_path = os.getenv("CLOUD_EDGE_DEPLOYMENT")

NUM = "number_of_clients"
REGION = "region"
DURATION = "duration"
WAIT_TIME = "wait_time"

NUM_ARGS = 4


def deploy_clients(wait_time, num, region, logs_dir, network, duration,
                   fallback, client_id, timeout, counts):
    time.sleep(wait_time)

    if os.path.exists(logs_dir):
        clean_clients_logs(logs_dir)
    else:
        os.mkdir(logs_dir)

    env = f'--env NUM_CLIENTS={num} --env REGION={region} --env CLIENTS_TIMEOUT={duration} ' \
          f'--env FALLBACK_URL={fallback} --env CLIENT_ID={client_id} --env TIMEOUT={timeout} ' \
        f' --env MEASURE_COUNTS={counts} --env-file {project_path}/scripts/client-env.list'
    volumes = f'-v {logs_dir}:/logs -v /tmp/services:/services -v /tmp/bandwidth_stats:/bandwidth_stats'
    cmd = f'docker run -d {env} --network "{network}" {volumes} brunoanjos/client:latest'

    res = subprocess.run(cmd, shell=True)
    if res.returncode != 0:
        print(res)
        exit(1)


def process_time_string(time_string):
    time_in_seconds = int(time_string[:-1])

    time_suffix = time_string[-1]
    if time_suffix == "m" or time_suffix == "M":
        time_in_seconds = time_in_seconds * 60
    elif time_suffix == "h" or time_suffix == "H":
        time_in_seconds = time_in_seconds * 60 * 60

    return time_in_seconds


def clean_clients_logs(logs_dir):
    print(f'Cleaning logs dir {logs_dir}')

    cmd = f'docker run -v {logs_dir}:/logs debian:latest sh -c "rm -rf /logs/*"'
    res = subprocess.run(cmd, shell=True)
    if res.returncode != 0:
        print(res)
        exit(1)


def clean_services():
    print('Cleaning services')
    cmd = 'docker run -v /tmp/services:/services debian:latest sh -c "rm -rf /services/*"'
    res = subprocess.run(cmd, shell=True)
    if res.returncode != 0:
        print(res)
        exit(1)


def main():
    pool = Pool(processes=os.cpu_count())

    args = sys.argv[1:]
    if len(args) != NUM_ARGS:
        print("usage: python3 scripts/deploy_novapokemon_clients.py "\
            "<clients_scenario.json> <fallback> <timeout> <counts>")
        exit(1)

    clients_scenarios = f"{os.path.expanduser('~')}/ced-client-scenarios/"
    path = f"{clients_scenarios}/{args[0]}"
    fallback = args[1]
    timeout = args[2]
    counts = args[3]

    clean_services()

    top_dir = '/tmp/client_logs'

    if not os.path.exists(top_dir):
        os.mkdir(top_dir)

    with open(path, 'r') as clients_scenario_fp:
        clients_scenario = json.load(clients_scenario_fp)

        async_waits = []

        for clients_id, clients_config in clients_scenario.items():
            num = clients_config[NUM]
            region = clients_config[REGION]
            duration = clients_config[DURATION]
            wait_time = process_time_string(clients_config[WAIT_TIME])

            print(f"Launching {clients_id}")
            async_waits.append(
                pool.apply_async(
                    deploy_clients,
                    (
                        int(wait_time), num, region, f"{top_dir}/{clients_id}",
                        "swarm-network", duration, fallback, clients_id,
                        timeout, counts
                    )
                )
            )

    for w in async_waits:
        w.get()

    pool.terminate()
    pool.close()

    print("Done!")


if __name__ == '__main__':
    main()

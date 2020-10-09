#!/usr/bin/python3
import json
import multiprocessing
import os
import subprocess
import sys


def build_binary():
    main_path = f"{os.path.dirname(os.path.realpath(__file__))}/../cmd/dummy_client/main.go"
    out_binary_path = f"{os.path.dirname(os.path.realpath(__file__))}/../build/dummy_client/dummy_client"
    subprocess.run(["env", "CGO_ENABLED=0", "GOOS=linux", "go-1.14", "build", "-o", out_binary_path, main_path],
                   check=True, cwd=f"{os.path.dirname(os.path.realpath(__file__))}/../")


def build_and_run_config(config_id, service, client_config, top_output_dir):
    output_dir = os.path.join(top_output_dir, config_id)
    os.mkdir(output_dir)

    original_dockerfile = os.path.join(top_output_dir, "Dockerfile")
    target_dockerfile = os.path.join(output_dir, "Dockerfile")
    subprocess.run(["cp", original_dockerfile, target_dockerfile])

    config = {
        service_key: service,
        req_timeout_key: client_config[req_timeout_key],
        max_reqs_key: client_config[max_reqs_key],
        num_clients_key: client_config[num_clients_key],
        fallback_key: fallback,
        location_key: client_config[location_key]
    }
    with open(f"{output_dir}/config.json", "w") as config_fp:
        json.dump(config, config_fp, indent=4, sort_keys=False)

    cmd = ["docker", "build", "-t", f"brunoanjos/dummy_client:{config_id}", output_dir]
    res = subprocess.run(cmd, capture_output=True, check=True)
    print(f"built {config_id} image")
    return res.returncode


args = sys.argv[1:]
if len(args) != 1:
    print("usage: python3 launch_clients.py launch_config.json")
    exit(1)

launch_config_filename = args[0]

with open(launch_config_filename, "r") as launch_config_fp:
    launch_config = json.load(launch_config_fp)

service_key = "service"
req_timeout_key = "request_timeout"
max_reqs_key = "max_requests"
num_clients_key = "number_of_clients"
fallback_key = "fallback"
location_key = "location"

with open(f"{os.path.dirname(os.path.realpath(__file__))}/../build/deployer/fallback.txt", "r") as fallback_fp:
    fallback = fallback_fp.readline()

    "${BUILD_DIR}/${SERVICE_NAME}/${SERVICE_NAME}" "${CMD_DIR}/${SERVICE_NAME}/main.go"

build_binary()

top_output_dir = f"{os.path.dirname(os.path.realpath(__file__))}/../build/dummy_client/"
for item in os.listdir(top_output_dir):
    item_path = os.path.join(top_output_dir, item)
    if os.path.isdir(item_path):
        print(f"deleting {item}")
        os.rmdir(item_path)

processes = []
pool = multiprocessing.Pool(processes=os.cpu_count())
for service in launch_config:
    print(f"building client config for service {service}")
    for i, client_config in enumerate(launch_config[service]):
        config_id = service + '_' + str(i)
        print(f"handling {config_id}")
        p = pool.apply_async(build_and_run_config, (config_id, service, client_config, top_output_dir))
        processes.append(p)

pool.close()

for p in processes:
    if p.get() != 0:
        print("Failed")
        sys.stdout.flush()

pool.join()

sys.stdout.flush()
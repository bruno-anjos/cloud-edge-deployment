#!/usr/bin/python3
import json
import os
import socket
import subprocess
import sys
import time
from functools import partial
from multiprocessing import Pool

import s2sphere
from netaddr import IPNetwork

project_path = os.path.expanduser("~/go/src/github.com/bruno-anjos/cloud-edge-deployment")
nodes = subprocess.getoutput("oarprint host").strip().split("\n")
host = socket.gethostname().strip()


def run_cmd_with_try(cmd):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=subprocess.DEVNULL)

    failed = False
    if cp.stderr is not None:
        failed = True
        print(f"StdErr: {cp.stderr}")
    if cp.returncode != 0:
        failed = True
        print(f"RetCode: {cp.returncode}")
    if failed:
        exit(1)


def exec_cmd_on_nodes(cmd):
    for node in nodes:
        print(f"Running | {cmd} | {node}")
        if node == host:
            run_cmd_with_try(cmd)
        else:
            exec_cmd_on_node(node, cmd)


def exec_cmd_on_nodes_parallel(cmd):
    cmd_pool = Pool(processes=os.cpu_count())

    other_nodes = [node for node in nodes if node != host]
    cmd_pool.map(partial(exec_cmd_on_node, cmd=cmd), other_nodes)
    run_cmd_with_try(cmd)

    cmd_pool.close()
    cmd_pool.terminate()


def exec_cmd_on_node(node, cmd):
    path_var = os.environ["PATH"]
    remote_cmd = f"oarsh {node} -- 'PATH=\"{path_var}\" && {cmd}'"
    run_cmd_with_try(remote_cmd)


def exec_cmd_on_node_with_output(cmd, node):
    remote_cmd = f"oarsh {node} -- {cmd}"
    (status, out) = subprocess.getstatusoutput(remote_cmd)
    if status != 0:
        print(out)
        exit(1)
    return out


def del_everything():
    remove_cmd = f"bash {project_path}/scripts/delete_everything.sh"
    run_cmd_with_try(remove_cmd)


def del_everything_swarm():
    remove_cmd = f"bash {project_path}/scripts/delete_everything.sh"
    exec_cmd_on_nodes_parallel(remove_cmd)


def create_network():
    print("Creating network...")
    create_network_cmd = f"docker network create --subnet={cidr_provided} {network}"
    run_cmd_with_try(create_network_cmd)


def setup_swarm():
    setup_swarm_cmd = f"bash {project_path}/scripts/setup_swarm.sh {cidr_provided} {network}"
    run_cmd_with_try(setup_swarm_cmd)


def build_dummy_node_image():
    print("Building dummy node image...")
    build_cmd = f"bash {project_path}/build/dummy_node/build_dummy_node.sh"
    if reuse:
        build_cmd += " --skip-final"
    run_cmd_with_try(build_cmd)


NAME = "name"
NODE_IP = "ip"
LOCATION = "location"
NODE = "node"
NUM = "num"


def launch_dummy(info):
    quotas_opts = ""
    if quotas is not None:
        quotas_opts = f"--memory={quotas[MEM_QUOTA_KEY]} --cpus={quotas[CPU_QUOTA_KEY]}"

    env_variables = f'--env NODE_IP="{info[NODE_IP]}" --env NODE_ID="{info[NAME]}" --env NODE_NUM="{info[NUM]}" ' \
                    f'--env LOCATION="{info[LOCATION]}" --env LANDMARKS="{landmarks}" ' \
                    f'--env IPS_FILE="config/banjos_config.txt" --env WAIT_FOR_START="true"'
    volumes = f'-v /tmp/images:/images -v /lib/modules:/lib/modules -v /tmp/bandwidth_stats:/bandwidth_stats ' \
              f'-v /tmp/tables:/tables'
    launch_cmd = f'docker run -d --network={network} --privileged --ip {info[NODE_IP]} ' \
                 f'--name={info[NAME]} --hostname {info[NAME]} {env_variables} {volumes} {quotas_opts} ' \
                 f'--cap-add=ALL brunoanjos/dummy_node:latest'
    if not swarm or info[NODE] == host:
        run_cmd_with_try(launch_cmd)
    else:
        exec_cmd_on_node(info[NODE], launch_cmd)


def start_services_in_dummy(info):
    start_services_cmd = f"docker exec {info[NAME]} ./deploy_services.sh"
    if not swarm or info[NODE] == host:
        run_cmd_with_try(start_services_cmd)
    else:
        exec_cmd_on_node(info[NODE], start_services_cmd)


def build_dummy_infos(num, s2_locs):
    print("Building dummy nodes infos...")

    infos = []
    if len(ips) < num:
        print(f"CIDR only has {len(ips)} IPs but {num} nodes were supposed to be launched...")
        exit(1)

    for i, ip in zip(range(1, num + 1), ips):
        name = f"dummy{i}"
        location = s2_locs[name]

        dummy_info = {
            NAME: name,
            NODE_IP: ip,
            LOCATION: location,
            NUM: i,
        }

        print(f"{name}: {ip} | {location}")
        infos.append(dummy_info)

    return infos


def build_dummy_infos_swarm(num, s2_locs, entrypoints_ips):
    print("Building dummy nodes infos for swarm...")
    print(f"Ignoring IPs {entrypoints_ips}")

    infos = []
    curr_ip = 0
    dummies_deployed = 1
    while dummies_deployed <= num:
        ip = ips[curr_ip]
        valid = False
        while not valid:
            if ip in entrypoints_ips:
                curr_ip += 1
                if curr_ip >= len(ips):
                    print("Not enough IPs in CIDR for swarm")
                    exit(1)
                ip = ips[curr_ip]
            else:
                valid = True

        name = f"dummy{dummies_deployed}"
        location = s2_locs[name]
        node = nodes[(dummies_deployed - 1) % len(nodes)]

        dummy_info = {
            NAME: name,
            NODE_IP: ip,
            LOCATION: location,
            NODE: node,
            NUM: dummies_deployed - 1,
        }

        print(f"{name}: {ip} | {location} | {node}")
        infos.append(dummy_info)

        dummies_deployed += 1
        curr_ip += 1

    return infos


def update_dependencies(build_path):
    print(f"Updating dependencies at {build_path}")
    cmd = f'cd {build_path} && GO111MODULE="on" go mod tidy'
    run_cmd_with_try(cmd)


def load_s2_locations():
    print("Loading s2 locations...")

    locations = scenario["locations"]

    s2_locs = {}
    for dummy_name, location in locations.items():
        s2_locs[dummy_name] = s2sphere.CellId.from_lat_lng(
            s2sphere.LatLng.from_degrees(
                location["lat"],
                location["lng"]
            )
        ).to_token()

    return s2_locs


def setup_anchors():
    entrypoints_ips = set()
    for node in nodes:
        print(f"Setting up anchor at {node}")
        anchor_cmd = f"docker run -d --name=anchor-{node} --network {network} alpine sleep 30m"
        exec_cmd_on_node(node, anchor_cmd)
        get_entrypoint_cmd = f"docker network inspect {network} | grep 'lb-{network}' -A 6"
        """
        Output is like:
        "lb-swarm-network": {
            "Name": "swarm-network-endpoint",
            "EndpointID": "ab543cead9c04275a95df7632165198601de77c183945f2a6ab82ed77a68fdd3",
            "MacAddress": "02:42:c0:a8:a0:03",
            "IPv4Address": "192.168.160.3/20",
            "IPv6Address": ""
        }
        
        so we split at max once thus giving us only the value and not the key
        """
        output = exec_cmd_on_node_with_output(get_entrypoint_cmd, node).strip().split(" ", 1)[1]
        entrypoint_json = json.loads(output)

        entrypoints_ips.add(entrypoint_json["IPv4Address"].split("/")[0])

        get_anchor_cmd = f"docker network inspect {network} | grep 'anchor' -A 5 -B 1"
        output = exec_cmd_on_node_with_output(get_anchor_cmd, node).strip().split(" ", 1)[1]

        if output[-1] == ",":
            output = output[:-1]

        anchor_json = json.loads(output)
        entrypoints_ips.add(anchor_json["IPv4Address"].split("/")[0])
    return entrypoints_ips


def generate_demmon_config(latencies, infos):
    if latencies is None:
        print("missing latencies")
        exit(1)

    latency_filename = f"{project_path}/lats.txt"
    with open(latency_filename, 'w') as f:
        for node_latencies in latencies:
            latencies_string = " ".join([str(latency) for latency in node_latencies])
            f.write(latencies_string)
            f.write("\n")

    subprocess.run(f"cp {latency_filename} {os.environ['DEMMON_DIR']}/config/latency_map.txt".split(" "))

    config_filename = "config/banjos_config.txt"
    ips_filename = "config/banjos_ips_config.txt"

    os.environ["LATENCY_MAP"] = config_filename
    os.environ["IPS_FILE"] = ips_filename

    config_file_path = f'{os.environ["DEMMON_DIR"]}/{config_filename}'
    ips_config_file_path = f'{os.environ["DEMMON_DIR"]}/{ips_filename}'

    with open(config_file_path, 'w') as config_file_fp:
        for info in infos:
            node_config = f"0 {info[NODE_IP]} {info[NAME]}\n"
            config_file_fp.write(node_config)

    with open(ips_config_file_path, 'w') as ips_config_file_fp:
        for info in infos:
            node_config = f"{info[NODE_IP]}\n"
            ips_config_file_fp.write(node_config)


def load_dummy_node_image_swarm():
    load_cmd = f"docker load < {project_path}/build/dummy_node/dummy_node.tar"
    exec_cmd_on_nodes(load_cmd)


def copy_tmp_images_swarm():
    for node in nodes:
        if node == host:
            continue
        print(f"Copying /tmp images to {node}...")
        cmd = f"rsync -r /tmp/images/ {node}:/tmp/images"
        run_cmd_with_try(cmd)


def clean_tables_dir():
    for node in nodes:
        print(f"Clearing tables dir in node {node}")
        exec_cmd_on_node(node, '( [ ! -d /tmp/tables ] || docker run -v /tmp/tables:/tables alpine sh -c "rm -rf '
                               '/tables/dummy*" )')


def setup_tables_dir():
    for node in nodes:
        print(f"Creating tables dir in node {node}")
        exec_cmd_on_node(node, '( [ -d /tmp/tables ] || mkdir /tmp/tables ) ')


def setup_bandwidth_dir():
    for node in nodes:
        print(f"Creating bandwidth stats dir in node {node}")
        exec_cmd_on_node(node, '( [ -d /tmp/bandwidth_stats ] || mkdir /tmp/bandwidth_stats ) ')


def clean_deployment(info):
    clean_cmd = f"docker exec {info[NAME]} ./clean_dummy.sh"
    if not swarm or info[NODE] == host:
        run_cmd_with_try(clean_cmd)
    else:
        exec_cmd_on_node(info[NODE], clean_cmd)


def setup_tc(info):
    clean_cmd = f"docker exec {info[NAME]} ./setup_tc.sh {bandwidth_limit}"
    if not swarm or info[NODE] == host:
        run_cmd_with_try(clean_cmd)
    else:
        exec_cmd_on_node(info[NODE], clean_cmd)


def load_scenario():
    with open(f"{os.path.expanduser('~/ced-scenarios')}/{scenario_filename}") as scenario_fp:
        return json.load(scenario_fp)


args = sys.argv[1:]
if len(args) < 3:
    print("usage: deploy_dummy_stack.py <scenario> <cidr> <bandwith_limit_in_mbits> [--swarm] [--demmon] ["
          "--build-only]")
    exit(1)

swarm = False
demmon = False
build_only = False
reuse = False
scenario_filename = ""
cidr_provided = ""
fallback_id = ""
map_name = ""
bandwidth_limit = ""
quotas_filename = ""

skip = False

for i, arg in enumerate(args):
    if skip:
        skip = False
        continue
    if arg == "--swarm":
        swarm = True
    elif arg == "--demmon":
        demmon = True
    elif arg == "--build-only":
        build_only = True
    elif arg == "--reuse":
        reuse = True
    elif arg == "--fallback":
        fallback_id = args[i + 1]
        skip = True
        print(f"Fallback: {fallback_id}")
    elif arg == "--mapname":
        map_name = args[i + 1]
        skip = True
        print(f"Map name: {map_name}")
    elif arg == "--quotas":
        quotas_filename = args[i + 1]
        skip = True
        print(f"Quotas filename: {quotas_filename}")
    elif scenario_filename == "":
        scenario_filename = arg
    elif cidr_provided == "":
        cidr_provided = arg
    elif bandwidth_limit == "":
        bandwidth_limit = arg
    else:
        print(f"unrecognized option {arg}")
        exit(1)

scenario = load_scenario()
num_nodes = len(scenario["locations"])

if swarm:
    print("Running in swarm mode")

print("Deleting...")

network = ""
landmarks = ""

dummy_infos = {}
if reuse:
    print("Cleaning dummy nodes...")
    with open(f"/tmp/dummy_infos.json", "r") as dummy_infos_fp:
        dummy_infos = json.load(dummy_infos_fp)
        pool = Pool(processes=os.cpu_count())
        pool.map(clean_deployment, dummy_infos)
        pool.close()
        pool.join()
else:
    print("Building from scratch...")
    if swarm:
        network = "swarm-network"
        os.environ["DOCKER_NET"] = network
        del_everything_swarm()
    else:
        network = "dummies-network"
        os.environ["DOCKER_NET"] = network
        del_everything()
    print(f"Setting network as {network}")

ips = [str(ip) for ip in IPNetwork(cidr_provided)]
# Ignore first two IPs since they normally are the NetAddr and the Gateway, and ignore last one since normally it's the
# broadcast IP
ips = ips[2:-1]

s2_locations = load_s2_locations()

print("Setting node configs...")

if not reuse:
    if swarm:
        setup_swarm()
        entrypoints = setup_anchors()
        dummy_infos = build_dummy_infos_swarm(num_nodes, s2_locations, entrypoints)
    else:
        create_network()
        dummy_infos = build_dummy_infos(num_nodes, s2_locations)

    print("Writing dummy infos...")
    with open(f"/tmp/dummy_infos.json", "w") as dummy_infos_fp:
        json.dump(dummy_infos, dummy_infos_fp, indent=4)

    print("Writing node IPs...")
    with open(f"{project_path}/build/deployer/node_ips.json", "w") as deployer_ips_fp, \
            open(f"{project_path}/build/autonomic/node_ips.json", "w") as autonomic_ips_fp:
        node_ips = {}
        for dummy in dummy_infos:
            node_ips[dummy[NAME]] = dummy[NODE_IP]
        json.dump(node_ips, deployer_ips_fp)
        json.dump(node_ips, autonomic_ips_fp)

print("Writing fallback...")
with open(f"{project_path}/build/deployer/fallback.json", "w") as fallback_fp:
    if fallback_id == "":
        fallback_id = scenario["fallback"]
    fallback = {"Id": fallback_id}
    for aux_node in dummy_infos:
        if aux_node[NAME] == fallback_id:
            print(f"Fallback IP: {aux_node[NODE_IP]}")
            fallback["Addr"] = aux_node[NODE_IP]
            break
    landmarks = fallback["Addr"]
    print(f"Wrote fallback: {fallback}")
    json.dump(fallback, fallback_fp)

update_dependencies(project_path)
nm_morais_path = os.path.expanduser("~/go/src/github.com/nm-morais/")
for file in os.listdir(nm_morais_path):
    update_dependencies(nm_morais_path + file)

clean_tables_dir()
setup_tables_dir()
setup_bandwidth_dir()

print("Building images...")

if demmon:
    print("Generating demmon config...")
    generate_demmon_config(scenario["latencies"], dummy_infos)

build_dummy_node_image()

if swarm:
    if not reuse:
        load_dummy_node_image_swarm()
    copy_tmp_images_swarm()

print("Launching...")

if build_only:
    print("Aborting launch due to build only...")
    exit(1)

MEM_QUOTA_KEY = "max_mem"
CPU_QUOTA_KEY = "max_cpu"

quotas = None
if quotas_filename != "" and reuse:
    print(f"[WARNING] Quotas can only be set when launching the stack from scratch. Ignoring quotas...")
elif quotas_filename != "":
    with open(quotas_filename, 'r') as quotas_fp:
        quotas = json.load(quotas_fp)
        if MEM_QUOTA_KEY in quotas:
            print(f"{MEM_QUOTA_KEY} set to {quotas[MEM_QUOTA_KEY]}")
        if CPU_QUOTA_KEY in quotas:
            print(f"{CPU_QUOTA_KEY} set to {quotas[CPU_QUOTA_KEY]}")

pool = Pool(processes=os.cpu_count())
if not reuse:
    start = time.time()
    for info in dummy_infos:
        launch_dummy(info)
    done = time.time()
    print(f"Took {done - start} seconds to launch dummies")

    start = time.time()
    pool.map(setup_tc, dummy_infos)
    done = time.time()
    print(f"Took {done - start} seconds to setup tc")

start = time.time()
pool.map(start_services_in_dummy, dummy_infos)
done = time.time()
print(f"Took {done - start} seconds to start services")

pool.close()
pool.join()

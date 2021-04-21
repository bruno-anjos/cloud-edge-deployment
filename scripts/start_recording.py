import json
import os
import subprocess
import sys
from multiprocessing import Pool


def run_cmd_with_try(cmd):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=subprocess.DEVNULL)
    if cp.stderr is not None:
        raise Exception(cp.stderr)


def exec_cmd_on_node(node, cmd):
    path_var = os.environ["PATH"]
    remote_cmd = f"oarsh {node} -- 'PATH=\"{path_var}\" && {cmd}'"
    run_cmd_with_try(remote_cmd)


def start_recording_in_dummy(info):
    node = info["node"]

    cmd = f"docker exec -t {info['name']} ./start_recording.sh {timeout_in_seconds} {measurement_counts}"

    print(f"Starting recording in {info['name']}")
    exec_cmd_on_node(node, cmd)


def process_time_string(time_string):
    time_in_ms = int(time_string[:-1]) * 1000

    time_suffix = time_string[-1]
    if time_suffix == "m" or time_suffix == "M":
        time_in_ms = time_in_ms * 60
    elif time_suffix == "h" or time_suffix == "H":
        time_in_ms = time_in_ms * 60 * 60

    return time_in_ms


args = sys.argv[1:]
if len(args) > 2:
    print("usage: python3 start_recording.py <duration> [timeout_between_samples]")
    exit(1)

timeout = ""
duration = ""

for arg in args:
    if duration == "":
        duration = arg
    elif timeout == "":
        timeout = arg

with open("/tmp/dummy_infos.json", "r") as dummy_infos_fp:
    dummy_infos = json.load(dummy_infos_fp)

timeout_in_seconds = process_time_string(timeout)
duration_in_seconds = process_time_string(duration)

measurement_counts = duration_in_seconds // timeout_in_seconds

pool = Pool(processes=os.cpu_count())
pool.map(start_recording_in_dummy, dummy_infos)

print("Done!")

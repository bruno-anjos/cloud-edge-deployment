#!/usr/bin/python3
import os
import re
import sys

client_logs_dirname = "client_logs"

timetook_regex = r"took (\d+)"


def get_clients_stats():
    client_logs = f"{logs_dir}/{client_logs_dirname}"
    matches = []

    for client_log in os.listdir(client_logs):
        with open(f"{client_logs}/{client_log}") as log_file:
            matches.extend(re.findall(timetook_regex, log_file.read()))

    avg = sum(matches) / len(matches)
    print(avg)


args = sys.argv[1:]
if len(args) != 1:
    print("usage: python3 get_stats.py logs_dir")
    exit(1)

logs_dir = args[0]

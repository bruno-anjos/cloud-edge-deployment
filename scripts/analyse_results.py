#!/usr/bin/python3

import json
import os
import sys


def explore_extra_path(optimal_deployment, result_deployment, node, extra_nodes):
    to_explore = [node]
    explored = []

    while True:
        if len(to_explore) == 0:
            break

        exploring = to_explore[0]
        to_explore = to_explore[1:]
        explored.append(exploring)

        if exploring not in extra_nodes:
            extra_nodes.append(exploring)

        if exploring not in result_deployment:
            continue

        children = result_deployment[exploring]
        for child in children:
            if child in explored or child in optimal_deployment:
                continue
            to_explore.append(child)


# load expect tree for each deployment
def load_optimal_trees():
    optimal_trees = {}
    with open(f"{os.path.dirname(os.path.realpath(__file__))}/../build/autonomic/metrics/services.tree",
              "r") as tree_fp:
        for line in tree_fp.readlines():
            line_splitted = line.split(":")

            deployment_id = line_splitted[0].strip()
            optimal_trees[deployment_id] = {}

            tree_list = line_splitted[1].split(" -> ")
            previous = ""
            for node in tree_list:
                node = node.strip()
                if previous:
                    optimal_trees[deployment_id][previous] = node
                else:
                    optimal_trees[deployment_id][node] = ""
                previous = node
    return optimal_trees


args = sys.argv[1:]
if len(args) != 1:
    print("usage: python3 analyse_results.py results_file.json")
    exit(1)

results_file = args[0]

with open(results_file, "r") as results_fp:
    results = json.load(results_fp)

with open(f"{os.path.dirname(os.path.realpath(__file__))}/../build/deployer/fallback.txt", "r") as fallback_fp:
    fallback = fallback_fp.readline()

print(f"Starting at: {fallback}")

optimal_trees = load_optimal_trees()

load_balance_factors = {}
extra_nodes = {}
for deployment_id in optimal_trees:
    if deployment_id not in results:
        print(f"{deployment_id} not in results: {results}")
        exit(1)

    optimal_deployment = optimal_trees[deployment_id]
    result_deployment = results[deployment_id]

    node = fallback
    deployment_extra_nodes = []
    load_balance_factors[deployment_id] = 1
    while True:
        if node not in result_deployment:
            break

        real_children = result_deployment[node]

        for real_child in real_children:
            if node not in optimal_deployment or real_child not in optimal_deployment[node]:
                explore_extra_path(optimal_deployment, result_deployment, real_child, deployment_extra_nodes)
            else:
                correct = True

        if node in optimal_deployment:
            next_child = optimal_deployment[node]
            if next_child not in optimal_deployment:
                load_balance_factors[deployment_id] = len(result_deployment[node])
            node = optimal_deployment[node]
        else:
            break

    extra_nodes[deployment_id] = deployment_extra_nodes
    print(f"-------------------------------- {deployment_id} --------------------------------")
    print(f"load_balance_factor: {load_balance_factors[deployment_id]}")
    print(f"extra_nodes: {len(deployment_extra_nodes)} ({deployment_extra_nodes})")

number_deployments = 0
load_balance_sum = 0
extra_nodes_sum = 0

for deployment_id in load_balance_factors:
    load_balance_factor = load_balance_factors[deployment_id]
    num_extra_nodes = extra_nodes[deployment_id]
    load_balance_sum += load_balance_factor
    extra_nodes_sum += len(num_extra_nodes)
    number_deployments += 1

print("-------------------------------- TOTAL STATS --------------------------------")
print(f"avg_load_balance_factor: {load_balance_sum / number_deployments}")
print(f"avg_extra_nodes: {extra_nodes_sum / number_deployments}")

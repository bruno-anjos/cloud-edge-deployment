#!/usr/bin/python3

import sys

args = sys.argv[1:]

if len(args) != 2:
    print("Usage: fix_latencies_file.py <input_filename> <output_filename>")
    exit(1)

input_filename = args[0]
output_filename = args[1]

print(f"Input file: {input_filename}")
print(f"Output file: {output_filename}")

with open(input_filename, 'r') as input_fp, open(output_filename, 'w') as output_fp:
    lines = input_fp.readlines()
    new_lines = []
    for line in lines:
        splitted = line.strip().split(" ")
        new_line = [str(round(float(value), 2)) for value in splitted]
        new_lines.append(" ".join(new_line))
    output_fp.writelines("\n".join(new_lines))

print("Finished")

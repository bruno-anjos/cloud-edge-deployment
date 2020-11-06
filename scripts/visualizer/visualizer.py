import os
import subprocess
import time

deployer_png_filename = "/home/b.anjos/deployer_pngs/deployer_plot.png"

archimedes_tex_filename = "/home/b.anjos/archimedes_tables.tex"
archimedes_tex_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.tex"
archimedes_pdf_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.pdf"
archimedes_out_local_path = "/Users/banjos/Desktop/archimedes_tables/"
archimedes_png_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.png"

results_tree_filename = "dicluster:/home/b.anjos/results/results.json"
results_tree_local_path = "/Users/banjos/Desktop/deployer_pngs/results.json"

wait = 5


def get_scp_file(origin, target):
    print("copying deployer image via scp")
    try:
        subprocess.run(["scp", "dicluster:%s" % origin, target], check=True)
    except subprocess.CalledProcessError as eAux:
        print(f"scp returned {eAux['returncode']}")
        return False
    return True


def rsync_folder_from_server(target, origin):
    print(f"rsyncing {target} to {origin}")
    try:
        subprocess.run(["rsync", "-av", target, origin, "--delete"], check=True)
    except subprocess.CalledProcessError as eAux:
        print("rsync error")
        return False
    return True


mypath = "/Users/banjos/Desktop/deployer_pngs/"
onlyfiles = [os.path.join(mypath, f) for f in os.listdir(mypath) if os.path.isfile(os.path.join(mypath, f))]
print(f"deleting {onlyfiles}")
for file in onlyfiles:
    os.remove(file)

while True:
    deployer_failed = False
    archimedes_failed = False
    if not rsync_folder_from_server("dicluster:/home/b.anjos/deployer_pngs", "/Users/banjos/Desktop/"):
        deployer_failed = True

    rsync_folder_from_server(results_tree_filename,results_tree_local_path)

    # if get_scp_file(archimedes_tex_filename, archimedes_tex_local_path):
    #     try:
    #         subprocess.run(["pdflatex", "-interaction=nonstopmode", "-output-directory", archimedes_out_local_path,
    #                         archimedes_tex_local_path], check=True)
    #     except subprocess.CalledProcessError as e:
    #         print("error running pdflatex: ", e.returncode)
    #
    #     try:
    #         subprocess.run(["convert", "-density", "300", archimedes_pdf_local_path,
    #                         "-quality", "90", "-strip", archimedes_png_local_path], check=True)
    #     except subprocess.CalledProcessError as e:
    #         print("error running convert: ", e.returncode)
    # else:
    #     archimedes_failed = True

    if deployer_failed and archimedes_failed:
        time.sleep(wait)

    time.sleep(20)

import subprocess
import time

import cv2

deployer_png_filename = "/home/b.anjos/deployer_plot.png"
deployer_png_local_path = "/Users/banjos/Desktop/deployer_plot.png"

archimedes_tex_filename = "/home/b.anjos/archimedes_tables.tex"
archimedes_tex_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.tex"
archimedes_pdf_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.pdf"
archimedes_out_local_path = "/Users/banjos/Desktop/archimedes_tables/"
archimedes_png_local_path = "/Users/banjos/Desktop/archimedes_tables/archimedes_tables.png"
wait = 5


def get_scp_file(origin, target):
    print("copying deployer image via scp")
    try:
        subprocess.run(["scp", "dicluster:%s" % origin, target], check=True)
    except subprocess.CalledProcessError as eAux:
        print("scp returned %d, will wait %d seconds and try again" % (eAux["returncode"], wait))
        return False
    return True


while True:
    deployer_failed = False
    archimedes_failed = False
    if get_scp_file(deployer_png_filename, deployer_png_local_path):
        image = cv2.imread(deployer_png_local_path)
        scale_percent = 100
        width = int(image.shape[1] * scale_percent / 100)
        height = int(image.shape[0] * scale_percent / 100)
        dimD = (width, height)
        resized = cv2.resize(image, dimD, interpolation=cv2.INTER_AREA)
        cv2.imshow('deployer_graph', resized)
    else:
        deployer_failed = True

    if get_scp_file(archimedes_tex_filename, archimedes_tex_local_path):
        try:
            subprocess.run(["pdflatex", "-output-directory", archimedes_out_local_path, archimedes_tex_local_path],
                           check=True)
        except subprocess.CalledProcessError as e:
            print("error running pdflatex: ", e.returncode)

        try:
            subprocess.run(["convert", "-density", "300", archimedes_pdf_local_path,
                            "-quality", "90", "-strip", archimedes_png_local_path], check=True)
        except subprocess.CalledProcessError as e:
            print("error running convert: ", e.returncode)
        image = cv2.imread(archimedes_png_local_path)
        scale_percent = 30
        width = int(image.shape[1] * scale_percent / 100)
        height = int(image.shape[0] * scale_percent / 100)
        dimA = (width, height)
        resized = cv2.resize(image, dimA, interpolation=cv2.INTER_AREA)
        cv2.imshow('archimedes_tables', resized)
    else:
        archimedes_failed = True

    if deployer_failed and archimedes_failed:
        time.sleep(wait)

    cv2.waitKey(0)

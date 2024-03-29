package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	deployerClient = deployer.NewDeployerClient(utils.LocalhostAddr + ":" + strconv.Itoa(deployer.Port))
)

func main() {
	debug := flag.Bool("d", false, "add debug logs")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "add a new deployment",
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 2 {
						log.Fatal("add: deployment_name yaml_file")
					}

					addDeployment(c.Args().First(), c.Args().Get(1), false)

					return nil
				},
				Subcommands: []*cli.Command{
					{
						Name:  "static",
						Usage: "add a new static deployment",
						Action: func(c *cli.Context) error {
							if c.Args().Len() != 2 {
								log.Fatal("add static: deployment_name yaml_file")
							}

							addDeployment(c.Args().First(), c.Args().Get(1), true)

							return nil
						},
					},
					{
						Name:  "node",
						Usage: "add a new node",
						Action: func(c *cli.Context) error {
							if c.Args().Len() != 1 {
								log.Fatal("add node: node_addr")
							}

							addNode(c.Args().First())

							return nil
						},
					},
				},
			},
			{
				Name:    "del",
				Aliases: []string{"d"},
				Usage:   "delete a deployment",
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 1 {
						log.Fatal("del: deployment_name")
					}

					deleteDeployment(c.Args().First())

					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func addNode(addr string) {
	status := deployerClient.AddNode(addr)
	if status != http.StatusOK {
		log.Fatalf("got status code %d while adding node to deployer", status)
	}
}

func addDeployment(serviceId, filename string, static bool) {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("error reading file: ", err)
	}

	status := deployerClient.RegisterService(serviceId, static, fileBytes, nil, nil)
	if status != http.StatusOK {
		log.Fatalf("got status %d from deployer", status)
	}
}

func deleteDeployment(serviceId string) {
	status := deployerClient.DeleteService(serviceId)
	if status != http.StatusOK {
		log.Fatalf("got status %d from deployer", status)
	}
}

package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"

	deployer2 "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

var deployerClient = client.NewDeployerClient(servers.DeployerLocalHostPort)

func main() {
	debug := flag.Bool("d", false, "add debug logs")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	//nolint:exhaustivestruct
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "add a new deployment",
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 2 { //nolint:gomnd
						log.Panic("add: deployment_name yaml_file")
					}

					addDeployment(c.Args().First(), c.Args().Get(1))

					return nil
				},
			},
			{
				Name:    "del",
				Aliases: []string{"d"},
				Usage:   "delete a deployment",
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 1 {
						log.Panic("del: deployment_name")
					}

					deleteDeployment(c.Args().First())

					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Panic(err)
	}
}

func addDeployment(deploymentID, filename string) {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Panic("error reading file: ", err)
	}

	var deploymentYAML deployer2.DeploymentYAML

	err = yaml.Unmarshal(fileBytes, &deploymentYAML)
	if err != nil {
		panic(err)
	}

	status := deployerClient.RegisterDeployment(deploymentID, deploymentYAML.Static, fileBytes, nil, nil, nil,
		deployer2.NotExploringTTL)
	if status != http.StatusOK {
		log.Panicf("got status %d from deployer", status)
	}
}

func deleteDeployment(deploymentID string) {
	status := deployerClient.DeleteDeployment(deploymentID)
	if status != http.StatusOK {
		log.Panicf("got status %d from deployer", status)
	}
}

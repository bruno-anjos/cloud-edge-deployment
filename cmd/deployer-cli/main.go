package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"

	deployer2 "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

var (
	deployerClient = deployer.NewDeployerClient(deployer.LocalHostPort)
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

func addDeployment(deploymentId, filename string) {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("error reading file: ", err)
	}

	var deploymentYAML deployer2.DeploymentYAML
	err = yaml.Unmarshal(fileBytes, &deploymentYAML)
	if err != nil {
		panic(err)
	}

	status := deployerClient.RegisterDeployment(deploymentId, deploymentYAML.Static, fileBytes, nil, nil, nil,
		deployer2.NotExploringTTL)
	if status != http.StatusOK {
		log.Fatalf("got status %d from deployer", status)
	}
}

func deleteDeployment(deploymentId string) {
	status := deployerClient.DeleteDeployment(deploymentId)
	if status != http.StatusOK {
		log.Fatalf("got status %d from deployer", status)
	}
}

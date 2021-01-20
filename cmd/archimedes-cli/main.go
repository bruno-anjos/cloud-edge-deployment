package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes/client"

	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

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
				Name:    "resolve",
				Aliases: []string{"r"},
				Usage:   "resolve a specific host",
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 2 { //nolint:gomnd
						log.Panic("resolve: archimedes_addr to_resolve")
					}

					resolveHost(c.Args().First(), c.Args().Get(1))

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

func resolveHost(resolveIn, toResolve string) {
	archClient := client.NewArchimedesClient(resolveIn)

	host, port, err := net.SplitHostPort(toResolve)
	if err != nil {
		log.Panic(err)
	}

	deploymentID := strings.Split(host, "-")[0]

	natPort, err := nat.NewPort("tcp", port)
	if err != nil {
		log.Panic(err)
	}

	rHost, rPort, status, timedOut := archClient.Resolve(host, natPort, deploymentID, s2.CellIDFromToken("2f"),
		uuid.New().String())

	if timedOut {
		log.Error("timed out...")
		return
	}

	if status != http.StatusOK {
		log.Errorf("got status %d", status)
		return
	}

	log.Infof("resolved to %s:%s", rHost, rPort)
}

package clientfactory

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client"
)

type ClientFactory struct{}

func (cf *ClientFactory) New(addr string) deployer.Client {
	return client.NewDeployerClient(addr)
}

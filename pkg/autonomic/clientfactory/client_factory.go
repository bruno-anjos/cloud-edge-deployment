package clientfactory

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic/client"
)

type ClientFactory struct{}

func (cf *ClientFactory) New() autonomic.Client {
	return client.NewAutonomicClient()
}

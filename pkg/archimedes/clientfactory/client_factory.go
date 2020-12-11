package clientfactory

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes/client"
)

type ClientFactory struct{}

func (cf *ClientFactory) New(addr string) archimedes.Client {
	return client.NewArchimedesClient(addr)
}

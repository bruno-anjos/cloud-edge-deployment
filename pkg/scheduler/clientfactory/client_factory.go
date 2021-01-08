package clientfactory

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler/client"
)

type ClientFactory struct{}

func (cf *ClientFactory) New() scheduler.Client {
	return client.NewSchedulerClient()
}

package client_factory

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler/client"
)

type ClientFactory struct{}

func (cf *ClientFactory) New(addr string) scheduler.Client {
	return client.NewSchedulerClient(addr)
}

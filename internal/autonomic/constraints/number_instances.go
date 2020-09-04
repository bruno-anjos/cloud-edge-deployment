package constraints

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

const (
	CONSTRAINT_NUMBER_OF_INSTANCES_ID = "CONSTRAINT_NUMBER_OF_INSTANCES"
)

type NumberOfInstances struct {
	ServiceId            string
	MaxNumberOfInstances int
}

func NewConstraintNumberOfInstances(serviceId string, maxNumInstances int) *NumberOfInstances {
	return &NumberOfInstances{
		MaxNumberOfInstances: maxNumInstances,
	}
}

func (n *NumberOfInstances) GetConstraintId() string {
	return CONSTRAINT_NUMBER_OF_INSTANCES_ID
}

func (n *NumberOfInstances) MetricId() string {
	return autonomic.METRIC_NUMBER_OF_INSTANCES_ID
}

func (n *NumberOfInstances) Validate(value interface{}) bool {
	metric := value.(int)
	if metric > n.MaxNumberOfInstances {
		return false
	}

	return true
}

func (n *NumberOfInstances) GenerateAction() actions.Action {
	// TODO take care of this

	// return actions.NewRemoveServiceAction(n.ServiceId, )
	return nil
}

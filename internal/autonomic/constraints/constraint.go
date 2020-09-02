package constraints

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type Constraint interface {
	GetConstraintId() string
	MetricId() string
	Validate(value interface{}) bool
	GenerateAction() actions.Action
}

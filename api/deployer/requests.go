package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
)

const (
	NotExploringTTL = -1
)

type (
	RegisterDeploymentRequestBody struct {
		DeploymentConfig *DeploymentDTO
		ExploringTTL     int
	}
	AddNodeRequestBody   = string
	DeadChildRequestBody struct {
		Grandchild   *utils.Node
		Alternatives map[string]*utils.Node
		Locations    []s2.CellID
	}
	IAmYourParentRequestBody struct {
		Parent      *utils.Node
		Grandparent *utils.Node
	}
	RegisterDeploymentInstanceRequestBody = archimedes.InstanceDTO
	AlternativesRequestBody               = []*utils.Node
	SetGrandparentRequestBody             = utils.Node
	FallbackRequestBody                   struct {
		Orphan         *utils.Node
		OrphanLocation s2.CellID
	}
	ExtendDeploymentConfig struct {
		Children  []*utils.Node
		Locations []s2.CellID
		ToExclude map[string]interface{}
	}
	ExtendDeploymentRequestBody struct {
		Node         *utils.Node
		ExploringTTL int
		Config       *ExtendDeploymentConfig
	}

	PropagateOpType                       string
	PropagateLocationToHorizonRequestBody struct {
		Operation PropagateOpType
		TTL       int8
		ChildId   string
		Location  s2.CellID
	}
)

const (
	Add    PropagateOpType = "ADD"
	Remove PropagateOpType = "REMOVE"
)

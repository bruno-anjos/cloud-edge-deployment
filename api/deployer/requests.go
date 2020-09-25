package deployer

import (
	archimedes "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	ParentIdx      = 0
	GrandparentIdx = 1
)

type (
	ExpandTreeRequestBody      = string
	RegisterServiceRequestBody = DeploymentDTO
	AddNodeRequestBody         = string
	DeadChildRequestBody       = struct {
		Grandchild   *utils.Node
		Alternatives map[string]*utils.Node
		Location     float64
	}
	TakeChildRequestBody               = utils.Node
	IAmYourParentRequestBody           = []*utils.Node
	RegisterServiceInstanceRequestBody = archimedes.InstanceDTO
	AlternativesRequestBody            = []*utils.Node
	SetGrandparentRequestBody          = utils.Node
	FallbackRequestBody                = struct {
		OrphanId       string
		OrphanLocation float64
	}
	ResolveInArchimedesRequestBody   = archimedes.ToResolveDTO
	StartResolveUpTheTreeRequestBody = archimedes.ToResolveDTO
	ResolveUpTheTreeRequestBody = struct {
		Origin    string
		ToResolve *archimedes.ToResolveDTO
	}
)

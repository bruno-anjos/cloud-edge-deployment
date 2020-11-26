package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
)

type (
	RegisterDeploymentRequestBody struct {
		Deployment *DeploymentDTO
		Host       *utils.Node
	}
	RegisterDeploymentInstanceRequestBody = InstanceDTO
	ResolveRequestBody                    struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
		Location     s2.CellID
		Id           string
		Redirects    []string
	}
	ResolveLocallyRequestBody      = ToResolveDTO
	RedirectRequestBody            = redirectDTO
	SetResolutionAnswerRequestBody struct {
		Resolved *ResolvedDTO
		Id       string
	}
	SetExploringClientLocationRequestBody = []s2.CellID
	AddDeploymentNodeRequestBody          struct {
		NodeId    string
		Location  s2.CellID
		Exploring bool
	}
)

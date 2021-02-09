package deployment

import (
	"strconv"
	"sync"
	"time"

	autonomicAPI "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/golang/geo/s2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	blacklistDuration = 5 * time.Minute
)

type (
	nodeWithLocation struct {
		Node     *utils.Node
		Location s2.CellID
	}

	exploringMapValue = int

	Deployment struct {
		DeploymentID string
		Strategy     strategy
		Children     *sync.Map
		Parent       *utils.Node
		Suspected    *sync.Map
		Environment  *environment.Environment
		Blacklist    *sync.Map
		Exploring    *sync.Map
		DepthFactor  float64
		autoFactory  autonomic.ClientFactory
		archFactory  archimedes.ClientFactory
		deplFactory  deployer.ClientFactory
		schedFactory scheduler.ClientFactory
	}
)

var Myself = utils.NodeFromEnv()

func New(deploymentID, strategyID string, suspected *sync.Map, depthFactor float64,
	env *environment.Environment, autoFactory autonomic.ClientFactory, archFactory archimedes.ClientFactory,
	deplFactory deployer.ClientFactory, schedFactory scheduler.ClientFactory) (*Deployment, error) {
	s := &Deployment{
		DeploymentID: deploymentID,
		Strategy:     nil,
		Children:     &sync.Map{},
		Parent:       nil,
		Suspected:    suspected,
		Environment:  env,
		Blacklist:    &sync.Map{},
		Exploring:    &sync.Map{},
		DepthFactor:  depthFactor,
		autoFactory:  autoFactory,
		archFactory:  archFactory,
		deplFactory:  deplFactory,
		schedFactory: schedFactory,
	}

	var strat strategy

	switch strategyID {
	case autonomic.StrategyLoadBalanceID:
		strat = newDefaultLoadBalanceStrategy(s)
	case autonomic.StrategyIdealLatencyID:
		strat = newDefaultIdealLatencyStrategy(s)
	default:
		return nil, errors.Errorf("invalid strategy: %s", strategyID)
	}

	s.Strategy = strat

	return s, nil
}

func (a *Deployment) AddChild(child *utils.Node, location s2.CellID) {
	node := &nodeWithLocation{
		Node:     child,
		Location: location,
	}
	a.Children.Store(child.ID, node)
}

func (a *Deployment) RemoveChild(childID string) {
	log.Debugf("removing child %s", childID)
	a.Children.Delete(childID)

	_, ok := a.Exploring.Load(childID)
	if ok {
		a.BlacklistNodes(Myself.ID, []string{childID}, map[string]struct{}{Myself.ID: {}})
	}

	log.Debugf("removed child %s", childID)
}

func (a *Deployment) AddSuspectedChild(childID string) {
	a.Suspected.Store(childID, nil)
}

func (a *Deployment) SetParent(parent *utils.Node) {
	a.Parent = parent
}

func (a *Deployment) GenerateAction() actions.Action {
	return a.Strategy.Optimize()
}

func (a *Deployment) ToDTO() *autonomicAPI.DeploymentDTO {
	var children []string

	a.Children.Range(func(key, value interface{}) bool {
		childID := key.(string)
		children = append(children, childID)

		return true
	})

	return &autonomicAPI.DeploymentDTO{
		DeploymentID: a.DeploymentID,
		StrategyID:   a.Strategy.GetID(),
		Children:     children,
		Parent:       a.Parent,
	}
}

func (a *Deployment) GetLoad() int {
	return environment.GetLoad(a.Environment.DemmonCli, a.DeploymentID, Myself)
}

func (a *Deployment) GetChildrenAsArray() (children []*utils.Node) {
	a.Children.Range(func(key, value interface{}) bool {
		childWithLoc := value.(*nodeWithLocation)

		children = append(children, childWithLoc.Node)

		return true
	})

	return
}

func (a *Deployment) BlacklistNodes(origin string, nodes []string, nodesVisited map[string]struct{}) {
	log.Debugf("blacklisting %+v", nodes)

	for _, node := range nodes {
		a.Blacklist.Store(node, nil)
	}

	nodesVisited[Myself.ID] = struct{}{}

	autoClient := a.autoFactory.New()
	if a.Parent != nil {
		_, hasVisitedParent := nodesVisited[a.Parent.ID]
		if !hasVisitedParent {
			addr := a.Parent.Addr + ":" + strconv.Itoa(autonomic.Port)
			if origin != a.Parent.ID {
				log.Debugf("telling parent %s to blacklist %+v for deployment %s", a.Parent.ID, nodes,
					a.DeploymentID)
				autoClient.BlacklistNodes(addr, a.DeploymentID, Myself.ID, nodes, nodesVisited)
			}
		}
	}

	log.Debugf("telling children to blacklist %+v for deployment %s", nodes, a.DeploymentID)
	a.Children.Range(func(key, value interface{}) bool {
		childID := key.(string)
		if childID == origin {
			return true
		}

		if _, ok := nodesVisited[childID]; ok {
			return true
		}

		log.Debugf("telling %s to blacklist %+v for %s", childID, nodes, a.DeploymentID)
		nodeWithLoc := value.(*nodeWithLocation)
		addr := nodeWithLoc.Node.Addr + ":" + strconv.Itoa(autonomic.Port)

		go autoClient.BlacklistNodes(addr, a.DeploymentID, Myself.ID, nodes, nodesVisited)

		return true
	})

	go func() {
		blacklistTimer := time.NewTimer(blacklistDuration)
		<-blacklistTimer.C

		log.Debugf("removing %+v from blacklist", nodes)

		for _, node := range nodes {
			a.Blacklist.Delete(node)
		}
	}()
}

func (a *Deployment) SetExploreSuccess(childID string) bool {
	a.Exploring.Delete(childID)
	log.Debugf("explored %s successfully", childID)

	return true
}

func (a *Deployment) setNodeAsExploring(nodeID string) {
	log.Debugf("exploring child %s", nodeID)
	a.Exploring.Store(nodeID, nil)
}

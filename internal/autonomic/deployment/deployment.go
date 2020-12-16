package deployment

import (
	"strconv"
	"sync"
	"time"

	autonomicAPI "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/golang/geo/s2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	blacklistDuration = 30 * time.Minute
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

	dependencies := strat.GetDependencies()

	for _, deploymentMetric := range dependencies {
		env.TrackMetric(deploymentMetric)
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
	a.Children.Delete(childID)

	_, ok := a.Exploring.Load(childID)
	if ok {
		a.BlacklistNodes(Myself.ID, childID)
	}
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

func (a *Deployment) GetLoad() float64 {
	metric := metrics.GetLoadPerDeployment(a.DeploymentID)

	value, ok := a.Environment.GetMetric(metric)
	if !ok {
		log.Debugf("no value for metric %s", metric)

		return 0
	}

	return value.(float64)
}

func (a *Deployment) BlacklistNodes(origin string, nodes ...string) {
	log.Debugf("blacklisting %+v", nodes)

	for _, node := range nodes {
		a.Blacklist.Store(node, nil)
	}

	autoClient := a.autoFactory.New("")

	if a.Parent != nil {
		autoClient.SetHostPort(a.Parent.Addr + ":" + strconv.Itoa(autonomic.Port))

		if a.Parent != nil && origin != a.Parent.ID {
			autoClient.BlacklistNodes(a.DeploymentID, Myself.ID, nodes...)
		}
	}

	a.Children.Range(func(key, value interface{}) bool {
		childID := key.(string)
		if childID == origin {
			return true
		}
		log.Debugf("telling %s to blacklist %+v for %s", childID, nodes, a.DeploymentID)
		nodeWithLoc := value.(*nodeWithLocation)
		autoClient.SetHostPort(nodeWithLoc.Node.Addr + ":" + strconv.Itoa(autonomic.Port))
		autoClient.BlacklistNodes(a.DeploymentID, Myself.ID, nodes...)

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

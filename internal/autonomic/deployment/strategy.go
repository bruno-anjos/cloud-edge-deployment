package deployment

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type strategy interface {
	Optimize() actions.Action
	GetDependencies() (metricIds []string)
	GetID() string
}

type basicStrategy struct {
	id    string
	goals []goal
}

func newBasicStrategy(id string, goals []goal) *basicStrategy {
	return &basicStrategy{
		id:    id,
		goals: goals,
	}
}

func (b *basicStrategy) Optimize() actions.Action {
	var (
		nextDomain             domain
		goalToChooseActionFrom goal
		goalActionArgs         []interface{}
	)

	for _, g := range b.goals {
		isAlreadyMax, optRange, actionArgs := g.Optimize(nextDomain)
		if isAlreadyMax {
			nextDomain = optRange
		} else {
			goalToChooseActionFrom = g
			goalActionArgs = actionArgs
		}
	}

	if goalToChooseActionFrom == nil || nextDomain == nil {
		return nil
	}

	return goalToChooseActionFrom.GenerateAction(nextDomain, goalActionArgs)
}

func (b *basicStrategy) GetDependencies() (metricIds []string) {
	for _, g := range b.goals {
		metricIds = append(metricIds, g.GetDependencies()...)
	}

	return
}

func (b *basicStrategy) GetID() string {
	return b.id
}

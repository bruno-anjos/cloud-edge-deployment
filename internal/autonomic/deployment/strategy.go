package deployment

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type strategy interface {
	Optimize() actions.Action
	GetDependencies() (metricIds []string)
	GetId() string
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

	for _, goal := range b.goals {
		isAlreadyMax, optRange, actionArgs := goal.Optimize(nextDomain)
		if isAlreadyMax {
			nextDomain = optRange
		} else {
			goalToChooseActionFrom = goal
			goalActionArgs = actionArgs
		}
	}

	if goalToChooseActionFrom == nil || nextDomain == nil {
		return nil
	}

	return goalToChooseActionFrom.GenerateAction(nextDomain, goalActionArgs)
}

func (b *basicStrategy) GetDependencies() (metricIds []string) {
	for _, goal := range b.goals {
		metricIds = append(metricIds, goal.GetDependencies()...)
	}

	return
}

func (b *basicStrategy) GetId() string {
	return b.id
}

package strategies

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
)

type Strategy interface {
	Optimize() actions.Action
	GetDependencies() (metricIds []string)
	GetId() string
}

type basicStrategy struct {
	id    string
	goals []goals.Goal
}

func newBasicStrategy(id string, goals []goals.Goal) *basicStrategy {
	return &basicStrategy{
		id:    id,
		goals: goals,
	}
}

func (b *basicStrategy) Optimize() actions.Action {
	var (
		nextDomain             goals.Domain
		goalToChooseActionFrom goals.Goal
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

	return goalToChooseActionFrom.GenerateAction(nextDomain[0], goalActionArgs)
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

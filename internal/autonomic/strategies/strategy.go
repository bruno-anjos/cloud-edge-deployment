package strategies

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
)

type Strategy interface {
	Optimize() actions.Action
	GetDependencies() (metricIds []string)
}

type BasicStrategy struct {
	goals []goals.Goal
}

func NewBasicStrategy(goals []goals.Goal) *BasicStrategy {
	return &BasicStrategy{
		goals: goals,
	}
}

func (b *BasicStrategy) Optimize() actions.Action {
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

func (b *BasicStrategy) GetDependencies() (metricIds []string) {
	for _, goal := range b.goals {
		metricIds = append(metricIds, goal.GetDependencies()...)
	}

	return
}

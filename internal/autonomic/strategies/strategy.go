package strategies

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
)

type Strategy struct {
	actionsStack []actions.Action
	goals        []goals.Goal
}

func NewStrategy(goals []goals.Goal) *Strategy {
	return &Strategy{
		goals: goals,
	}
}

func (b *Strategy) Optimize() actions.Action {
	var (
		nextDomain             goals.Domain
		goalToChooseActionFrom goals.Goal
	)

	for _, goal := range b.goals {
		isAlreadyMax, optRange := goal.Optimize(nextDomain)
		if isAlreadyMax {
			nextDomain = optRange
		} else {
			goalToChooseActionFrom = goal
		}
	}

	if goalToChooseActionFrom == nil || nextDomain == nil {
		return nil
	}

	return goalToChooseActionFrom.GenerateAction(nextDomain[0])
}

func (b *Strategy) GetDependencies() (metricIds []string) {
	for _, goal := range b.goals {
		metricIds = append(metricIds, goal.GetDependencies()...)
	}

	return
}

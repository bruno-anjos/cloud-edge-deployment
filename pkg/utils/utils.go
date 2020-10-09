package utils

import (
	"math"
)

const (
	ArchimedesServiceName = "archimedes"
	DeployerServiceName   = "deployer"
	SchedulerServiceName  = "scheduler"
	AutonomicServiceName  = "autonomic"
)

type Location struct {
	X float64
	Y float64
}

func (l *Location) CalcDist(l2 *Location) float64 {
	dX := l2.X - l.X
	dY := l2.Y - l.Y
	return math.Sqrt(math.Pow(dX, 2) + math.Pow(dY, 2))
}

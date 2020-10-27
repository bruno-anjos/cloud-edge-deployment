package utils

import (
	"fmt"
	"math"

	"github.com/golang/geo/s2"
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

func (l *Location) GetId() string {
	return fmt.Sprintf("%f_%f", l.X, l.Y)
}

func (l *Location) ToLatLng() s2.LatLng {
	return s2.LatLngFromDegrees(l.Y, l.X)
}

func FromLatLngToLocation(ll *s2.LatLng) *Location {
	return Location{
		X: ll.Lng.Degrees(),
		Y: ll.Lat.Degrees(),
	}
}

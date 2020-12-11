package servers

import (
	"fmt"
	"os"

	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s1"
)

const (
	TCP string = "tcp"
	UDP string = "udp"
)

var (
	ArchimedesLocalHostPort = getLocalHostPort(archimedes.Port)
	AutonomicLocalHostPort  = getLocalHostPort(autonomic.Port)
	DeployerLocalHostPort   = getLocalHostPort(deployer.Port)
	SchedulerLocalHostPort  = getLocalHostPort(scheduler.Port)
)

func getLocalHostPort(port int) string {
	nodeIP, exists := os.LookupEnv(utils.NodeIPEnvVarName)
	if !exists {
		panic("no NODE_IP set in environment")
	}

	return fmt.Sprintf("%s:%d", nodeIP, port)
}

func RandomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[internalUtils.GetRandInt(len(letters))]
	}

	return string(s)
}

const (
	EarthRadius = 6_378
)

func ChordAngleToKM(angle s1.ChordAngle) float64 {
	return angle.Angle().Radians() * EarthRadius
}

package deployer

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	HeartbeatCheckerTimeout = 60
)

func SendHeartbeatInstanceToDeployer(deployerHostPort string) {
	serviceId := os.Getenv(utils.ServiceEnvVarName)
	instanceId := os.Getenv(utils.InstanceEnvVarName)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	log.Infof("will start sending heartbeats to %s as %s from %s", deployerHostPort, instanceId, serviceId)

	alivePath := GetServiceInstanceAlivePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, deployerHostPort, alivePath, nil)
	status, _ := utils.DoRequest(httpClient, req, nil)

	switch status {
	case http.StatusConflict:
		log.Debugf("service %s instance %s already has a heartbeat sender", serviceId, instanceId)
		return
	case http.StatusOK:
	default:
		panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
	}

	ticker := time.NewTicker((HeartbeatCheckerTimeout / 3) * time.Second)
	req = utils.BuildRequest(http.MethodPut, deployerHostPort, alivePath, nil)
	for {
		<-ticker.C
		log.Info("sending heartbeat to deployer")
		status, _ = utils.DoRequest(httpClient, req, nil)

		switch status {
		case http.StatusNotFound:
			log.Warnf("heartbeat to deployer retrieved not found")
		case http.StatusOK:
		default:
			panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
		}
	}
}

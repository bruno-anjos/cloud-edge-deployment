package autonomic

type deploymentConfig struct {
	StrategyId string
}

type DeploymentDTO struct {
	DeploymentId string
	StrategyId   string
	Children     []string
	ParentId     string
}

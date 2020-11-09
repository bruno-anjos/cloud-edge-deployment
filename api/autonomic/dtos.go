package autonomic

type deploymentConfig struct {
	StrategyId   string
	ExploringTTL int
}

type DeploymentDTO struct {
	DeploymentId string
	StrategyId   string
	Children     []string
	ParentId     string
}

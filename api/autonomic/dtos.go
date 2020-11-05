package autonomic

type deploymentConfig struct {
	StrategyId string
	Exploring  bool
}

type DeploymentDTO struct {
	DeploymentId string
	StrategyId   string
	Children     []string
	ParentId     string
}

package autonomic

type deploymentConfig struct {
	DepthFactor  float64
	StrategyId   string
	ExploringTTL int
}

type DeploymentDTO struct {
	DeploymentId string
	StrategyId   string
	Children     []string
	ParentId     string
}

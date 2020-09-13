package autonomic

type serviceConfig struct {
	StrategyId string
}

type ServiceDTO struct {
	ServiceId  string
	StrategyId string
	Children   []string
	ParentId   string
}

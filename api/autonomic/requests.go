package autonomic

type (
	AddServiceRequestBody  = serviceConfig
	ClosestNodeRequestBody = struct {
		Location  float64
		ToExclude map[string]struct{}
	}
)

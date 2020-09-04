package autonomic

type Metric interface {
	Id() string
	Value() interface{}
}

const (
	METRIC_NODE_ID = "METRIC_NODE_ID"

	METRIC_NUMBER_OF_INSTANCES_ID = "METRIC_NUMBER_OF_INSTANCES"
	METRIC_LOAD                   = "METRIC_LOAD"
	METRIC_LOAD_IN_VICINITY       = "METRIC_LOAD_IN_VICINITY"
	METRIC_LOCATION               = "METRIC_LOCATION"
	METRIC_LOCATION_IN_VICINITY   = "METRIC_LOCATION_VICINITY"

	// SERVICE METRICS
	METRIC_LOAD_PER_SERVICE                        = "METRIC_LOAD_PER_SERVICE"
	METRIC_LOAD_PER_SERVICE_IN_VICINITY            = "METRIC_LOAD_PER_SERVICE_IN_VICINITY"
	METRIC_LOAD_PER_SERVICE_IN_CHILD               = "METRIC_LOAD_PER_SERVICE_IN_CHILD"
	METRIC_MAX_LOAD_IN_CHILD                       = "METRIC_MAX_LOAD_IN_CHILD"
	METRIC_AGG_LOAD_PER_SERVICE_IN_CHILDREN        = "METRIC_AGG_LOAD_PER_SERVICE_IN_CHILDREN"
	METRIC_CLIENT_LATENCY_PER_SERVICE              = "METRIC_CLIENT_LATENCY"
	METRIC_CLIENT_LATENCY_PER_SERVICE_IN_VICINITY  = "METRIC_CLIENT_LATENCY_VICINITY"
	METRIC_PROCESSING_TIME_PER_SERVICE             = "METRIC_PROCESSING_TIME"
	METRIC_PROCESSING_TIME_PER_SERVICE_IN_VICINITY = "METRIC_PROCESSING_TIME_VICINITY"
	METRIC_AVERAGE_CLIENT_LOCATION                 = "METRIC_AVERAGE_CLIENT_LOCATION"
)

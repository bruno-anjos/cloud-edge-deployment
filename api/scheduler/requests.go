package scheduler

type (
	StartInstanceRequestBody = ContainerInstanceDTO
	StopInstanceRequestBody  struct {
		RemovePath string
		URL        string
	}
)

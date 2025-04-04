package scheduler

type TaskDependency struct {
	PreferredResourceID int
	ResourceType        uint8
	ResourceQuantity    uint8
}

type Task struct {
	Name              string
	EstimatedDuration float64
	Dependencies      []TaskDependency

	ID int
}

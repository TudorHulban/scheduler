package scheduler

type TaskDependency struct {
	PreferredResourceID int
	ResourceType        uint8
	ResourceQuantity    uint8
}

type TaskLoad struct {
	Load     float32
	LoadUnit uint8
}

type Task struct {
	Name         string
	Dependencies []TaskDependency

	TaskLoad

	ID                int
	InitiatorID       int
	EstimatedDuration int
}

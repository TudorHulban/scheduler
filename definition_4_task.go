package scheduler

type RunDependency struct {
	PreferredResourceID int
	ResourceType        uint8
	ResourceQuantity    uint8
}

type RunLoad struct {
	Load     float32
	LoadUnit uint8
}

type Run struct {
	Name         string
	Dependencies []RunDependency

	RunLoad

	ID                int64
	InitiatorID       int64
	EstimatedDuration int64
}

func (r *Run) GetNeededResourceTypes() []uint8 {
	resourceTypes := make(map[uint8]bool)

	for _, dependency := range r.Dependencies {
		resourceTypes[dependency.ResourceType] = true
	}

	result := make([]uint8, len(resourceTypes), len(resourceTypes))

	var ix uint16

	for rt := range resourceTypes {
		result[ix] = rt

		ix++
	}

	return result
}

package scheduler

// schedulingOption represents a potential slot for scheduling a task
type schedulingOption struct {
	WhenCanStart      int64
	SelectedResources []*ResourceScheduled
	Cost              float32
}

func (loc *Location) findBestSchedulingOption(possibilitiesResp *ResponseGetPossibilities, params *ParamsCanRun) (*schedulingOption, error) {
	var totalNeeded int

	for _, qty := range possibilitiesResp.resourcesNeededPerType {
		totalNeeded = totalNeeded + int(qty)
	}

	earliest, selectedResources := findEarliestSlot(
		&paramsFindEarliestSlot{
			Possibilities:    possibilitiesResp.Possibilities,
			NeededCount:      totalNeeded,
			OffsetDifference: possibilitiesResp.offsetedTimeInterval.SecondsOffset,
		},
	)

	result := &schedulingOption{
		WhenCanStart:      earliest,
		SelectedResources: selectedResources,
	}

	if earliest != _NoAvailability {
		for _, resource := range selectedResources {
			cost, _ := calculateTaskCost(params.TaskRun, resource)
			result.Cost += cost
		}
	}

	return result, nil
}

type paramsScheduleResources struct {
	Resources []*ResourceScheduled

	TimeInterval
	TaskRunID RunID
}

func (loc *Location) scheduleResources(params *paramsScheduleResources) {
	for _, resource := range params.Resources {
		resource.schedule[params.TimeInterval] = params.TaskRunID
	}
}

package scheduler

func (loc *Location) findBestSchedulingOption(possibilitiesResp *ResponseGetPossibilities, params *ParamsCanRun) (*SchedulingOption, error) {
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

	result := &SchedulingOption{
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
	loc.mu.Lock()

	for _, resource := range params.Resources {
		resource.schedule[params.TimeInterval] = params.TaskRunID
	}

	loc.mu.Unlock()
}

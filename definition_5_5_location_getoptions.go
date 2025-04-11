package scheduler

import "slices"

func (loc *Location) GetSchedulingOptions(params *ParamsCanRun) ([]*SchedulingOption, error) {
	possibilitiesResp, err := loc.GetPossibilities(params)
	if err != nil {
		return nil, err
	}

	options := make([]*SchedulingOption, 0)

	for timeSlot, resourcesByType := range possibilitiesResp.Possibilities {
		var cost float32

		for _, resource := range resourcesByType {
			resourceCost, _ := calculateTaskCost(params.TaskRun, resource)
			cost = cost + resourceCost
		}

		options = append(
			options,
			&SchedulingOption{
				WhenCanStart:      timeSlot.TimeStart,
				SelectedResources: resourcesByType,
				Cost:              cost,
			},
		)

	}

	slices.SortFunc(
		options,
		func(a, b *SchedulingOption) int {
			if a.WhenCanStart < b.WhenCanStart {
				return -1
			}

			if a.WhenCanStart > b.WhenCanStart {
				return 1
			}
			return 0
		},
	)

	return options, nil
}

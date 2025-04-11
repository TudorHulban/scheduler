package scheduler

import "slices"

func (loc *Location) GetSchedulingOptions(params *ParamsCanRun) ([]*SchedulingOption, error) {
	possibilitiesResp, errGetPossibilities := loc.GetPossibilities(params)
	if errGetPossibilities != nil {
		return nil, errGetPossibilities
	}

	options := make([]*SchedulingOption, 0)

	for timeSlot, resources := range possibilitiesResp.Possibilities {
		// Group resources by type
		resourcesByType := make(ResourcesPerType)

		for _, res := range resources {
			resourcesByType[res.ResourceType] = append(
				resourcesByType[res.ResourceType],
				res,
			)
		}

		// If AllPossibilities, generate combinations; otherwise, use needed quantities
		if params.AllPossibilities {
			needed := possibilitiesResp.resourcesNeededPerType
			typeOptions := make([][]*ResourceScheduled, len(needed))
			indexes := make([]int, len(needed))
			typeKeys := make([]uint8, 0, len(needed))

			for rt := range needed {
				typeKeys = append(typeKeys, rt)
			}
			slices.Sort(typeKeys) // Consistent order

			// Prepare resource options per type
			for i, rt := range typeKeys {
				typeOptions[i] = resourcesByType[rt]
			}

			// Generate combinations
			for {
				var selectedResources []*ResourceScheduled
				var cost float32

				for i := range typeKeys {
					if indexes[i] < len(typeOptions[i]) {
						res := typeOptions[i][indexes[i]]
						selectedResources = append(selectedResources, res)
						resourceCost, _ := calculateTaskCost(params.TaskRun, res)
						cost = cost + resourceCost
					}
				}

				if len(selectedResources) == len(needed) { // Ensure full set
					options = append(
						options,
						&SchedulingOption{
							WhenCanStart:      timeSlot.TimeStart,
							SelectedResources: selectedResources,
							Cost:              cost,
						},
					)
				}

				// Next combination
				for i := len(typeKeys) - 1; i >= 0; i-- {
					indexes[i]++
					if indexes[i] < len(typeOptions[i]) {
						break
					}

					indexes[i] = 0
					if i == 0 {
						goto done
					}
				}
			}
		done:
		} else {
			// Original single-option logic
			var cost float32

			for _, resource := range resources {
				resourceCost, _ := calculateTaskCost(params.TaskRun, resource)
				cost = cost + resourceCost
			}

			options = append(
				options,
				&SchedulingOption{
					WhenCanStart:      timeSlot.TimeStart,
					SelectedResources: resources,
					Cost:              cost,
				},
			)
		}
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

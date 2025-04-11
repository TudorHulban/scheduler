package scheduler

import (
	"math"
	"slices"
)

func (loc *Location) findFallbackOption(possibilitiesResp *ResponseGetPossibilities, params *ParamsCanRun) *SchedulingOption {
	resourcesByType := make(map[uint8][]*ResourceScheduled)
	earliestByResource := make(map[*ResourceScheduled]int64)
	costByResource := make(map[*ResourceScheduled]float32)

	// First gather availability information for all resources
	for _, res := range loc.Resources {
		if slices.Contains(possibilitiesResp.resourceTypesNeeded, res.ResourceType) {
			when := res.findAvailableTime(
				&paramsFindAvailableTime{
					TimeStart:             possibilitiesResp.offsetedTimeInterval.TimeStart,
					MaximumTimeStart:      possibilitiesResp.offsetedTimeInterval.TimeEnd + params.TaskRun.EstimatedDuration,
					SecondsDuration:       params.TaskRun.EstimatedDuration,
					SecondsOffsetTask:     params.SecondsOffset,
					SecondsOffsetLocation: loc.LocationOffset,
				},
			)

			if when != _NoAvailability {
				whenTaskTime := when - possibilitiesResp.offsetedTimeInterval.SecondsOffset
				resourcesByType[res.ResourceType] = append(resourcesByType[res.ResourceType], res)
				earliestByResource[res] = whenTaskTime

				// Cache cost calculation
				cost, _ := calculateTaskCost(params.TaskRun, res)
				costByResource[res] = cost
			}
		}
	}

	// Check if we have enough resources of each type
	for resourceType, needed := range possibilitiesResp.resourcesNeededPerType {
		if len(resourcesByType[resourceType]) < int(needed) {
			// Not enough resources of this type available
			return &SchedulingOption{
				WhenCanStart:      params.TimeEnd,
				SelectedResources: nil,
				Cost:              0,
			}
		}
	}

	// Group resources by their earliest available time
	timeToResources := make(map[int64][]*ResourceScheduled)
	var allTimes []int64

	for _, resources := range resourcesByType {
		for _, res := range resources {
			t := earliestByResource[res]
			timeToResources[t] = append(timeToResources[t], res)

			// Keep track of unique times
			if len(timeToResources[t]) == 1 {
				allTimes = append(allTimes, t)
			}
		}
	}

	slices.Sort(allTimes)

	// Find the earliest time where we can schedule all required resources
	earliestFallback := _NoAvailability
	selectedCombination := make([]*ResourceScheduled, 0)
	lowestCost := float32(math.MaxFloat32)

	var totalNeeded int
	for _, qty := range possibilitiesResp.resourcesNeededPerType {
		totalNeeded = totalNeeded + int(qty)
	}

	// For each possible start time
	for _, startTime := range allTimes {
		if startTime > params.TimeEnd {
			break
		}

		// Collect all available resources at or before this time
		availableResources := make(map[uint8][]*ResourceScheduled)
		for t := allTimes[0]; t <= startTime; t++ {
			for _, res := range timeToResources[t] {
				availableResources[res.ResourceType] = append(
					availableResources[res.ResourceType],
					res,
				)
			}
		}

		// Check if we have enough resources of each type
		hasEnough := true
		for rType, needed := range possibilitiesResp.resourcesNeededPerType {
			if len(availableResources[rType]) < int(needed) {
				hasEnough = false
				break
			}
		}

		if !hasEnough {
			continue
		}

		// Try all possible combinations of resources to find cheapest
		combinations := generateCheapestCombinations(
			&paramsGenerateCheapestCombinations{
				AvailableResources:     availableResources,
				ResourcesNeededPerType: possibilitiesResp.resourcesNeededPerType,
				CostByResource:         costByResource,
			},
		)

		for _, combo := range combinations {
			// Calculate total cost
			comboCost := float32(0)

			for _, res := range combo {
				comboCost += costByResource[res]
			}

			if comboCost < lowestCost {
				lowestCost = comboCost
				selectedCombination = combo
				earliestFallback = startTime
			}
		}

		// If we found a valid combination, we can stop looking
		if len(selectedCombination) == totalNeeded {
			break
		}
	}

	if earliestFallback == _NoAvailability || len(selectedCombination) != totalNeeded {
		return &SchedulingOption{
			WhenCanStart:      params.TimeEnd,
			SelectedResources: nil,
			Cost:              0,
		}
	}

	return &SchedulingOption{
		WhenCanStart:      earliestFallback,
		SelectedResources: selectedCombination,
		Cost:              lowestCost,
	}
}

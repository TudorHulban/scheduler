package scheduler

import (
	"math"
	"slices"

	goerrors "github.com/TudorHulban/go-errors"
)

type ResponseGetPossibilities struct {
	Possibilities ResourcesPerTimeInterval

	resourceTypesNeeded    []uint8
	resourcesNeededPerType map[uint8]uint16

	offsetedTimeInterval TimeInterval
}

// GetPossibilities returns all possible time slots when resources are available
func (loc *Location) GetPossibilities(params *ParamsCanRun) (*ResponseGetPossibilities, error) {
	if params.TimeEnd-params.TimeStart < params.TaskRun.EstimatedDuration {
		return nil,
			goerrors.ErrValidation{
				Caller: "GetPossibilities",
				Issue: goerrors.ErrInvalidInput{
					InputName: "ParamsCanRun - interval too short",
				},
			}
	}

	resourceTypeCandidates := make(map[uint8][]*ResourceScheduled)
	resourceTypesNeeded := params.TaskRun.GetNeededResourceTypes()
	resourcesNeededPerType := params.TaskRun.GetNeededResourcesPerType()

	for _, candidate := range loc.Resources {
		if slices.Contains(resourceTypesNeeded, candidate.ResourceType) {
			resourceTypeCandidates[candidate.ResourceType] = append(
				resourceTypeCandidates[candidate.ResourceType],
				candidate,
			)
		}
	}

	offsetDifference := params.SecondsOffset - loc.LocationOffset

	offsetedTimeInterval := TimeInterval{
		TimeStart:     params.TimeStart + offsetDifference,
		TimeEnd:       params.TimeEnd + offsetDifference,
		SecondsOffset: offsetDifference,
	}

	possibilities := populatePossibilities(
		&paramsPopulatePossibilities{
			Candidates:             resourceTypeCandidates,
			ResourcesNeededPerType: resourcesNeededPerType,
			TimeInterval:           offsetedTimeInterval,

			Duration: params.TaskRun.EstimatedDuration,
		},
	)

	return &ResponseGetPossibilities{
			Possibilities: possibilities,

			resourceTypesNeeded:    resourceTypesNeeded,
			resourcesNeededPerType: resourcesNeededPerType,
			offsetedTimeInterval:   offsetedTimeInterval,
		},
		nil
}

type ResponseCanRun struct {
	WhenCanStart int64
	Cost         float32
	WasScheduled bool
}

// CanSchedule returns zero for WhenCanStart if it can run within passed interval and
// also schedules the task to the cheapest available resource and provides the cost.
//
// If it cannot run at TimeStart, it provides the timestamp
// from which it could in WhenCanStart and the cost of this run.
func (loc *Location) CanSchedule(params *ParamsCanRun) (*ResponseCanRun, error) {
	possibilitiesResp, errGet := loc.GetPossibilities(params)
	if errGet != nil {
		return nil,
			errGet
	}

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

	var totalCost float32

	if earliest != _NoAvailability {
		for _, resource := range selectedResources {
			cost, _ := calculateTaskCost(params.TaskRun, resource)
			totalCost = totalCost + cost
		}

		if earliest == params.TimeStart {
			// Schedule all resources atomically
			timeInterval := TimeInterval{
				TimeStart:     earliest + possibilitiesResp.offsetedTimeInterval.SecondsOffset,
				TimeEnd:       earliest + params.TaskRun.EstimatedDuration + possibilitiesResp.offsetedTimeInterval.SecondsOffset,
				SecondsOffset: loc.LocationOffset,
			}
			runID := RunID(params.TaskRun.ID)

			for _, resource := range selectedResources {
				resource.schedule[timeInterval] = runID
			}

			return &ResponseCanRun{
					WhenCanStart: _ScheduledForStart,
					Cost:         totalCost,
					WasScheduled: true,
				},
				nil
		}

		return &ResponseCanRun{
				WhenCanStart: earliest,
				Cost:         totalCost,
				WasScheduled: false,
			},
			nil
	}

	// More efficient algorithm for finding fallback resources
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
			return &ResponseCanRun{
					WhenCanStart: params.TimeEnd,
					Cost:         0,
					WasScheduled: false,
				},
				nil
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

	if earliestFallback != _NoAvailability && len(selectedCombination) == totalNeeded {
		// Schedule if we're starting immediately
		if earliestFallback == params.TimeStart {
			// Schedule all resources atomically
			timeInterval := TimeInterval{
				TimeStart:     earliestFallback + possibilitiesResp.offsetedTimeInterval.SecondsOffset,
				TimeEnd:       earliestFallback + params.TaskRun.EstimatedDuration + possibilitiesResp.offsetedTimeInterval.SecondsOffset,
				SecondsOffset: loc.LocationOffset,
			}

			for _, resource := range selectedCombination {
				resource.schedule[timeInterval] = RunID(params.TaskRun.ID)
			}
		}

		return &ResponseCanRun{
				WhenCanStart: earliestFallback,
				Cost:         lowestCost,
				WasScheduled: earliestFallback == params.TimeStart,
			},
			nil
	}

	return &ResponseCanRun{
			WhenCanStart: params.TimeEnd,
			Cost:         0,
			WasScheduled: false,
		},
		nil
}

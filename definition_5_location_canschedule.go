package scheduler

import (
	"slices"

	goerrors "github.com/TudorHulban/go-errors"
)

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
	defer traceExit()

	if params.TimeEnd-params.TimeStart < params.TaskRun.EstimatedDuration {
		return nil,
			goerrors.ErrValidation{
				Caller: "CanSchedule",
				Issue: goerrors.ErrInvalidInput{
					InputName: "ParamsCanRun - interval too short",
				},
			}
	}

	resourceTypeCandidates := make(map[uint8][]*Resource)
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
	start := params.TimeStart + offsetDifference
	end := params.TimeEnd + offsetDifference

	possibilities := populatePossibilities(
		&paramsPopulatePossibilities{
			Candidates:             resourceTypeCandidates,
			ResourcesNeededPerType: resourcesNeededPerType,
			TimeInterval: TimeInterval{
				TimeStart:     start,
				TimeEnd:       end,
				SecondsOffset: loc.LocationOffset,
			},
			Duration: params.TaskRun.EstimatedDuration,
		},
	)

	var totalNeeded int

	for _, qty := range resourcesNeededPerType {
		totalNeeded = totalNeeded + int(qty)
	}

	earliest, selectedResources := findEarliestSlot(
		possibilities,
		totalNeeded,
		offsetDifference,
	)

	var totalCost float32

	if earliest != _NoAvailability {
		for _, resource := range selectedResources {
			cost, _ := calculateTaskCost(params.TaskRun, resource)
			totalCost = totalCost + cost
		}

		if earliest == params.TimeStart {
			for _, resource := range selectedResources {
				resource.schedule[TimeInterval{
					TimeStart:     earliest + offsetDifference,
					TimeEnd:       earliest + params.TaskRun.EstimatedDuration + offsetDifference,
					SecondsOffset: loc.LocationOffset,
				}] = RunID(params.TaskRun.ID)
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

	earliestFallback := _NoAvailability
	fallbackByTime := make(map[int64][]*Resource)
	totalCost = 0
	for _, res := range loc.Resources {
		if slices.Contains(resourceTypesNeeded, res.ResourceType) {
			when := res.findAvailableTime(
				&paramsFindAvailableTime{
					TimeStart:             start,
					MaximumTimeStart:      end + params.TaskRun.EstimatedDuration,
					SecondsDuration:       params.TaskRun.EstimatedDuration,
					SecondsOffsetTask:     params.SecondsOffset,
					SecondsOffsetLocation: loc.LocationOffset,
				},
			)
			if when != _NoAvailability {
				whenTaskTime := when - offsetDifference
				fallbackByTime[whenTaskTime] = append(fallbackByTime[whenTaskTime], res)
				if earliestFallback == _NoAvailability || whenTaskTime < earliestFallback {
					earliestFallback = whenTaskTime
				}
			}
		}
	}

	var fallbackResources []*Resource

	if earliestFallback != _NoAvailability {
		for whenTaskTime := earliestFallback; whenTaskTime <= end; whenTaskTime++ {
			typeCounts := make(map[uint8]int)
			availableResources := make([]*Resource, 0)
			// Check resources available at or before this time
			for t := earliestFallback; t <= whenTaskTime; t++ {
				if resources, ok := fallbackByTime[t]; ok {
					for _, res := range resources {
						if typeCounts[res.ResourceType] < int(resourcesNeededPerType[res.ResourceType]) {
							availableResources = append(availableResources, res)
							typeCounts[res.ResourceType]++
						}
					}
				}
			}
			if len(availableResources) >= totalNeeded {
				earliestFallback = whenTaskTime
				fallbackResources = availableResources[:totalNeeded]
				totalCost = 0
				for _, res := range fallbackResources {
					cost, _ := calculateTaskCost(params.TaskRun, res)
					totalCost += cost
				}
				break
			}
		}
	}

	if earliestFallback != _NoAvailability && len(fallbackResources) == totalNeeded {
		if earliestFallback == params.TimeStart {
			for _, resource := range fallbackResources {
				resource.schedule[TimeInterval{
					TimeStart:     earliestFallback + offsetDifference,
					TimeEnd:       earliestFallback + params.TaskRun.EstimatedDuration + offsetDifference,
					SecondsOffset: loc.LocationOffset,
				}] = RunID(params.TaskRun.ID)
			}
		}
		return &ResponseCanRun{
			WhenCanStart: earliestFallback,
			Cost:         totalCost,
			WasScheduled: earliestFallback == params.TimeStart,
		}, nil
	}

	return &ResponseCanRun{
			WhenCanStart: params.TimeEnd,
			Cost:         totalCost,
			WasScheduled: false,
		},
		nil
}

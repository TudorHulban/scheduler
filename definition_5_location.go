package scheduler

import (
	"slices"
	"sort"

	goerrors "github.com/TudorHulban/go-errors"
	"github.com/asaskevich/govalidator"
)

type Location struct {
	Name      string
	Resources []*Resource

	ID             int64
	LocationOffset int64
}

type ParamsNewLocation struct {
	Name      string      `valid:"required"`
	Resources []*Resource `valid:"required"`

	ID             int64 `valid:"required"`
	LocationOffset int64
}

func NewLocation(params *ParamsNewLocation) (*Location, error) {
	if _, errValidation := govalidator.ValidateStruct(params); errValidation != nil {
		return nil,
			goerrors.ErrServiceValidation{
				ServiceName: "Organigram",
				Caller:      "CreateCompany",
				Issue:       errValidation,
			}
	}

	return &Location{
			ID:             params.ID,
			Name:           params.Name,
			LocationOffset: params.LocationOffset,

			Resources: params.Resources,
		},
		nil
}

type ParamsCanRun struct {
	TimeInterval

	TaskRun *Run
}

type ResponseCanRun struct {
	WhenCanStart int64
	Cost         float32
	WasScheduled bool
}

// CanRun returns zero for WhenCanStart if it can run within passed nterval and
// also schedules the task to the cheapest available resource and provides the cost.
//
// If it cannot run within interval, it provides the timestamp
// from which it could in WhenCanStart and the cost of this run.
func (loc *Location) CanRun(params *ParamsCanRun) (*ResponseCanRun, error) {
	if params.TimeEnd-params.TimeStart < params.TaskRun.EstimatedDuration {
		return &ResponseCanRun{
				WhenCanStart: params.TimeEnd,
				Cost:         0,
				WasScheduled: false,
			},
			nil
	}

	resourceTypeCandidates := make(map[uint8][]*Resource)
	resourceTypesNeeded := params.TaskRun.GetNeededResourceTypes()

	for _, candidate := range loc.Resources {
		if slices.Contains(resourceTypesNeeded, candidate.ResourceType) {
			resourceTypeCandidates[candidate.ResourceType] = append(resourceTypeCandidates[candidate.ResourceType], candidate)
		}
	}

	possibilities := make(map[TimeInterval][]*Resource)

	offsetDifference := params.SecondsOffset - loc.LocationOffset
	start := params.TimeStart + offsetDifference
	end := params.TimeEnd + offsetDifference

	for _, candidates := range resourceTypeCandidates {
		sort.Slice(
			candidates,
			func(i, j int) bool {
				return candidates[i].costPerLoadUnit[params.TaskRun.LoadUnit] < candidates[j].costPerLoadUnit[params.TaskRun.LoadUnit]
			},
		)

		for ix, candidate := range candidates {
			availSlots, available := candidate.GetAvailability(
				&TimeInterval{
					TimeStart:     start,
					TimeEnd:       end,
					SecondsOffset: loc.LocationOffset,
				},
			)
			if ix == 0 && available {
				slot := TimeInterval{
					TimeStart:     start,
					TimeEnd:       end,
					SecondsOffset: loc.LocationOffset,
				}

				possibilities[slot] = append(possibilities[slot], candidate)
			} else {
				for _, slot := range availSlots {
					if slot.TimeEnd-slot.TimeStart >= params.TaskRun.EstimatedDuration {
						possibilities[slot] = append(possibilities[slot], candidate)
					}
				}
			}
		}
	}

	earliest := int64(_NoAvailability)

	var bestResources []*Resource
	neededCount := len(resourceTypesNeeded)

	for slot, resources := range possibilities {
		if len(resources) >= neededCount {
			startTaskTime := slot.TimeStart - offsetDifference

			if earliest == _NoAvailability || startTaskTime < earliest {
				earliest = startTaskTime
				bestResources = resources[:neededCount] // Take only needed count
			}
		}
	}

	var totalCost float32
	if earliest != _NoAvailability {
		for _, resource := range bestResources {
			cost, _ := calculateTaskCost(params.TaskRun, resource)
			totalCost = totalCost + cost

			resource.schedule[TimeInterval{
				TimeStart:     earliest + (params.SecondsOffset - loc.LocationOffset),
				TimeEnd:       earliest + params.TaskRun.EstimatedDuration + (params.SecondsOffset - loc.LocationOffset),
				SecondsOffset: loc.LocationOffset,
			}] = RunID(params.TaskRun.ID)
		}

		return &ResponseCanRun{
				WhenCanStart: earliest - params.TimeStart, // Relative to TimeStart
				Cost:         totalCost,
				WasScheduled: true,
			},
			nil
	}

	for _, res := range loc.Resources[:neededCount] { // Fallback to cheapest available later
		cost, _ := calculateTaskCost(params.TaskRun, res)
		totalCost = totalCost + cost
	}

	return &ResponseCanRun{
			WhenCanStart: params.TimeEnd,
			Cost:         totalCost,
			WasScheduled: false,
		},
		nil
}

// GetRunCost returns earliest when task could start and at what cost but does not schedule the task.
// func (loc *Location) GetRunCost(params *ParamsCanRun) (*ResponseCanRun, error) {
// 	var earliestStartTimeOverall int64 = 0
// 	var totalCost float32 = 0
// 	earliestStartTimes := make(map[uint8]int64)
// 	minCosts := make(map[uint8]float32)

// 	for _, dependency := range params.TaskRun.Dependencies {
// 		earliestStartTimeForType := _NoAvailability
// 		var maxCostForType float32 = math.MaxFloat32
// 		var resourceFound bool

// 		for _, res := range loc.Resources {
// 			if res.ResourceType == dependency.ResourceType {
// 				resourceFound = true

// 				// Convert task start time to resource's time zone for checking availability
// 				checkStartTime := params.TimeStart + (params.TaskOffset - loc.LocationOffset)

// 				availableStartTime := res.findEarliestAvailableTimeFrom(
// 					&paramsFindEarliestAvailableTime{
// 						TimeStart:      checkStartTime,
// 						Duration:       params.TaskRun.EstimatedDuration,
// 						OffsetTask:     params.TaskOffset,
// 						OffsetLocation: loc.LocationOffset,
// 					},
// 				)

// 				if availableStartTime != _NoAvailability {
// 					costPerUnit, ok := res.costPerLoadUnit[params.TaskRun.LoadUnit]
// 					if !ok {
// 						continue // Resource doesn't support this load unit
// 					}

// 					cost := params.TaskRun.Load * costPerUnit

// 					// Convert available start time back to task's timezone for comparison
// 					availableStartTimeInTaskTZ := availableStartTime - (params.TaskOffset - loc.LocationOffset)

// 					if earliestStartTimeForType == -1 || availableStartTimeInTaskTZ < earliestStartTimeForType {
// 						earliestStartTimeForType = availableStartTimeInTaskTZ
// 					}

// 					if cost < maxCostForType {
// 						maxCostForType = cost
// 					}
// 				}
// 			}
// 		}

// 		if !resourceFound {
// 			return nil,
// 				fmt.Errorf(
// 					"no resource of type %d found at location",
// 					dependency.ResourceType,
// 				)
// 		}

// 		if earliestStartTimeForType == _NoAvailability {
// 			return nil,
// 				fmt.Errorf(
// 					"no available time slot found for resource type %d",
// 					dependency.ResourceType,
// 				)
// 		}

// 		earliestStartTimes[dependency.ResourceType] = earliestStartTimeForType
// 		minCosts[dependency.ResourceType] = maxCostForType
// 	}

// 	// Find the latest of all earliest start times
// 	for _, startTime := range earliestStartTimes {
// 		if startTime > earliestStartTimeOverall {
// 			earliestStartTimeOverall = startTime
// 		}
// 	}

// 	for _, cost := range minCosts {
// 		if cost != math.MaxFloat32 {
// 			totalCost = totalCost + cost

// 			continue
// 		}

// 		return nil,
// 			errors.New("could not determine cost for all dependencies")
// 	}

// 	return &ResponseCanRun{
// 			WhenCanStart: earliestStartTimeOverall,
// 			Cost:         totalCost,
// 		},
// 		nil
// }

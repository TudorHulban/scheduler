package scheduler

import (
	"errors"
	"fmt"
	"math"
)

type Location struct {
	ID             int64
	LocationOffset int64

	Name      string
	Resources []*Resource
}

type ParamsCanRun struct {
	TimeStart  int64
	TimeEnd    int64
	TaskOffset int64

	Task *Task
}

type ResponseCanRun struct {
	WhenCanStart    int64
	CostIfScheduled float32
}

// CanRunCheapest returns zero for WhenCanStart if it can run within passed nterval and
// also schedules the task to the cheapest available resource and provides the cost.
//
// If it cannot run within interval, it provides the timestamp from which it could in WhenCanStart but no cost.
func (loc *Location) CanRunCheapest(params *ParamsCanRun) (*ResponseCanRun, error) {
	task := params.Task
	startTime := params.TimeStart
	endTime := params.TimeEnd

	availableResources := make(map[int]*Resource) // Map of resource ID to resource

	// Check for available resources that meet all dependencies
	for _, dependency := range task.Dependencies {
		foundResource := false
		var cheapestResource *Resource
		var minCost float32 = math.MaxFloat32

		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				// Convert task times to resource's time zone
				taskStartInResourceTZ := startTime + (params.TaskOffset - loc.LocationOffset)
				taskEndInResourceTZ := endTime + (params.TaskOffset - loc.LocationOffset)

				overlap := res.isAvailable(taskStartInResourceTZ, taskEndInResourceTZ)
				if overlap[0] == 0 && overlap[1] == 0 { // No overlap, resource is available
					cost, errCost := calculateTaskCost(task, res)
					if errCost != nil {
						continue // Skip this resource if cost calculation fails
					}

					if cheapestResource == nil || cost < minCost {
						cheapestResource = res
						minCost = cost
						foundResource = true
					}
				}
			}
		}

		if !foundResource {
			earliestStartTime := _NoAvailability

			for _, res := range loc.Resources {
				if res.ResourceType == dependency.ResourceType {
					earliestAvailable := res.findEarliestAvailableTime(
						&paramsFindEarliestAvailableTime{
							TimeStart:      endTime + (params.TaskOffset - loc.LocationOffset),
							Duration:       task.EstimatedDuration,
							OffsetTask:     params.TaskOffset,
							OffsetLocation: loc.LocationOffset,
						},
					)
					if earliestAvailable != _NoAvailability {
						if earliestStartTime == _NoAvailability || earliestAvailable < earliestStartTime {
							earliestStartTime = earliestAvailable
						}
					}
				}
			}

			if earliestStartTime != _NoAvailability {
				return &ResponseCanRun{
						WhenCanStart:    earliestStartTime,
						CostIfScheduled: 0,
					},
					nil
			}

			return &ResponseCanRun{
					WhenCanStart:    endTime,
					CostIfScheduled: 0,
				},
				nil // No resource available at all
		}

		if cheapestResource != nil {
			availableResources[cheapestResource.ID] = cheapestResource
		}
	}

	// If all dependencies can be met, schedule on the cheapest combination (simplified for now)
	if len(availableResources) == len(task.Dependencies) {
		var chosenResource *Resource

		var minTotalCost float32 = math.MaxFloat32

		// Simple approach: find the cheapest resource that can fulfill the first dependency
		if len(task.Dependencies) > 0 {
			for _, res := range availableResources {
				if res.ResourceType == task.Dependencies[0].ResourceType {
					cost, err := calculateTaskCost(task, res)
					if err != nil {
						continue
					}

					if chosenResource == nil || cost < minTotalCost {
						chosenResource = res
						minTotalCost = cost
					}
				}
			}
		}

		if chosenResource != nil && minTotalCost != math.MaxFloat32 {
			scheduleStart := startTime + params.TaskOffset
			scheduleEnd := endTime + params.TaskOffset
			interval := [3]int64{
				scheduleStart,
				scheduleEnd,
				loc.LocationOffset,
			}

			chosenResource.schedule[interval] = task.ID

			return &ResponseCanRun{
					WhenCanStart:    0,
					CostIfScheduled: minTotalCost,
				},
				nil
		}
	}

	// If not all dependencies can be met within the interval, return earliest start time
	earliestStartTimeOverall := _NoAvailability

	for _, dependency := range task.Dependencies {
		earliestStartTime := _NoAvailability

		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				earliestAvailable := res.findEarliestAvailableTime(
					&paramsFindEarliestAvailableTime{
						TimeStart:      endTime + (params.TaskOffset - loc.LocationOffset),
						Duration:       task.EstimatedDuration,
						OffsetTask:     params.TaskOffset,
						OffsetLocation: loc.LocationOffset,
					},
				)
				if earliestAvailable != _NoAvailability {
					if earliestStartTime == _NoAvailability || earliestAvailable < earliestStartTime {
						earliestStartTime = earliestAvailable
					}
				}
			}
		}

		if earliestStartTime != _NoAvailability {
			if earliestStartTimeOverall == _NoAvailability || earliestStartTime < earliestStartTimeOverall {
				earliestStartTimeOverall = earliestStartTime
			}
		}
	}

	if earliestStartTimeOverall != _NoAvailability {
		return &ResponseCanRun{
				WhenCanStart:    earliestStartTimeOverall,
				CostIfScheduled: 0,
			},
			nil
	}

	return &ResponseCanRun{
			WhenCanStart:    endTime,
			CostIfScheduled: 0,
		},
		errors.New("no suitable resources available")
}

// CanRun returns zero for WhenCanStart if it can run within passed nterval and
// also schedules the task to the first available resource and provides the cost.
//
// If it cannot run within interval, it provides the timestamp from which it could in WhenCanStart but no cost.
func (loc *Location) CanRun(params *ParamsCanRun) (*ResponseCanRun, error) {
	task := params.Task
	endTime := params.TimeEnd

	scheduledResources := make(map[uint8]*Resource) // Track if a resource type is already scheduled
	var totalCost float32 = 0

	// Check if all dependencies can be met within the time interval
	for _, dependency := range task.Dependencies {
		var foundResource bool

		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				if _, alreadyScheduled := scheduledResources[res.ResourceType]; !alreadyScheduled {
					// Convert task times to resource's time zone
					taskStartInResourceTZ := params.TimeStart + params.TaskOffset - loc.LocationOffset
					taskEndInResourceTZ := params.TimeEnd + params.TaskOffset - loc.LocationOffset

					overlap := res.isAvailable(taskStartInResourceTZ, taskEndInResourceTZ)
					if overlap[0] == 0 && overlap[1] == 0 { // No overlap, resource is available
						scheduledResources[res.ResourceType] = res

						cost, err := calculateTaskCost(task, res)
						if err != nil {
							fmt.Printf("Error calculating cost for task %d on resource %d: %v\n", task.ID, res.ID, err)
							// Decide how to handle cost calculation errors - perhaps skip this resource type
						} else {
							totalCost += cost
						}

						foundResource = true

						break // Found a resource for this dependency
					}
				}
			}
		}

		if !foundResource {
			earliestStartTime := _NoAvailability

			for _, res := range loc.Resources {
				if res.ResourceType == dependency.ResourceType {
					earliestAvailable := res.findEarliestAvailableTime(
						&paramsFindEarliestAvailableTime{
							TimeStart:      endTime + (params.TaskOffset - loc.LocationOffset),
							Duration:       task.EstimatedDuration,
							OffsetTask:     params.TaskOffset,
							OffsetLocation: loc.LocationOffset,
						},
					)
					if earliestAvailable != _NoAvailability {
						if earliestStartTime == _NoAvailability || earliestAvailable < earliestStartTime {
							earliestStartTime = earliestAvailable
						}
					}
				}
			}

			if earliestStartTime != _NoAvailability {
				return &ResponseCanRun{
						WhenCanStart:    earliestStartTime,
						CostIfScheduled: 0,
					},
					nil
			}

			return &ResponseCanRun{
					WhenCanStart:    endTime,
					CostIfScheduled: 0,
				},
				errors.New("no suitable resources available")
		}
	}

	// If all dependencies can be met, schedule the task
	if len(scheduledResources) == len(task.Dependencies) {
		interval := [3]int64{
			params.TimeStart + params.TaskOffset,
			endTime + params.TaskOffset,
			loc.LocationOffset,
		}

		for _, res := range scheduledResources {
			res.schedule[interval] = task.ID // Schedule on all the resources that met dependencies
		}

		return &ResponseCanRun{
				WhenCanStart:    0,
				CostIfScheduled: totalCost,
			},
			nil
	}

	// If not all dependencies could be met within the interval, return earliest start time
	earliestStartTimeOverall := _NoAvailability

	for _, dependency := range task.Dependencies {
		earliestStartTime := _NoAvailability

		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				earliestAvailable := res.findEarliestAvailableTime(
					&paramsFindEarliestAvailableTime{
						TimeStart:      endTime + params.TaskOffset - loc.LocationOffset,
						Duration:       params.Task.EstimatedDuration,
						OffsetTask:     params.TaskOffset,
						OffsetLocation: loc.LocationOffset,
					},
				)
				if earliestAvailable != _NoAvailability {
					if earliestStartTime == _NoAvailability || earliestAvailable < earliestStartTime {
						earliestStartTime = earliestAvailable
					}
				}
			}
		}
		if earliestStartTime != -1 {
			if earliestStartTimeOverall == -1 || earliestStartTime < earliestStartTimeOverall {
				earliestStartTimeOverall = earliestStartTime
			}
		}
	}

	if earliestStartTimeOverall != -1 {
		return &ResponseCanRun{
				WhenCanStart:    earliestStartTimeOverall,
				CostIfScheduled: 0,
			},
			nil
	}

	return &ResponseCanRun{
			WhenCanStart:    endTime,
			CostIfScheduled: 0,
		},
		errors.New("could not schedule task")
}

// GetRunCost returns earliest when task could start and at what cost but does not schedule the task.
func (loc *Location) GetRunCost(params *ParamsCanRun) (*ResponseCanRun, error) {
	var earliestStartTimeOverall int64 = 0
	var totalCost float32 = 0
	earliestStartTimes := make(map[uint8]int64)
	minCosts := make(map[uint8]float32)

	for _, dependency := range params.Task.Dependencies {
		earliestStartTimeForType := _NoAvailability
		var maxCostForType float32 = math.MaxFloat32
		var resourceFound bool

		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				resourceFound = true

				// Convert task start time to resource's time zone for checking availability
				checkStartTime := params.TimeStart + (params.TaskOffset - loc.LocationOffset)

				availableStartTime := res.findEarliestAvailableTimeFrom(
					&paramsFindEarliestAvailableTime{
						TimeStart:      checkStartTime,
						Duration:       params.Task.EstimatedDuration,
						OffsetTask:     params.TaskOffset,
						OffsetLocation: loc.LocationOffset,
					},
				)

				if availableStartTime != _NoAvailability {
					costPerUnit, ok := res.costPerLoadUnit[params.Task.LoadUnit]
					if !ok {
						continue // Resource doesn't support this load unit
					}

					cost := params.Task.Load * costPerUnit

					// Convert available start time back to task's timezone for comparison
					availableStartTimeInTaskTZ := availableStartTime - (params.TaskOffset - loc.LocationOffset)

					if earliestStartTimeForType == -1 || availableStartTimeInTaskTZ < earliestStartTimeForType {
						earliestStartTimeForType = availableStartTimeInTaskTZ
					}

					if cost < maxCostForType {
						maxCostForType = cost
					}
				}
			}
		}

		if !resourceFound {
			return nil,
				fmt.Errorf(
					"no resource of type %d found at location",
					dependency.ResourceType,
				)
		}

		if earliestStartTimeForType == _NoAvailability {
			return nil,
				fmt.Errorf(
					"no available time slot found for resource type %d",
					dependency.ResourceType,
				)
		}

		earliestStartTimes[dependency.ResourceType] = earliestStartTimeForType
		minCosts[dependency.ResourceType] = maxCostForType
	}

	// Find the latest of all earliest start times
	for _, startTime := range earliestStartTimes {
		if startTime > earliestStartTimeOverall {
			earliestStartTimeOverall = startTime
		}
	}

	for _, cost := range minCosts {
		if cost != math.MaxFloat32 {
			totalCost = totalCost + cost

			continue
		}

		return nil,
			errors.New("could not determine cost for all dependencies")
	}

	return &ResponseCanRun{
			WhenCanStart:    earliestStartTimeOverall,
			CostIfScheduled: totalCost,
		},
		nil
}

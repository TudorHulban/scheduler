package scheduler

import (
	"errors"
	"fmt"
	"math"
)

type Location struct {
	ID             int
	LocationOffset int

	Name      string
	Resources []*Resource
}

type ParamsCanRun struct {
	TimeStart int
	TimeEnd   int
	GMTOffset int

	Task *Task
}

type ResponseCanRun struct {
	WhenCanStart    int
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
	taskOffset := params.GMTOffset
	locationOffset := loc.LocationOffset

	availableResources := make(map[int]*Resource) // Map of resource ID to resource

	// Check for available resources that meet all dependencies
	for _, dependency := range task.Dependencies {
		foundResource := false
		var cheapestResource *Resource
		var minCost float32 = math.MaxFloat32

		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				resourceOffset := 0 // Assuming resource's schedule offset is its local offset

				// Convert task times to resource's time zone
				taskStartInResourceTZ := startTime + (taskOffset - resourceOffset)
				taskEndInResourceTZ := endTime + (taskOffset - resourceOffset)

				overlap := res.isAvailable(taskStartInResourceTZ, taskEndInResourceTZ)
				if overlap[0] == 0 && overlap[1] == 0 { // No overlap, resource is available
					cost, err := calculateTaskCost(task, res)
					if err != nil {
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
			// Could not find a resource for this dependency within the time interval
			earliestStartTime := -1

			for _, res := range loc.Resources {
				if res.ResourceType == dependency.ResourceType {
					resourceOffset := 0 // Assuming resource's schedule offset is its local offset
					earliestAvailable := res.findEarliestAvailableTime(endTime+(taskOffset-resourceOffset), int(task.EstimatedDuration), taskOffset, locationOffset)
					if earliestAvailable != -1 {
						if earliestStartTime == -1 || earliestAvailable < earliestStartTime {
							earliestStartTime = earliestAvailable
						}
					}
				}
			}

			if earliestStartTime != -1 {
				return &ResponseCanRun{WhenCanStart: earliestStartTime, CostIfScheduled: 0}, nil
			}
			return &ResponseCanRun{WhenCanStart: endTime, CostIfScheduled: 0}, nil // No resource available at all
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
			scheduleStart := startTime + taskOffset
			scheduleEnd := endTime + taskOffset
			interval := [3]int{scheduleStart, scheduleEnd, locationOffset}
			chosenResource.schedule[interval] = task.ID
			return &ResponseCanRun{WhenCanStart: 0, CostIfScheduled: minTotalCost}, nil
		}
	}

	// If not all dependencies can be met within the interval, return earliest start time
	earliestStartTimeOverall := -1
	for _, dependency := range task.Dependencies {
		earliestStartTime := -1
		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				resourceOffset := 0 // Assuming resource's schedule offset is its local offset
				earliestAvailable := res.findEarliestAvailableTime(endTime+(taskOffset-resourceOffset), task.EstimatedDuration, taskOffset, locationOffset)
				if earliestAvailable != -1 {
					if earliestStartTime == -1 || earliestAvailable < earliestStartTime {
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
		return &ResponseCanRun{WhenCanStart: earliestStartTimeOverall, CostIfScheduled: 0}, nil
	}

	return &ResponseCanRun{WhenCanStart: endTime, CostIfScheduled: 0}, errors.New("no suitable resources available")
}

// CanRun returns zero for WhenCanStart if it can run within passed nterval and
// also schedules the task to the first available resource and provides the cost.
//
// If it cannot run within interval, it provides the timestamp from which it could in WhenCanStart but no cost.
func (loc *Location) CanRun(params *ParamsCanRun) (*ResponseCanRun, error) {
	task := params.Task
	startTime := params.TimeStart
	endTime := params.TimeEnd
	taskOffset := params.GMTOffset
	locationOffset := loc.LocationOffset

	scheduledResources := make(map[uint8]*Resource) // Track if a resource type is already scheduled
	var totalCost float32 = 0

	// Check if all dependencies can be met within the time interval
	for _, dependency := range task.Dependencies {
		foundResource := false
		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				if _, alreadyScheduled := scheduledResources[res.ResourceType]; !alreadyScheduled {
					resourceOffset := 0 // Assuming resource's schedule offset is its local offset

					// Convert task times to resource's time zone
					taskStartInResourceTZ := startTime + (taskOffset - resourceOffset)
					taskEndInResourceTZ := endTime + (taskOffset - resourceOffset)

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
			// Could not find a resource for this dependency within the time interval
			earliestStartTime := -1
			for _, res := range loc.Resources {
				if res.ResourceType == dependency.ResourceType {
					resourceOffset := 0 // Assuming resource's schedule offset is its local offset
					earliestAvailable := res.findEarliestAvailableTime(endTime+(taskOffset-resourceOffset), int(task.EstimatedDuration), taskOffset, locationOffset)
					if earliestAvailable != -1 {
						if earliestStartTime == -1 || earliestAvailable < earliestStartTime {
							earliestStartTime = earliestAvailable
						}
					}
				}
			}
			if earliestStartTime != -1 {
				return &ResponseCanRun{WhenCanStart: earliestStartTime, CostIfScheduled: 0}, nil
			}
			return &ResponseCanRun{WhenCanStart: endTime, CostIfScheduled: 0}, errors.New("no suitable resources available")
		}
	}

	// If all dependencies can be met, schedule the task
	if len(scheduledResources) == len(task.Dependencies) {
		scheduleStart := startTime + taskOffset
		scheduleEnd := endTime + taskOffset
		interval := [3]int{scheduleStart, scheduleEnd, locationOffset}
		for _, res := range scheduledResources {
			res.schedule[interval] = task.ID // Schedule on all the resources that met dependencies
		}
		return &ResponseCanRun{WhenCanStart: 0, CostIfScheduled: totalCost}, nil
	}

	// If not all dependencies could be met within the interval, return earliest start time
	earliestStartTimeOverall := -1
	for _, dependency := range task.Dependencies {
		earliestStartTime := -1
		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				resourceOffset := 0 // Assuming resource's schedule offset is its local offset
				earliestAvailable := res.findEarliestAvailableTime(endTime+(taskOffset-resourceOffset), int(task.EstimatedDuration), taskOffset, locationOffset)
				if earliestAvailable != -1 {
					if earliestStartTime == -1 || earliestAvailable < earliestStartTime {
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
		return &ResponseCanRun{WhenCanStart: earliestStartTimeOverall, CostIfScheduled: 0}, nil
	}

	return &ResponseCanRun{WhenCanStart: endTime, CostIfScheduled: 0}, errors.New("could not schedule task")
}

// GetRunCost returns earliest when task could start and at what cost but does not schedule the task.
func (loc *Location) GetRunCost(params *ParamsCanRun) (*ResponseCanRun, error) {
	task := params.Task
	startTime := params.TimeStart
	taskOffset := params.GMTOffset
	locationOffset := loc.LocationOffset

	earliestStartTimeOverall := 0
	var totalCost float32 = 0
	earliestStartTimes := make(map[uint8]int)
	minCosts := make(map[uint8]float32)

	for _, dependency := range task.Dependencies {
		earliestStartTimeForType := -1
		var maxCostForType float32 = math.MaxFloat32
		resourceFound := false

		for _, res := range loc.Resources {
			if res.ResourceType == dependency.ResourceType {
				resourceFound = true
				resourceOffset := 0 // Assuming resource's schedule offset is its local offset

				// Convert task start time to resource's time zone for checking availability
				checkStartTime := startTime + (taskOffset - resourceOffset)
				duration := int(task.EstimatedDuration)

				availableStartTime := res.findEarliestAvailableTimeFrom(checkStartTime, duration, taskOffset, locationOffset)

				if availableStartTime != -1 {
					costPerUnit, ok := res.costPerLoadUnit[task.TaskLoad.LoadUnit]
					if !ok {
						continue // Resource doesn't support this load unit
					}
					cost := task.TaskLoad.Load * costPerUnit

					// Convert available start time back to task's timezone for comparison
					availableStartTimeInTaskTZ := availableStartTime - (taskOffset - resourceOffset)

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
			return &ResponseCanRun{WhenCanStart: 0, CostIfScheduled: 0}, errors.New(fmt.Sprintf("no resource of type %d found at location", dependency.ResourceType))
		}

		if earliestStartTimeForType == -1 {
			return &ResponseCanRun{WhenCanStart: 0, CostIfScheduled: 0}, errors.New(fmt.Sprintf("no available time slot found for resource type %d", dependency.ResourceType))
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

	// Calculate the total cost
	for _, cost := range minCosts {
		if cost != math.MaxFloat32 {
			totalCost += cost
		} else {
			// Handle case where a cost wasn't found (shouldn't happen with the checks above)
			return &ResponseCanRun{WhenCanStart: 0, CostIfScheduled: 0}, errors.New("could not determine cost for all dependencies")
		}
	}

	return &ResponseCanRun{WhenCanStart: earliestStartTimeOverall, CostIfScheduled: totalCost}, nil
}

// Helper function to find the earliest available time slot on a resource from a given start time
func (res *Resource) findEarliestAvailableTimeFrom(startTime int, duration int, taskOffset int, locationOffset int) int {
	resourceOffset := 0 // Assuming resource's schedule offset is its local offset
	checkStart := startTime + (taskOffset - resourceOffset)
	checkEnd := checkStart + duration

	// Check availability from the given start time
	if res.isAvailable(checkStart, checkEnd) == [2]int{0, 0} {
		return checkStart
	}

	// In a real scenario, you'd need to look ahead in the schedule more comprehensively
	// This is a very basic placeholder looking at the next potential slot
	latestEndTime := checkStart
	for interval := range res.schedule {
		scheduleEnd := interval[1] - interval[2] // End time in UTC
		if scheduleEnd > latestEndTime {
			latestEndTime = scheduleEnd
		}
	}

	nextPossibleStart := latestEndTime
	nextPossibleEnd := nextPossibleStart + duration
	if res.isAvailable(nextPossibleStart, nextPossibleEnd) == [2]int{0, 0} {
		return nextPossibleStart
	}

	return -1 // Indicate no availability found
}

package scheduler

import (
	"context"
	"errors"
)

type Resource struct {
	Name        string
	schedule    map[[3]int]int // [3]int is [unix_start_time, unix_end_time, GMT offset], int is task ID
	CostPerHour float64

	ID           int
	ResourceType uint8
}

type ParamsNewResource struct {
	Name        string
	CostPerHour float64
}

func NewResource(params *ParamsNewResource) *Resource {
	return &Resource{
		Name:        params.Name,
		CostPerHour: params.CostPerHour,

		schedule: map[[3]int]int{},
	}
}

type ParamsTask struct {
	TimeStart int
	TimeEnd   int
	GMTOffset int

	TaskID int
}

func (res *Resource) RemoveTask(_ context.Context, params *ParamsTask) error {
	keysToDelete := []([3]int){}

	for interval, taskID := range res.schedule {
		if taskID == params.TaskID {
			if params.TimeStart <= interval[0] &&
				interval[1] <= params.TimeEnd &&
				interval[2] == params.GMTOffset {
				keysToDelete = append(
					keysToDelete,
					interval,
				)
			}
		}
	}

	if len(keysToDelete) == 0 {
		return errors.New("no schedules found within the given timeframe")
	}

	for _, keyToDelete := range keysToDelete {
		delete(res.schedule, keyToDelete)
	}

	return nil
}

func (res *Resource) isAvailable(timeStart, timeEnd int) [2]int {
	for interval := range res.schedule {
		scheduleStart := interval[0]
		scheduleEnd := interval[1]
		offset := interval[2]

		withOffsetScheduleStart := scheduleStart - offset
		withOffsetScheduleEnd := scheduleEnd - offset

		overlapStart := max(timeStart, withOffsetScheduleStart)
		overlapEnd := min(timeEnd, withOffsetScheduleEnd)

		if overlapStart < overlapEnd {
			return [2]int{overlapStart, overlapEnd}
		}
	}

	return [2]int{}
}

func (res *Resource) AddTask(_ context.Context, params *ParamsTask) ([2]int, error) {
	overlap := res.isAvailable(params.TimeStart, params.TimeEnd)

	if overlap == [2]int{} {
		res.schedule[[3]int{
			params.TimeStart,
			params.TimeEnd,
			params.GMTOffset,
		}] = params.TaskID

		return [2]int{},
			nil
	}

	return overlap,
		errors.New("busy")
}

// GetTasks returns a slice of when task ID finishes.
func (res *Resource) GetTasks(atTimestamp, offset int) [][2]int {
	var finishedTasks [][2]int

	for interval, taskID := range res.schedule {
		scheduleEnd := interval[1]

		// Adjust the times to offset
		scheduleEndUTC := scheduleEnd - interval[2]
		atTimestampUTC := atTimestamp - offset

		if scheduleEndUTC >= atTimestampUTC {
			finishedTasks = append(
				finishedTasks,
				[2]int{
					taskID,
					scheduleEnd,
				},
			)
		}
	}

	return finishedTasks
}

func (res *Resource) findEarliestAvailableTime(startTime, duration, taskOffset, locationOffset int) int {
	resourceOffset := 0 // Assuming resource's schedule offset is its local offset

	// Convert start time to resource's timezone
	checkStart := startTime + (taskOffset - resourceOffset)
	checkEnd := checkStart + duration

	if res.isAvailable(checkStart, checkEnd) == [2]int{0, 0} {
		return checkStart - (taskOffset - resourceOffset) // Convert back to task's timezone
	}

	nextAvailable := checkEnd
	if res.isAvailable(nextAvailable, nextAvailable+duration) == [2]int{0, 0} {
		return nextAvailable - (taskOffset - resourceOffset) // Convert back to task's timezone
	}

	return -1 // Indicate no availability found
}

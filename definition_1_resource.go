package scheduler

import (
	"context"
	"errors"

	goerrors "github.com/TudorHulban/go-errors"
)

type Resource struct {
	Name string

	schedule        map[[3]int64]int64 // [3]int is [unix_start_time, unix_end_time, GMT offset]Task ID
	costPerLoadUnit map[uint8]float32  // load unit | cost per unit

	ID           int
	ResourceType uint8
}

type ParamsNewResource struct {
	Name            string
	CostPerLoadUnit map[uint8]float32
	ResourceType    uint8
}

func (param *ParamsNewResource) IsValid() error {
	if len(param.Name) == 0 {
		return goerrors.ErrValidation{
			Caller: "IsValid - ParamsNewResource",
			Issue: goerrors.ErrNilInput{
				InputName: "Name",
			},
		}
	}

	if param.ResourceType <= 0 {
		return goerrors.ErrValidation{
			Caller: "IsValid - ParamsNewResource",
			Issue: goerrors.ErrInvalidInput{
				InputName: "ResourceType",
			},
		}
	}

	if param.CostPerLoadUnit == nil {
		return goerrors.ErrValidation{
			Caller: "IsValid - ParamsNewResource",
			Issue: goerrors.ErrNilInput{
				InputName: "CostPerLoadUnit",
			},
		}
	}

	for _, cost := range param.CostPerLoadUnit {
		if cost < 0 {
			return goerrors.ErrValidation{
				Caller: "IsValid - ParamsNewResource",
				Issue: goerrors.ErrNegativeInput{
					InputName: "CostPerLoadUnit",
				},
			}
		}
	}

	return nil
}

func NewResource(params *ParamsNewResource) (*Resource, error) {
	if errValidation := params.IsValid(); errValidation != nil {
		return nil,
			errValidation
	}

	return &Resource{
			Name:         params.Name,
			ResourceType: params.ResourceType,

			costPerLoadUnit: params.CostPerLoadUnit,
			schedule:        make(map[[3]int64]int64),
		},
		nil
}

// isAvailable returns overlapStart, overlapEnd.
func (res *Resource) isAvailable(timeStart, timeEnd int64) [2]int64 {
	for interval := range res.schedule {
		scheduleStart := interval[0]
		scheduleEnd := interval[1]
		offset := interval[2]

		withOffsetScheduleStart := scheduleStart - offset
		withOffsetScheduleEnd := scheduleEnd - offset

		overlapStart := max(timeStart, withOffsetScheduleStart)
		overlapEnd := min(timeEnd, withOffsetScheduleEnd)

		if overlapStart < overlapEnd {
			return [2]int64{
				overlapStart,
				overlapEnd,
			}
		}
	}

	return [2]int64{}
}

type ParamsTask struct {
	TimeStart int64
	TimeEnd   int64
	GMTOffset int64

	TaskID int64
}

func (res *Resource) AddTask(_ context.Context, params *ParamsTask) ([2]int64, error) {
	if params.TimeStart >= params.TimeEnd {
		return [2]int64{},
			goerrors.ErrInvalidInput{
				Caller:     "AddTask",
				InputName:  "TimeEnd",
				InputValue: params.TimeEnd,
				Issue: errors.New(
					"time start greater or equal to time end",
				),
			}
	}

	if params.TaskID <= 0 {
	}

	overlap := res.isAvailable(params.TimeStart, params.TimeEnd)

	if overlap == [2]int64{} {
		res.schedule[[3]int64{
			params.TimeStart,
			params.TimeEnd,
			params.GMTOffset,
		}] = params.TaskID

		return [2]int64{},
			nil
	}

	return overlap,
		errors.New("busy")
}

type ResponseGetTask struct {
	TaskID      int64
	TaskEndTime int64
}

// GetTask returns scheduled task id and when estimated to finish if there is one scheduled.
func (res *Resource) GetTask(atTimestamp, offset int64) (*ResponseGetTask, error) {
	for interval, taskID := range res.schedule {
		scheduleStart := interval[0]
		scheduleEnd := interval[1]

		// Adjust the times to offset
		scheduleStartUTC := scheduleStart - interval[2]
		scheduleEndUTC := scheduleEnd - interval[2]
		atTimestampUTC := atTimestamp - offset

		if scheduleEndUTC >= atTimestampUTC && scheduleStartUTC <= atTimestampUTC {
			return &ResponseGetTask{
					TaskID:      taskID,
					TaskEndTime: scheduleEnd,
				},
				nil
		}
	}

	return nil,
		errors.New(
			"no task scheduled at given timestamp",
		)
}

func (res *Resource) RemoveTask(_ context.Context, params *ParamsTask) error {
	keysToDelete := make([][3]int64, 0)

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

type paramsFindEarliestAvailableTime struct {
	TimeStart      int64
	Duration       int64
	OffsetTask     int64
	OffsetLocation int64
}

func (res *Resource) findEarliestAvailableTime(params *paramsFindEarliestAvailableTime) int64 {
	checkStart := params.TimeStart + (params.OffsetTask - params.OffsetLocation)
	checkEnd := checkStart + params.Duration

	if res.isAvailable(checkStart, checkEnd) == [2]int64{} {
		return checkStart - (params.OffsetTask - params.OffsetLocation) // Convert back to task's timezone
	}

	nextAvailable := checkEnd
	if res.isAvailable(nextAvailable, nextAvailable+params.Duration) == [2]int64{} {
		return nextAvailable - (params.OffsetTask - params.OffsetLocation) // Convert back to task's timezone
	}

	return _NoAvailability
}

// Helper function to find the earliest available time slot on a resource from a given start time
func (res *Resource) findEarliestAvailableTimeFrom(params *paramsFindEarliestAvailableTime) int64 {
	checkStart := params.TimeStart + (params.OffsetTask - params.OffsetLocation)
	checkEnd := checkStart + params.Duration

	if res.isAvailable(checkStart, checkEnd) == [2]int64{} {
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
	nextPossibleEnd := nextPossibleStart + params.Duration

	if res.isAvailable(nextPossibleStart, nextPossibleEnd) == [2]int64{} {
		return nextPossibleStart
	}

	return _NoAvailability
}

package scheduler

import (
	"context"
	"errors"
	"sort"

	goerrors "github.com/TudorHulban/go-errors"
)

type Resource struct {
	Name string

	schedule        map[TimeInterval]int64 // TimeInterval | Task ID
	costPerLoadUnit map[uint8]float32      // load unit | cost per unit

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
			schedule:        make(map[TimeInterval]int64),
		},
		nil
}

func (res *Resource) GetAvailability(searchInterval *TimeInterval) []TimeInterval {
	var busyUTCIntervals []TimeInterval

	for scheduledInterval := range res.schedule {
		busyUTCIntervals = append(
			busyUTCIntervals,
			TimeInterval{
				TimeStart: scheduledInterval.GetUTCTimeStart(),
				TimeEnd:   scheduledInterval.GetUTCTimeEnd(),
			},
		)
	}

	// Sort busy intervals by start time
	sort.Slice(
		busyUTCIntervals,
		func(i, j int) bool {
			return busyUTCIntervals[i].TimeStart < busyUTCIntervals[j].TimeStart
		},
	)

	var availableIntervals []TimeInterval

	currentStart := searchInterval.GetUTCTimeStart()
	searchEnd := searchInterval.GetUTCTimeEnd()

	for _, busy := range busyUTCIntervals {
		// Skip busy intervals that don't overlap with our search window
		if busy.TimeEnd <= currentStart {
			continue
		}

		// Busy interval is completely after our search window
		if busy.TimeStart >= searchEnd {
			break
		}

		// If there's a gap before this busy interval, add it as available
		if busy.TimeStart > currentStart {
			availableIntervals = append(
				availableIntervals,
				TimeInterval{
					TimeStart: currentStart,
					TimeEnd:   busy.TimeStart,
					Offset:    searchInterval.Offset, // Maintain original offset
				},
			)
		}

		// Move currentStart to the end of this busy interval
		currentStart = max(currentStart, busy.TimeEnd)
	}

	// Add remaining time after last busy interval
	if currentStart < searchEnd {
		availableIntervals = append(
			availableIntervals,
			TimeInterval{
				TimeStart: currentStart,
				TimeEnd:   searchEnd,
				Offset:    searchInterval.Offset,
			},
		)
	}

	return availableIntervals
}

type ParamsTask struct {
	TimeInterval

	TaskID int64
}

func (res *Resource) AddTask(_ context.Context, params *ParamsTask) (*Availability, error) {
	if params.TimeStart >= params.TimeEnd {
		return nil,
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

	overlap := res.GetAvailability(&params.TimeInterval)

	if overlap == nil {
		res.schedule[TimeInterval{
			TimeStart: params.TimeStart,
			TimeEnd:   params.TimeEnd,
			Offset:    params.Offset,
		}] = params.TaskID

		return nil,
			nil
	}

	return overlap,
		errors.New("busy")
}

type ResponseGetTask struct {
	TaskID                      int64
	AlreadyScheduledTaskEndTime int64
}

func (res *Resource) GetTask(atTimestamp, offset int64) (*ResponseGetTask, error) {
	for interval, taskID := range res.schedule {
		offsetDifference := interval.Offset - offset

		scheduleStartUTC := interval.TimeStart + offsetDifference
		scheduleEndUTC := interval.TimeEnd + offsetDifference
		atTimestampUTC := atTimestamp + offsetDifference

		if scheduleEndUTC >= atTimestampUTC && scheduleStartUTC <= atTimestampUTC {
			return &ResponseGetTask{
					TaskID:                      taskID,
					AlreadyScheduledTaskEndTime: scheduleEndUTC,
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
	keysToDelete := make([]TimeInterval, 0)

	for interval, taskID := range res.schedule {
		if taskID == params.TaskID {
			offsetDifference := interval.Offset - params.Offset

			if params.TimeStart+offsetDifference <= interval.TimeStart &&
				interval.TimeEnd+offsetDifference <= params.TimeEnd &&
				interval.Offset == params.Offset {
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

	offsetDifference := params.OffsetTask - params.OffsetLocation

	if res.GetAvailability(
		&TimeInterval{
			TimeStart: checkStart,
			TimeEnd:   checkEnd,
			Offset:    offsetDifference,
		},
	) == nil {
		return checkStart - offsetDifference
	}

	// TODO: here it should be a loop, to check also for next task until we find availability.
	nextAvailable := checkEnd

	if res.GetAvailability(
		&TimeInterval{
			TimeStart: nextAvailable,
			TimeEnd:   nextAvailable + params.Duration,
		},
	) == nil {
		return nextAvailable - offsetDifference
	}

	return _NoAvailability
}

// Helper function to find the earliest available time slot on a resource from a given start time
func (res *Resource) findEarliestAvailableTimeFrom(params *paramsFindEarliestAvailableTime) int64 {
	checkStart := params.TimeStart + (params.OffsetTask - params.OffsetLocation)
	checkEnd := checkStart + params.Duration

	if res.GetAvailability(
		&TimeInterval{
			TimeStart: checkStart,
			TimeEnd:   checkEnd,
		},
	) == nil {
		return checkStart
	}

	// In a real scenario, you'd need to look ahead in the schedule more comprehensively
	// This is a very basic placeholder looking at the next potential slot
	latestEndTime := checkStart

	for interval := range res.schedule {
		scheduleEnd := interval.GetUTCTimeEnd()

		if scheduleEnd > latestEndTime {
			latestEndTime = scheduleEnd
		}
	}

	nextPossibleStart := latestEndTime
	nextPossibleEnd := nextPossibleStart + params.Duration

	if res.GetAvailability(
		&TimeInterval{
			TimeStart: nextPossibleStart,
			TimeEnd:   nextPossibleEnd,
		},
	) == nil {
		return nextPossibleStart
	}

	return _NoAvailability
}

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

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

func (res *Resource) GetSchedule() string {
	if len(res.schedule) == 0 {
		return "Schedule: (empty)"
	}

	// Extract and sort intervals
	intervals := make([]TimeInterval, 0, len(res.schedule))
	for interval := range res.schedule {
		intervals = append(intervals, interval)
	}

	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].TimeStart < intervals[j].TimeStart
	})

	var sb strings.Builder
	sb.WriteString("Schedule:\n")

	for _, interval := range intervals {
		taskID := res.schedule[interval]

		sb.WriteString(
			fmt.Sprintf(
				"- [%d-%d] (UTC %d-%d) Offset %.1fh → Task %d\n",

				interval.TimeStart,
				interval.TimeEnd,
				interval.GetUTCTimeStart(),
				interval.GetUTCTimeEnd(),
				float64(interval.SecondsOffset)/3600,
				taskID,
			),
		)
	}

	return sb.String()
}

// GetAvailability returns:
//   - (nil, true)   = Fully available (no busy intervals or no overlap)
//   - (slots, false) = Partially available (returns available time slots)
//   - (nil, false)  = Completely unavailable (requested interval is fully booked)
func (res *Resource) GetAvailability(searchInterval *TimeInterval) ([]TimeInterval, bool) {
	var busyUTCIntervals []TimeInterval

	for scheduledInterval := range res.schedule {
		busyUTCIntervals = append(
			busyUTCIntervals, TimeInterval{
				TimeStart:     scheduledInterval.GetUTCTimeStart(),
				TimeEnd:       scheduledInterval.GetUTCTimeEnd(),
				SecondsOffset: scheduledInterval.SecondsOffset,
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

	fullyAvailable := len(busyUTCIntervals) == 0 // Start optimistic if no busy intervals

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
				availableIntervals, TimeInterval{
					TimeStart:     currentStart + searchInterval.SecondsOffset,
					TimeEnd:       busy.TimeStart + searchInterval.SecondsOffset,
					SecondsOffset: searchInterval.SecondsOffset,
				},
			)

			fullyAvailable = false
		} else if busy.TimeStart < currentStart {
			fullyAvailable = false
		}

		// Move currentStart to the end of this busy interval
		currentStart = max(currentStart, busy.TimeEnd)
	}

	// Add remaining time after last busy interval
	if currentStart < searchEnd {
		availableIntervals = append(
			availableIntervals,
			TimeInterval{
				TimeStart:     currentStart + searchInterval.SecondsOffset,
				TimeEnd:       searchEnd + searchInterval.SecondsOffset,
				SecondsOffset: searchInterval.SecondsOffset,
			},
		)
	} else if len(busyUTCIntervals) > 0 {
		fullyAvailable = false
	}

	if fullyAvailable {
		return nil,
			true
	}

	return availableIntervals,
		false
}

type ParamsTask struct {
	TimeInterval

	TaskID int64
}

func (res *Resource) AddTask(_ context.Context, params *ParamsTask) ([]TimeInterval, error) {
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

	overlaps, hasAvailability := res.GetAvailability(&params.TimeInterval)

	if hasAvailability {
		res.schedule[TimeInterval{
			TimeStart:     params.TimeStart,
			TimeEnd:       params.TimeEnd,
			SecondsOffset: params.SecondsOffset,
		}] = params.TaskID

		return overlaps,
			nil
	}

	return overlaps,
		errors.New("busy")
}

type ResponseGetTask struct {
	TaskID                      int64
	AlreadyScheduledTaskEndTime int64
}

func (res *Resource) GetTask(atTimestamp, offset int64) (*ResponseGetTask, error) {
	for interval, taskID := range res.schedule {
		offsetDifference := interval.SecondsOffset - offset

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
			offsetDifference := interval.SecondsOffset - params.SecondsOffset

			if params.TimeStart+offsetDifference <= interval.TimeStart &&
				interval.TimeEnd+offsetDifference <= params.TimeEnd &&
				interval.SecondsOffset == params.SecondsOffset {
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

// func (res *Resource) findEarliestAvailableTime(params *paramsFindEarliestAvailableTime) int64 {
// 	checkStart := params.TimeStart + (params.OffsetTask - params.OffsetLocation)
// 	checkEnd := checkStart + params.Duration

// 	offsetDifference := params.OffsetTask - params.OffsetLocation

// 	if res.GetAvailability(
// 		&TimeInterval{
// 			TimeStart: checkStart,
// 			TimeEnd:   checkEnd,
// 			Offset:    offsetDifference,
// 		},
// 	) == nil {
// 		return checkStart - offsetDifference
// 	}

// 	// TODO: here it should be a loop, to check also for next task until we find availability.
// 	nextAvailable := checkEnd

// 	if res.GetAvailability(
// 		&TimeInterval{
// 			TimeStart: nextAvailable,
// 			TimeEnd:   nextAvailable + params.Duration,
// 		},
// 	) == nil {
// 		return nextAvailable - offsetDifference
// 	}

// 	return _NoAvailability
// }

// Helper function to find the earliest available time slot on a resource from a given start time
// func (res *Resource) findEarliestAvailableTimeFrom(params *paramsFindEarliestAvailableTime) int64 {
// 	checkStart := params.TimeStart + (params.OffsetTask - params.OffsetLocation)
// 	checkEnd := checkStart + params.Duration

// 	if res.GetAvailability(
// 		&TimeInterval{
// 			TimeStart: checkStart,
// 			TimeEnd:   checkEnd,
// 		},
// 	) == nil {
// 		return checkStart
// 	}

// 	// In a real scenario, you'd need to look ahead in the schedule more comprehensively
// 	// This is a very basic placeholder looking at the next potential slot
// 	latestEndTime := checkStart

// 	for interval := range res.schedule {
// 		scheduleEnd := interval.GetUTCTimeEnd()

// 		if scheduleEnd > latestEndTime {
// 			latestEndTime = scheduleEnd
// 		}
// 	}

// 	nextPossibleStart := latestEndTime
// 	nextPossibleEnd := nextPossibleStart + params.Duration

// 	if res.GetAvailability(
// 		&TimeInterval{
// 			TimeStart: nextPossibleStart,
// 			TimeEnd:   nextPossibleEnd,
// 		},
// 	) == nil {
// 		return nextPossibleStart
// 	}

// 	return _NoAvailability
// }

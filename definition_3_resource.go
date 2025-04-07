package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	goerrors "github.com/TudorHulban/go-errors"
)

type RunID int64

const Maintenance = RunID(0)

type Resource struct {
	Name string

	schedule        map[TimeInterval]RunID
	costPerLoadUnit map[uint8]float32 // load unit | cost per unit

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
			schedule:        make(map[TimeInterval]RunID),
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

	sort.Slice(
		intervals,
		func(i, j int) bool {
			return intervals[i].TimeStart < intervals[j].TimeStart
		},
	)

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
		busyUTCIntervals = append(busyUTCIntervals, TimeInterval{
			TimeStart:     scheduledInterval.GetUTCTimeStart(),
			TimeEnd:       scheduledInterval.GetUTCTimeEnd(),
			SecondsOffset: scheduledInterval.SecondsOffset,
		})
	}

	sort.Slice(
		busyUTCIntervals,
		func(i, j int) bool {
			return busyUTCIntervals[i].TimeStart < busyUTCIntervals[j].TimeStart
		},
	)

	var availableIntervals []TimeInterval

	currentStart := searchInterval.GetUTCTimeStart()
	searchEnd := searchInterval.GetUTCTimeEnd()

	// Check if any busy interval overlaps
	hasOverlap := false

	for _, busy := range busyUTCIntervals {
		if busy.TimeEnd <= currentStart {
			continue
		}
		if busy.TimeStart >= searchEnd {
			break
		}

		hasOverlap = true

		if busy.TimeStart > currentStart {
			availableIntervals = append(
				availableIntervals,
				TimeInterval{
					TimeStart:     currentStart + searchInterval.SecondsOffset,
					TimeEnd:       busy.TimeStart + searchInterval.SecondsOffset,
					SecondsOffset: searchInterval.SecondsOffset,
				},
			)
		}

		currentStart = max(currentStart, busy.TimeEnd)
	}

	if currentStart < searchEnd {
		availableIntervals = append(
			availableIntervals,
			TimeInterval{
				TimeStart:     currentStart + searchInterval.SecondsOffset,
				TimeEnd:       searchEnd + searchInterval.SecondsOffset,
				SecondsOffset: searchInterval.SecondsOffset,
			},
		)
	}

	if !hasOverlap {
		return nil, true // Fully available if no overlap
	}

	return availableIntervals, false
}

type ParamsRun struct {
	TimeInterval

	ID RunID
}

func (res *Resource) AddRun(_ context.Context, params *ParamsRun) ([]TimeInterval, error) {
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

	if params.ID <= 0 {
		return nil, goerrors.ErrInvalidInput{
			Caller:     "AddRun",
			InputName:  "ID",
			InputValue: params.ID,
			Issue: goerrors.ErrNegativeInput{
				InputName: "ID",
			},
		}
	}

	for interval := range res.schedule {
		if res.schedule[interval] == params.ID {
			return nil,
				fmt.Errorf(
					"run ID %d already exists",
					params.ID,
				)
		}
	}

	overlaps, available := res.GetAvailability(&params.TimeInterval)
	if !available {
		return overlaps,
			errors.New("requested time slot is busy")
	}

	// Add the run
	res.schedule[TimeInterval{
		TimeStart:     params.TimeStart,
		TimeEnd:       params.TimeEnd,
		SecondsOffset: params.SecondsOffset,
	}] = params.ID

	return nil, nil
}

type ResponseGetRun struct {
	ID                          RunID
	AlreadyScheduledTaskEndTime int64
}

func (res *Resource) GetRun(atTimestamp, offset int64) (*ResponseGetRun, error) {
	for interval, runID := range res.schedule {
		offsetDifference := interval.SecondsOffset - offset

		scheduleStartUTC := interval.TimeStart + offsetDifference
		scheduleEndUTC := interval.TimeEnd + offsetDifference
		atTimestampUTC := atTimestamp + offsetDifference

		if scheduleEndUTC >= atTimestampUTC && scheduleStartUTC <= atTimestampUTC {
			return &ResponseGetRun{
					ID:                          runID,
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

func (res *Resource) RemoveRun(runID RunID) error {
	for interval, id := range res.schedule {
		if id == runID {
			delete(res.schedule, interval)

			return nil
		}
	}

	return fmt.Errorf("run %d not found in schedule", runID)
}

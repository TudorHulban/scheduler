package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	goerrors "github.com/TudorHulban/go-errors"
)

type RunID int64

const Maintenance = RunID(0)

type ResourceType uint8

type ResourceInfo struct {
	Name            string
	CostPerLoadUnit map[uint8]float32 // load unit | cost per unit
	ID              int
	ResourceType    uint8
	ServedQuantity  uint16 // ex. apartment w 2 rooms serves 2, room serves 1
}

func (r ResourceInfo) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ID: %d,", r.ID))
	sb.WriteString(fmt.Sprintf("Name: %q,", r.Name))
	sb.WriteString(fmt.Sprintf("ResourceType: %d", r.ResourceType))

	return sb.String()
}

// ResourceScheduled is mutex protected through Location ops.
type ResourceScheduled struct {
	ResourceInfo

	mu sync.RWMutex

	schedule map[TimeInterval]RunID
}

type ParamsNewResource struct {
	Name            string
	CostPerLoadUnit map[uint8]float32
	ID              int
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

func NewResource(params *ParamsNewResource) (*ResourceScheduled, error) {
	if errValidation := params.IsValid(); errValidation != nil {
		return nil,
			errValidation
	}

	return &ResourceScheduled{
			ResourceInfo: ResourceInfo{
				ID:           params.ID,
				Name:         params.Name,
				ResourceType: params.ResourceType,

				CostPerLoadUnit: params.CostPerLoadUnit,
			},

			schedule: make(map[TimeInterval]RunID),
		},
		nil
}

func (res *ResourceScheduled) GetSchedule() string {
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

type ParamsRun struct {
	TimeInterval

	ID RunID
}

// ID = 0 reserved for Maintenance.
func (res *ResourceScheduled) AddRun(_ context.Context, params *ParamsRun) ([]TimeInterval, error) {
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

func (res *ResourceScheduled) GetRun(atTimestamp, offset int64) (*ResponseGetRun, error) {
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

// removeRun should be called through Location which is mutex protected.
func (res *ResourceScheduled) removeRun(runID RunID) error {
	for interval, id := range res.schedule {
		if id == runID {
			delete(res.schedule, interval)

			return nil
		}
	}

	return fmt.Errorf("run %d not found in schedule", runID)
}

func (res *ResourceScheduled) GetRunCost(run *Run) (float32, error) {
	cost, exists := res.CostPerLoadUnit[run.LoadUnit]
	if !exists {
		return 0,
			errors.New("unsupported load unit")
	}

	return cost * run.Load,
		nil
}

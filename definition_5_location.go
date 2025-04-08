package scheduler

import (
	"slices"

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

// CanSchedule returns zero for WhenCanStart if it can run within passed interval and
// also schedules the task to the cheapest available resource and provides the cost.
//
// If it cannot run within interval, it provides the timestamp
// from which it could in WhenCanStart and the cost of this run.
func (loc *Location) CanSchedule(params *ParamsCanRun) (*ResponseCanRun, error) {
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

	for _, candidate := range loc.Resources {
		if slices.Contains(resourceTypesNeeded, candidate.ResourceType) {
			resourceTypeCandidates[candidate.ResourceType] = append(resourceTypeCandidates[candidate.ResourceType], candidate)
		}
	}

	offsetDifference := params.SecondsOffset - loc.LocationOffset
	start := params.TimeStart + offsetDifference
	end := params.TimeEnd + offsetDifference

	possibilities := populatePossibilities(
		&paramsPopulatePossibilities{
			Candidates: resourceTypeCandidates,
			TimeInterval: TimeInterval{
				TimeStart:     start,
				TimeEnd:       end,
				SecondsOffset: loc.LocationOffset,
			},
			Duration: params.TaskRun.EstimatedDuration,
		},
	)

	earliest, selectedResources := findEarliestSlot(possibilities, len(resourceTypesNeeded), offsetDifference)

	var totalCost float32
	if earliest != _NoAvailability {
		for _, resource := range selectedResources {
			cost, _ := calculateTaskCost(params.TaskRun, resource)
			totalCost = totalCost + cost
		}

		if earliest == params.TimeStart {
			for _, resource := range selectedResources {
				resource.schedule[TimeInterval{
					TimeStart:     earliest + (params.SecondsOffset - loc.LocationOffset),
					TimeEnd:       earliest + params.TaskRun.EstimatedDuration + (params.SecondsOffset - loc.LocationOffset),
					SecondsOffset: loc.LocationOffset,
				}] = RunID(params.TaskRun.ID)
			}

			return &ResponseCanRun{
					WhenCanStart: _ScheduledForStart,
					Cost:         totalCost,
					WasScheduled: earliest == params.TimeStart,
				},
				nil
		}

		return &ResponseCanRun{
				WhenCanStart: earliest,
				Cost:         totalCost,
				WasScheduled: earliest == params.TimeStart,
			},
			nil
	}

	earliestFallback := _NoAvailability
	var bestFallbackRes *Resource

	for _, res := range loc.Resources {
		if slices.Contains(resourceTypesNeeded, res.ResourceType) {
			when := res.findAvailableTime(&paramsFindAvailableTime{
				TimeStart:             start,
				MaximumTimeStart:      end + 3600,
				SecondsDuration:       params.TaskRun.EstimatedDuration,
				SecondsOffsetTask:     params.SecondsOffset,
				SecondsOffsetLocation: loc.LocationOffset,
				IsLatest:              false,
			})

			if when != _NoAvailability {
				whenTaskTime := when - offsetDifference
				if earliestFallback == _NoAvailability || whenTaskTime < earliestFallback {
					earliestFallback = whenTaskTime
					bestFallbackRes = res
				}
			}
		}
	}

	if earliestFallback != _NoAvailability {
		cost, _ := calculateTaskCost(params.TaskRun, bestFallbackRes)
		totalCost = cost

		return &ResponseCanRun{
				WhenCanStart: earliestFallback - params.TimeStart,
				Cost:         totalCost,
				WasScheduled: true,
			},
			nil
	}

	return &ResponseCanRun{
			WhenCanStart: params.TimeEnd,
			Cost:         totalCost,
			WasScheduled: false,
		},
		nil
}

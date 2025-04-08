package scheduler

import (
	"fmt"
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

// CanSchedule returns zero for WhenCanStart if it can run within passed nterval and
// also schedules the task to the cheapest available resource and provides the cost.
//
// If it cannot run within interval, it provides the timestamp
// from which it could in WhenCanStart and the cost of this run.
func (loc *Location) CanSchedule(params *ParamsCanRun) (*ResponseCanRun, error) {
	if params.TimeEnd-params.TimeStart < params.TaskRun.EstimatedDuration {
		return &ResponseCanRun{
			WhenCanStart: params.TimeEnd,
			Cost:         0,
			WasScheduled: false,
		}, nil
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

		for _, candidate := range candidates {
			targetSlot := TimeInterval{
				TimeStart:     start,
				TimeEnd:       end,
				SecondsOffset: loc.LocationOffset,
			}

			availSlots, available := candidate.GetAvailability(&targetSlot)
			if available {
				possibilities[targetSlot] = append(possibilities[targetSlot], candidate)
			} else {
				for _, slot := range availSlots {
					if slot.TimeEnd-slot.TimeStart >= params.TaskRun.EstimatedDuration {
						possibilities[slot] = append(possibilities[slot], candidate)
					}
				}
			}
		}
	}

	earliest, bestResources := findEarliestSlot(possibilities, len(resourceTypesNeeded), offsetDifference)

	fmt.Println(
		len(bestResources),
		bestResources[0].ID,
		possibilities,
	)

	var totalCost float32
	if earliest != _NoAvailability {
		for _, resource := range bestResources {
			cost, _ := calculateTaskCost(params.TaskRun, resource)
			totalCost = totalCost + cost
		}
		if earliest == params.TimeStart {
			for _, resource := range bestResources {
				resource.schedule[TimeInterval{
					TimeStart:     earliest + (params.SecondsOffset - loc.LocationOffset),
					TimeEnd:       earliest + params.TaskRun.EstimatedDuration + (params.SecondsOffset - loc.LocationOffset),
					SecondsOffset: loc.LocationOffset,
				}] = RunID(params.TaskRun.ID)
			}
		}
		return &ResponseCanRun{
			WhenCanStart: earliest - params.TimeStart,
			Cost:         totalCost,
			WasScheduled: earliest == params.TimeStart,
		}, nil
	}

	earliestFallback := int64(_NoAvailability)
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
			WasScheduled: false,
		}, nil
	}

	return &ResponseCanRun{
		WhenCanStart: params.TimeEnd,
		Cost:         totalCost,
		WasScheduled: false,
	}, nil
}

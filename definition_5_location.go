package scheduler

import (
	"fmt"
	"slices"
	"strings"

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
				Caller:      "NewLocation",
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

func (p ParamsCanRun) String() string {
	var sb strings.Builder

	sb.WriteString("ParamsCanRun{\n")

	// TimeInterval fields
	sb.WriteString("\tTimeInterval{\n")
	sb.WriteString(fmt.Sprintf("\t\tTimeStart: %d,\n", p.TimeStart))
	sb.WriteString(fmt.Sprintf("\t\tTimeEnd: %d,\n", p.TimeEnd))
	sb.WriteString(fmt.Sprintf("\t\tSecondsOffset: %d,\n", p.SecondsOffset))
	sb.WriteString("\t},\n")

	// TaskRun
	if p.TaskRun != nil {
		sb.WriteString("\tTaskRun: &Run{\n")
		sb.WriteString(fmt.Sprintf("\t\tName: %q,\n", p.TaskRun.Name))

		// Dependencies
		if len(p.TaskRun.Dependencies) > 0 {
			sb.WriteString("\t\tDependencies: []RunDependency{\n")
			for _, dep := range p.TaskRun.Dependencies {
				sb.WriteString("\t\t\t{\n")
				sb.WriteString(fmt.Sprintf("\t\t\t\tPreferredResourceID: %d,\n", dep.PreferredResourceID))
				sb.WriteString(fmt.Sprintf("\t\t\t\tResourceType: %d,\n", dep.ResourceType))
				sb.WriteString(fmt.Sprintf("\t\t\t\tResourceQuantity: %d,\n", dep.ResourceQuantity))
				sb.WriteString("\t\t\t},\n")
			}
			sb.WriteString("\t\t},\n")
		} else {
			sb.WriteString("\t\tDependencies: nil,\n")
		}

		// RunLoad fields
		sb.WriteString("\t\tRunLoad: {\n")
		sb.WriteString(fmt.Sprintf("\t\t\tLoad: %f,\n", p.TaskRun.RunLoad.Load))
		sb.WriteString(fmt.Sprintf("\t\t\tLoadUnit: %d,\n", p.TaskRun.RunLoad.LoadUnit))
		sb.WriteString("\t\t},\n")

		// Other Run fields
		sb.WriteString(fmt.Sprintf("\t\tID: %d,\n", p.TaskRun.ID))
		sb.WriteString(fmt.Sprintf("\t\tInitiatorID: %d,\n", p.TaskRun.InitiatorID))
		sb.WriteString(fmt.Sprintf("\t\tEstimatedDuration: %d,\n", p.TaskRun.EstimatedDuration))
		sb.WriteString("\t},\n")
	} else {
		sb.WriteString("\tTaskRun: nil,\n")
	}

	sb.WriteString("}")

	return sb.String()
}

type ResponseCanRun struct {
	WhenCanStart int64
	Cost         float32
	WasScheduled bool
}

// CanSchedule returns zero for WhenCanStart if it can run within passed interval and
// also schedules the task to the cheapest available resource and provides the cost.
//
// If it cannot run at TimeStart, it provides the timestamp
// from which it could in WhenCanStart and the cost of this run.
func (loc *Location) CanSchedule(params *ParamsCanRun) (*ResponseCanRun, error) {
	defer traceExitWMarker(params.String())

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
	resourcesNeededPerType := params.TaskRun.GetNeededResourcesPerType()

	for _, candidate := range loc.Resources {
		if slices.Contains(resourceTypesNeeded, candidate.ResourceType) {
			resourceTypeCandidates[candidate.ResourceType] = append(
				resourceTypeCandidates[candidate.ResourceType],
				candidate,
			)
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

	noNeeded := resourcesNeededPerType[1] // Simplify - only one type of resources for now.

	earliest, selectedResources := findEarliestSlot(
		possibilities,
		int(noNeeded),
		offsetDifference,
	)

	var totalCost float32

	if earliest != _NoAvailability {
		for _, resource := range selectedResources {
			cost, _ := calculateTaskCost(params.TaskRun, resource)
			totalCost = totalCost + cost
		}

		if earliest == params.TimeStart {
			for _, resource := range selectedResources {
				resource.schedule[TimeInterval{
					TimeStart:     earliest + offsetDifference,
					TimeEnd:       earliest + params.TaskRun.EstimatedDuration + offsetDifference,
					SecondsOffset: loc.LocationOffset,
				}] = RunID(params.TaskRun.ID)
			}

			return &ResponseCanRun{
					WhenCanStart: _ScheduledForStart,
					Cost:         totalCost,
					WasScheduled: true,
				},
				nil
		}

		return &ResponseCanRun{
				WhenCanStart: earliest,
				Cost:         totalCost,
				WasScheduled: false,
			},
			nil
	}

	earliestFallback := _NoAvailability
	var fallbackResources []*Resource
	totalCost = 0
	needed := int(resourcesNeededPerType[1]) // Assuming type 1

	for _, res := range loc.Resources {
		if slices.Contains(resourceTypesNeeded, res.ResourceType) && len(fallbackResources) < needed {
			when := res.findAvailableTime(
				&paramsFindAvailableTime{
					TimeStart:             start,
					MaximumTimeStart:      end + params.TaskRun.EstimatedDuration,
					SecondsDuration:       params.TaskRun.EstimatedDuration,
					SecondsOffsetTask:     params.SecondsOffset,
					SecondsOffsetLocation: loc.LocationOffset,
				},
			)

			if when != _NoAvailability {
				whenTaskTime := when - offsetDifference
				if earliestFallback == _NoAvailability || whenTaskTime < earliestFallback {
					earliestFallback = whenTaskTime
					fallbackResources = []*Resource{res} // Reset to earliest

					cost, _ := calculateTaskCost(params.TaskRun, res)
					totalCost = cost
				} else if whenTaskTime == earliestFallback {
					fallbackResources = append(fallbackResources, res)

					cost, _ := calculateTaskCost(params.TaskRun, res)
					totalCost = totalCost + cost
				}
			}
		}
	}

	if earliestFallback != _NoAvailability && len(fallbackResources) == needed {
		return &ResponseCanRun{
				WhenCanStart: earliestFallback - params.TimeStart,
				Cost:         totalCost,
				WasScheduled: earliestFallback == params.TimeStart,
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

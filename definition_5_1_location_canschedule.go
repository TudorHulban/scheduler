package scheduler

import (
	"slices"

	goerrors "github.com/TudorHulban/go-errors"
)

type ResponseGetPossibilities struct {
	Possibilities ResourcesPerTimeInterval

	resourceTypesNeeded    []uint8
	resourcesNeededPerType map[uint8]uint16

	offsetedTimeInterval TimeInterval
}

// GetPossibilities returns all possible time slots when resources are available
func (loc *Location) GetPossibilities(params *ParamsCanRun) (*ResponseGetPossibilities, error) {
	if params.TimeEnd-params.TimeStart < params.TaskRun.EstimatedDuration {
		return nil,
			goerrors.ErrValidation{
				Caller: "GetPossibilities",
				Issue: goerrors.ErrInvalidInput{
					InputName: "ParamsCanRun - interval too short",
				},
			}
	}

	resourceTypeCandidates := make(map[uint8][]*ResourceScheduled)
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

	offsetedTimeInterval := TimeInterval{
		TimeStart:     params.TimeStart + offsetDifference,
		TimeEnd:       params.TimeEnd + offsetDifference,
		SecondsOffset: offsetDifference,
	}

	possibilities := populatePossibilities(
		&paramsPopulatePossibilities{
			Candidates:             resourceTypeCandidates,
			ResourcesNeededPerType: resourcesNeededPerType,
			TimeInterval:           offsetedTimeInterval,

			Duration: params.TaskRun.EstimatedDuration,
		},
	)

	return &ResponseGetPossibilities{
			Possibilities: possibilities,

			resourceTypesNeeded:    resourceTypesNeeded,
			resourcesNeededPerType: resourcesNeededPerType,
			offsetedTimeInterval:   offsetedTimeInterval,
		},
		nil
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
	possibilitiesResp, errGetPossibilities := loc.GetPossibilities(params)
	if errGetPossibilities != nil {
		return nil,
			errGetPossibilities
	}

	// Find the best slot using the standard algorithm
	result, errSchedulingOptions := loc.findBestSchedulingOption(possibilitiesResp, params)
	if errSchedulingOptions != nil {
		return nil,
			errSchedulingOptions
	}

	timeStart := params.TimeStart + possibilitiesResp.offsetedTimeInterval.SecondsOffset
	targetTimeInterval := TimeInterval{
		TimeStart:     timeStart,
		TimeEnd:       timeStart + params.TaskRun.EstimatedDuration,
		SecondsOffset: possibilitiesResp.offsetedTimeInterval.SecondsOffset,
	}

	// If we found a viable option with the standard algorithm
	if result.WhenCanStart != _NoAvailability {
		// Schedule if needed and return
		if result.WhenCanStart == params.TimeStart {
			loc.scheduleResources(
				&paramsScheduleResources{
					Resources:    result.SelectedResources,
					TaskRunID:    RunID(params.TaskRun.ID),
					TimeInterval: targetTimeInterval,
				},
			)

			return &ResponseCanRun{
					WhenCanStart: _ScheduledForStart,
					Cost:         result.Cost,
					WasScheduled: true,
				},
				nil
		}

		return &ResponseCanRun{
				WhenCanStart: result.WhenCanStart,
				Cost:         result.Cost,
				WasScheduled: false,
			},
			nil
	}

	// Fallback algorithm when standard approach fails
	fallbackResult := loc.findFallbackOption(possibilitiesResp, params)

	// If fallback algorithm found an option and it's for immediate scheduling
	if fallbackResult.WhenCanStart != _NoAvailability {
		if fallbackResult.WhenCanStart == params.TimeStart {
			loc.scheduleResources(
				&paramsScheduleResources{
					Resources:    fallbackResult.SelectedResources,
					TaskRunID:    RunID(params.TaskRun.ID),
					TimeInterval: targetTimeInterval,
				},
			)
		}

		return &ResponseCanRun{
				WhenCanStart: fallbackResult.WhenCanStart,
				Cost:         fallbackResult.Cost,
				WasScheduled: fallbackResult.WhenCanStart == params.TimeStart,
			},
			nil
	}

	// No viable options found
	return &ResponseCanRun{
			WhenCanStart: params.TimeEnd,
			Cost:         0,
			WasScheduled: false,
		},
		nil
}

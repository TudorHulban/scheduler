package scheduler

import (
	"math"
	"sort"
)

type paramsFindEarliestSlot struct {
	Possibilities    ResourcesPerTimeInterval
	NeededCount      int
	OffsetDifference int64
}

func findEarliestSlot(params *paramsFindEarliestSlot) (int64, []*ResourceScheduled) {
	earliest := int64(_NoAvailability)
	bestResources := make([]*ResourceScheduled, 0)
	bestCost := float32(math.MaxFloat32)

	for slot, resources := range params.Possibilities {
		if len(resources) >= params.NeededCount { // Ensure total quantity across types
			start := slot.TimeStart - params.OffsetDifference
			var totalCost float32

			for _, res := range resources[:params.NeededCount] { // Take only needed
				cost := res.CostPerLoadUnit[1] // Assuming LoadUnit 1

				totalCost += cost
			}

			if totalCost < bestCost || (totalCost == bestCost && (earliest == _NoAvailability || start < earliest)) {
				bestCost = totalCost
				earliest = start

				sort.Slice(
					resources,
					func(i, j int) bool {
						return resources[i].CostPerLoadUnit[1] < resources[j].CostPerLoadUnit[1]
					},
				)

				bestResources = resources[:params.NeededCount]
			}
		}
	}

	return earliest,
		bestResources
}

type paramsPopulatePossibilities struct {
	Candidates             map[uint8][]*ResourceScheduled
	ResourcesNeededPerType map[uint8]uint16

	TimeInterval

	Duration int64
}

func populatePossibilities(params *paramsPopulatePossibilities) ResourcesPerTimeInterval {
	result := make(ResourcesPerTimeInterval)
	typeSlots := make(map[TimeInterval]ResourcesPerType)

	for resourceType, candidates := range params.Candidates {
		for _, candidate := range candidates {
			availSlots, availableEntireInterval := candidate.GetAvailability(&params.TimeInterval)

			if availableEntireInterval {
				noIntervals := params.TimeInterval.NoIntervals(params.Duration)
				for i := 0; i < noIntervals; i++ {
					timeStart := params.TimeStart + int64(i)*params.Duration
					slot := TimeInterval{
						TimeStart:     timeStart,
						TimeEnd:       timeStart + params.Duration,
						SecondsOffset: params.TimeInterval.SecondsOffset,
					}
					current := typeSlots[slot]
					if current == nil {
						current = make(ResourcesPerType)
						typeSlots[slot] = current
					}
					current[resourceType] = append(current[resourceType], candidate)
				}
			} else {
				for _, slot := range availSlots {
					exactSlots := slot.BreakDown(params.Duration)
					for _, exactSlot := range exactSlots {
						normalizedSlot := TimeInterval{
							TimeStart:     exactSlot.TimeStart,
							TimeEnd:       exactSlot.TimeEnd,
							SecondsOffset: params.TimeInterval.SecondsOffset,
						}
						current := typeSlots[normalizedSlot]
						if current == nil {
							current = make(ResourcesPerType)
							typeSlots[normalizedSlot] = current
						}
						current[resourceType] = append(current[resourceType], candidate)
					}
				}
			}
		}
	}

	// Filter slots meeting all type-quantity requirements
	for slot, resourcesByType := range typeSlots {
		allSatisfied := true
		var slotResources []*ResourceScheduled
		for resourceType, needed := range params.ResourcesNeededPerType {
			if len(resourcesByType[resourceType]) < int(needed) {
				allSatisfied = false
				break
			}
			sort.Slice(
				resourcesByType[resourceType],
				func(i, j int) bool {
					return resourcesByType[resourceType][i].CostPerLoadUnit[1] <
						resourcesByType[resourceType][j].CostPerLoadUnit[1]
				},
			)
			slotResources = append(slotResources, resourcesByType[resourceType][:int(needed)]...)
		}
		if allSatisfied {
			result[slot] = slotResources
		}
	}

	return result
}

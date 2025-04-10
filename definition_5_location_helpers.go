package scheduler

import (
	"math"
	"sort"
)

type paramsPopulatePossibilities struct {
	Candidates             map[uint8][]*Resource
	ResourcesNeededPerType map[uint8]uint16

	TimeInterval

	Duration int64
}

// func populatePossibilities(params *paramsPopulatePossibilities) map[TimeInterval][]*Resource {
// 	result := make(map[TimeInterval][]*Resource)

// 	// Collect all possible slots across types
// 	typeSlots := make(map[TimeInterval]map[uint8][]*Resource)

// 	for resourceType, candidates := range params.Candidates {
// 		needed := int(params.ResourcesNeededPerType[resourceType])

// 		sort.Slice(
// 			candidates,
// 			func(i, j int) bool {
// 				return candidates[i].costPerLoadUnit[1] < candidates[j].costPerLoadUnit[1]
// 			},
// 		)

// 		for _, candidate := range candidates {
// 			availSlots, available := candidate.GetAvailability(&params.TimeInterval)

// 			if available {
// 				current := typeSlots[params.TimeInterval]
// 				if current == nil {
// 					current = make(map[uint8][]*Resource)
// 					typeSlots[params.TimeInterval] = current
// 				}

// 				if len(current[resourceType]) < needed {
// 					current[resourceType] = append(current[resourceType], candidate)
// 				}
// 			}

// 			if !available {
// 				for _, slot := range availSlots {
// 					if slot.TimeEnd-slot.TimeStart >= params.Duration {
// 						normalizedSlot := TimeInterval{
// 							TimeStart:     slot.TimeStart,
// 							TimeEnd:       slot.TimeEnd,
// 							SecondsOffset: params.TimeInterval.SecondsOffset,
// 						}

// 						current := typeSlots[normalizedSlot]
// 						if current == nil {
// 							current = make(map[uint8][]*Resource)
// 							typeSlots[normalizedSlot] = current
// 						}

// 						if len(current[resourceType]) < needed {
// 							current[resourceType] = append(current[resourceType], candidate)
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}

// 	// Filter slots meeting all type-quantity requirements
// 	for slot, resourcesByType := range typeSlots {
// 		allSatisfied := true

// 		for resourceType, needed := range params.ResourcesNeededPerType {
// 			if len(resourcesByType[resourceType]) < int(needed) {
// 				allSatisfied = false
// 				break
// 			}
// 		}

// 		if allSatisfied {
// 			var slotResources []*Resource

// 			for _, resources := range resourcesByType {
// 				slotResources = append(slotResources, resources...)
// 			}

// 			result[slot] = slotResources
// 		}
// 	}

// 	return result
// }

func findEarliestSlot(possibilities map[TimeInterval][]*Resource, neededCount int, offsetDifference int64) (int64, []*Resource) {
	earliest := int64(_NoAvailability)
	var bestResources []*Resource
	bestCost := float32(math.MaxFloat32)

	for slot, resources := range possibilities {
		if len(resources) >= neededCount { // Ensure total quantity across types
			start := slot.TimeStart - offsetDifference
			var totalCost float32
			for _, res := range resources[:neededCount] { // Take only needed
				cost := res.costPerLoadUnit[1] // Assuming LoadUnit 1
				totalCost += cost
			}
			if totalCost < bestCost || (totalCost == bestCost && (earliest == _NoAvailability || start < earliest)) {
				bestCost = totalCost
				earliest = start
				sort.Slice(resources, func(i, j int) bool {
					return resources[i].costPerLoadUnit[1] < resources[j].costPerLoadUnit[1]
				})
				bestResources = resources[:neededCount]
			}
		}
	}

	return earliest, bestResources
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
		var slotResources []*Resource
		for resourceType, needed := range params.ResourcesNeededPerType {
			if len(resourcesByType[resourceType]) < int(needed) {
				allSatisfied = false
				break
			}
			sort.Slice(
				resourcesByType[resourceType],
				func(i, j int) bool {
					return resourcesByType[resourceType][i].costPerLoadUnit[1] <
						resourcesByType[resourceType][j].costPerLoadUnit[1]
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

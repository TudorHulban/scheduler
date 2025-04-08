package scheduler

import (
	"math"
	"sort"
)

type paramsPopulatePossibilities struct {
	Candidates map[uint8][]*Resource

	TimeInterval

	Duration int64
}

func populatePossibilities(params *paramsPopulatePossibilities) map[TimeInterval][]*Resource {
	result := make(map[TimeInterval][]*Resource)

	for _, candidates := range params.Candidates {
		sort.Slice(
			candidates,
			func(i, j int) bool {
				return candidates[i].costPerLoadUnit[1] < candidates[j].costPerLoadUnit[1]
			},
		)

		for ix, candidate := range candidates {
			availSlots, available := candidate.GetAvailability(&params.TimeInterval)
			if ix == 0 && available {
				result[params.TimeInterval] = append(result[params.TimeInterval], candidate)

				continue
			}

			if !available {
				for _, slot := range availSlots {
					if slot.TimeEnd-slot.TimeStart >= params.Duration {
						result[slot] = append(result[slot], candidate)
					}
				}
			}
		}
	}

	return result
}

func findEarliestSlot(possibilities map[TimeInterval][]*Resource, neededCount int, offsetDifference int64) (int64, []*Resource) {
	earliest := int64(_NoAvailability)

	var bestResources []*Resource
	bestCost := float32(math.MaxFloat32)

	for slot, resources := range possibilities {
		if len(resources) >= neededCount {
			start := slot.TimeStart - offsetDifference
			var minCost float32

			for i, res := range resources[:neededCount] {
				cost := res.costPerLoadUnit[1] // Assuming LoadUnit 1
				if i == 0 || cost < minCost {
					minCost = cost
				}
			}

			if minCost < bestCost || (minCost == bestCost && (earliest == _NoAvailability || start < earliest)) {
				bestCost = minCost
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

package scheduler

import "sort"

type paramsGenerateCheapestCombinations struct {
	AvailableResources     ResourcesPerType
	ResourcesNeededPerType map[uint8]uint16
	CostByResource         map[*ResourceScheduled]float32
}

func generateCheapestCombinations(params *paramsGenerateCheapestCombinations) [][]*ResourceScheduled {
	// For each resource type, sort by cost
	for resourceType, resources := range params.AvailableResources {
		sort.Slice(
			resources,
			func(i, j int) bool {
				return params.CostByResource[resources[i]] < params.CostByResource[resources[j]]
			},
		)

		// Keep only the N cheapest resources we need
		needed := int(params.ResourcesNeededPerType[resourceType])

		if needed > 0 && len(resources) > needed {
			params.AvailableResources[resourceType] = resources[:needed]
		}
	}

	// Build the cheapest combination
	var cheapestCombo []*ResourceScheduled

	for resourceType, resources := range params.AvailableResources {
		needed := int(params.ResourcesNeededPerType[resourceType])

		if needed > 0 {
			cheapestCombo = append(
				cheapestCombo,
				resources[:needed]...,
			)
		}
	}

	return [][]*ResourceScheduled{cheapestCombo}
}

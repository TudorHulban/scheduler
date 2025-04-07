package scheduler

import (
	"fmt"
	"sort"
)

func calculateTaskCost(task *Run, res *Resource) (float32, error) {
	costPerUnit, ok := res.costPerLoadUnit[task.RunLoad.LoadUnit]
	if !ok {
		return 0,
			fmt.Errorf("resource does not support load unit %d", task.RunLoad.LoadUnit)
	}

	cost := task.RunLoad.Load * costPerUnit

	return cost, nil
}

func canMeetDependencies(dependencies []RunDependency, available map[uint8][]resourceOption) bool {
	for _, dependency := range dependencies {
		if len(available[dependency.ResourceType]) < int(dependency.ResourceQuantity) {
			return false
		}
	}

	return true
}

func selectCheapestResources(deps []RunDependency, available map[uint8][]resourceOption) (map[int]*Resource, float32) {
	chosen := make(map[int]*Resource)
	var totalCost float32

	for _, dep := range deps {
		options := available[dep.ResourceType]

		// Sort by cost to pick cheapest
		sort.Slice(options, func(i, j int) bool { return options[i].cost < options[j].cost })

		for i := 0; i < int(dep.ResourceQuantity); i++ {
			res := options[i].res

			if _, exists := chosen[res.ID]; !exists { // Avoid reusing same resource unless quantity allows
				chosen[res.ID] = res
				totalCost += options[i].cost
			}
		}
	}

	return chosen, totalCost
}

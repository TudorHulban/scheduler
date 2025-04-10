package scheduler

import (
	"fmt"
)

func calculateTaskCost(task *Run, res *ResourceScheduled) (float32, error) {
	costPerUnit, ok := res.CostPerLoadUnit[task.RunLoad.LoadUnit]
	if !ok {
		return 0,
			fmt.Errorf("resource does not support load unit %d", task.RunLoad.LoadUnit)
	}

	cost := task.RunLoad.Load * costPerUnit

	return cost, nil
}

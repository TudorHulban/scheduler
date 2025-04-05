package scheduler

import "fmt"

func calculateTaskCost(task *Task, res *Resource) (float32, error) {
	costPerUnit, ok := res.costPerLoadUnit[task.TaskLoad.LoadUnit]
	if !ok {
		return 0,
			fmt.Errorf("resource does not support load unit %d", task.TaskLoad.LoadUnit)
	}

	cost := task.TaskLoad.Load * costPerUnit

	return cost, nil
}

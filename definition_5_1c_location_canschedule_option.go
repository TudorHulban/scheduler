package scheduler

import (
	"fmt"
	"strings"
)

// SchedulingOption represents a potential slot for scheduling a task
type SchedulingOption struct {
	WhenCanStart      int64
	SelectedResources []*ResourceScheduled
	Cost              float32
}

func (so *SchedulingOption) String() string {
	var resourcesStr []string
	for _, res := range so.SelectedResources {
		resourcesStr = append(resourcesStr, fmt.Sprintf("{ID: %d, Name: %q}", res.ID, res.Name))
	}
	return fmt.Sprintf(
		"SchedulingOption{\n"+
			"  WhenCanStart: %d,\n"+
			"  SelectedResources: [%s],\n"+
			"  Cost: %.2f\n"+
			"}",
		so.WhenCanStart,
		strings.Join(resourcesStr, ", "),
		so.Cost,
	)
}

type SchedulingOptions []*SchedulingOption

func (sos SchedulingOptions) String() string {
	var sb strings.Builder

	sb.WriteString("[\n")

	for i, opt := range sos {
		sb.WriteString(
			fmt.Sprintf(
				"  %d: %s",
				i,
				opt.String(),
			),
		)

		if i < len(sos)-1 {
			sb.WriteString(",\n")
		}
	}

	sb.WriteString("\n]")

	return sb.String()
}

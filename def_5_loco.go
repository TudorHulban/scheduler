package scheduler

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

type OptionSchedule struct {
	WhenCanStart int64
	Resources    ResourcesPerType
}

func (option OptionSchedule) GetCostFor(task *Run) (float32, error) {
	var costTotal float32

	for _, resources := range option.Resources {
		for _, resource := range resources {
			costResource, errGetCost := calculateTaskCost(task, resource)
			if errGetCost != nil {
				return 0,
					errGetCost
			}

			costTotal = costTotal + costResource
		}
	}

	return costTotal,
		nil
}

func (option OptionSchedule) String(task *Run) string {
	var sb strings.Builder

	sb.WriteString("\nOptionSchedule {")
	sb.WriteString(fmt.Sprintf("WhenCanStart: %d, ", option.WhenCanStart))
	sb.WriteString("Resources: ")

	// Sort resource types for consistent output
	var resourceTypes []uint8
	for rt := range option.Resources {
		resourceTypes = append(resourceTypes, rt)
	}

	slices.Sort(resourceTypes)

	for ix, rt := range resourceTypes {
		resources := option.Resources[rt]

		sb.WriteString(fmt.Sprintf("%d: [", rt))

		for _, res := range resources {
			sb.WriteString(fmt.Sprintf("%s", res.String()))
		}

		sb.WriteString(
			ternary(
				ix == len(resourceTypes)-1,

				"]\n",
				"],\n",
			),
		)

	}

	sb.WriteString("},\n")

	cost, _ := option.GetCostFor(task)

	sb.WriteString(
		fmt.Sprintf(
			" cost: %.2f",

			cost,
		),
	)

	sb.WriteString("}")

	return sb.String()
}

type OptionsSchedule []*OptionSchedule

func (options *OptionsSchedule) String(task *Run) string {
	var sb strings.Builder

	sb.WriteString("OptionsSchedule\n")

	for i, option := range *options {
		stringOption := option.String(task)

		stringOption = strings.Replace(stringOption, "\n", "", -1) // Add indentation
		sb.WriteString(fmt.Sprintf("%d: %s\n", i+1, stringOption))
	}

	return sb.String()
}

type Loco struct {
	Name      string
	Resources ResourcesPerType

	mu sync.Mutex

	ID             int64
	LocationOffset int64
}

// 1. breakdown interval
// 2. check available resources
// 3. sort resources as per search attributes

func (loc *Loco) GetSchedulingOptions(params *ParamsCanRun) (OptionsSchedule, error) {
	intervalsSought := params.TimeInterval.BreakDown(params.TaskRun.EstimatedDuration)

	resourcesNeededPerType := params.TaskRun.GetNeededResourcesPerType()
	neededTypes := params.TaskRun.GetNeededResourceTypes()

	result := make([]*OptionSchedule, 0)

	for _, interval := range intervalsSought {
		intervalResourcesNeeded := make(ResourcesPerType)

		for _, neededType := range neededTypes {
			resourcesNeededPerCurrentType := resourcesNeededPerType[neededType]

			intervalResourcesPerCurrentType := make([]*ResourceScheduled, 0)

			var qty uint16

			for _, resource := range loc.Resources[neededType] {
				if isAvailable := resource.IsAvailableIn(&interval); !isAvailable {
					continue
				}

				intervalResourcesPerCurrentType = append(intervalResourcesPerCurrentType, resource)

				qty = qty + resource.ServedQuantity

				if qty == resourcesNeededPerCurrentType {
					break
				}
			}

			if qty < resourcesNeededPerCurrentType {
				break //interval cannot provide all resources
			}

			intervalResourcesNeeded[neededType] = intervalResourcesPerCurrentType
		}

		result = append(
			result,

			&OptionSchedule{
				WhenCanStart: interval.TimeStart,
				Resources:    intervalResourcesNeeded,
			},
		)
	}

	return result,
		nil
}

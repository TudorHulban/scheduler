package scheduler

import (
	"fmt"
	"strings"
	"sync"

	goerrors "github.com/TudorHulban/go-errors"
	"github.com/asaskevich/govalidator"
)

type Location struct {
	Name      string
	Resources []*ResourceScheduled
	mu        sync.Mutex

	ID             int64
	LocationOffset int64
}

type ParamsNewLocation struct {
	Name      string               `valid:"required"`
	Resources []*ResourceScheduled `valid:"required"`

	ID             int64 `valid:"required"`
	LocationOffset int64
}

func NewLocation(params *ParamsNewLocation) (*Location, error) {
	if _, errValidation := govalidator.ValidateStruct(params); errValidation != nil {
		return nil,
			goerrors.ErrServiceValidation{
				ServiceName: "Organigram",
				Caller:      "NewLocation",
				Issue:       errValidation,
			}
	}

	return &Location{
			ID:             params.ID,
			Name:           params.Name,
			LocationOffset: params.LocationOffset,

			Resources: params.Resources,
		},
		nil
}

type ParamsCanRun struct {
	TimeInterval

	TaskRun *Run

	PossibilitiesUpTo uint8
	AllPossibilities  bool
}

func (p ParamsCanRun) String() string {
	var sb strings.Builder

	sb.WriteString("ParamsCanRun{\n")

	// TimeInterval fields
	sb.WriteString("\tTimeInterval{\n")
	sb.WriteString(fmt.Sprintf("\t\tTimeStart: %d,\n", p.TimeStart))
	sb.WriteString(fmt.Sprintf("\t\tTimeEnd: %d,\n", p.TimeEnd))
	sb.WriteString(fmt.Sprintf("\t\tSecondsOffset: %d,\n", p.SecondsOffset))
	sb.WriteString("\t},\n")

	// TaskRun
	if p.TaskRun != nil {
		sb.WriteString("\tTaskRun: &Run{\n")
		sb.WriteString(fmt.Sprintf("\t\tName: %q,\n", p.TaskRun.Name))

		// Dependencies
		if len(p.TaskRun.Dependencies) > 0 {
			sb.WriteString("\t\tDependencies: []RunDependency{\n")
			for _, dep := range p.TaskRun.Dependencies {
				sb.WriteString("\t\t\t{\n")
				sb.WriteString(fmt.Sprintf("\t\t\t\tPreferredResourceID: %d,\n", dep.PreferredResourceID))
				sb.WriteString(fmt.Sprintf("\t\t\t\tResourceType: %d,\n", dep.ResourceType))
				sb.WriteString(fmt.Sprintf("\t\t\t\tResourceQuantity: %d,\n", dep.ResourceQuantity))
				sb.WriteString("\t\t\t},\n")
			}
			sb.WriteString("\t\t},\n")
		} else {
			sb.WriteString("\t\tDependencies: nil,\n")
		}

		// RunLoad fields
		sb.WriteString("\t\tRunLoad: {\n")
		sb.WriteString(fmt.Sprintf("\t\t\tLoad: %f,\n", p.TaskRun.RunLoad.Load))
		sb.WriteString(fmt.Sprintf("\t\t\tLoadUnit: %d,\n", p.TaskRun.RunLoad.LoadUnit))
		sb.WriteString("\t\t},\n")

		// Other Run fields
		sb.WriteString(fmt.Sprintf("\t\tID: %d,\n", p.TaskRun.ID))
		sb.WriteString(fmt.Sprintf("\t\tInitiatorID: %d,\n", p.TaskRun.InitiatorID))
		sb.WriteString(fmt.Sprintf("\t\tEstimatedDuration: %d,\n", p.TaskRun.EstimatedDuration))
		sb.WriteString("\t},\n")
	} else {
		sb.WriteString("\tTaskRun: nil,\n")
	}

	sb.WriteString("}")

	return sb.String()
}

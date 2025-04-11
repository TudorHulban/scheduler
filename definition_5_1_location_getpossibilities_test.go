package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPossibilities(t *testing.T) {
	location, errCr := NewLocation(
		&ParamsNewLocation{
			ID:   1,
			Name: t.Name(),

			Resources: []*ResourceScheduled{
				{
					ResourceInfo: ResourceInfo{
						ID:              1,
						Name:            "Resource 1",
						CostPerLoadUnit: map[uint8]float32{1: 2.0},
						ResourceType:    1,
					},

					schedule: map[TimeInterval]RunID{},
				},
			},
		},
	)
	require.NoError(t, errCr)
	require.NotNil(t, location)

	possibilities, errGet := location.GetPossibilities(
		&ParamsCanRun{
			TimeInterval: TimeInterval{
				TimeStart: now,
				TimeEnd:   now + halfHour,
			},

			TaskRun: &Run{
				ID:                1,
				EstimatedDuration: halfHour,

				Dependencies: []RunDependency{
					RunDependency{
						ResourceType:     1,
						ResourceQuantity: 1,
					},
				},

				RunLoad: RunLoad{
					Load:     1,
					LoadUnit: 1,
				},
			},
		},
	)
	require.NoError(t, errGet)
	require.NotNil(t, possibilities)

	fmt.Println(
		possibilities,
	)
}

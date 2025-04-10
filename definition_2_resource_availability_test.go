package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAvailability(t *testing.T) {
	targetInterval := TimeInterval{
		TimeStart: now,
		TimeEnd:   now + 2*oneHour,
	}

	t.Run(
		"1. Resource Low Cost",

		func(t *testing.T) {
			resource, errCr := NewResource(
				&ParamsNewResource{
					Name:            "Low Cost",
					ResourceType:    1,
					CostPerLoadUnit: map[uint8]float32{1: 2.0},
				},
			)
			require.NoError(t, errCr)
			require.NotNil(t, resource)

			resource.schedule = map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
				{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
			}

			intervals, isAvailable := resource.GetAvailability(&targetInterval)
			require.False(t, isAvailable)
			require.NotEmpty(t, intervals)

			fmt.Println(
				t.Name(),
				intervals,
			)
		},
	)

	t.Run(
		"2. Resource High Cost",

		func(t *testing.T) {
			resource, errCr := NewResource(
				&ParamsNewResource{
					Name:            "High Cost",
					ResourceType:    1,
					CostPerLoadUnit: map[uint8]float32{1: 3.0},
				},
			)
			require.NoError(t, errCr)
			require.NotNil(t, resource)

			resource.schedule = map[TimeInterval]RunID{
				{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
			}

			intervals, isAvailable := resource.GetAvailability(&targetInterval)
			require.False(t, isAvailable)
			require.NotEmpty(t, intervals)

			fmt.Println(
				t.Name(),
				intervals,
			)
		},
	)

	t.Run(
		"3. Resource Type 2",

		func(t *testing.T) {
			resource, errCr := NewResource(
				&ParamsNewResource{
					Name:            "Type 2",
					ResourceType:    1,
					CostPerLoadUnit: map[uint8]float32{1: 1.0},
				},
			)
			require.NoError(t, errCr)
			require.NotNil(t, resource)

			resource.schedule = map[TimeInterval]RunID{
				{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
			}

			intervals, isAvailable := resource.GetAvailability(&targetInterval)
			require.False(t, isAvailable)
			require.NotEmpty(t, intervals)

			fmt.Println(
				t.Name(),
				intervals,
			)
		},
	)
}

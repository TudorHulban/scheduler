package scheduler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorsResource(t *testing.T) {
	t.Run(
		"1. empty params",
		func(t *testing.T) {
			res, errCr := NewResource(
				&ParamsNewResource{},
			)
			require.Error(t, errCr)
			require.Nil(t, res)
		},
	)

	t.Run(
		"2. empty name",
		func(t *testing.T) {
			res, errCr := NewResource(
				&ParamsNewResource{
					CostPerLoadUnit: map[uint8]float32{
						1: 0.1,
					},
					ResourceType: 1,
				},
			)
			require.Error(t, errCr)
			require.Nil(t, res)
		},
	)

	t.Run(
		"3. empty costs",
		func(t *testing.T) {
			res, errCr := NewResource(
				&ParamsNewResource{
					Name:         "res 1",
					ResourceType: 1,
				},
			)
			require.Error(t, errCr)
			require.Nil(t, res)
		},
	)
}

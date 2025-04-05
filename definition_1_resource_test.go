package scheduler

import (
	"context"
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

func TestLifeCycleResource(t *testing.T) {
	res, errCr := NewResource(
		&ParamsNewResource{
			Name: "res",
			CostPerLoadUnit: map[uint8]float32{
				1: 0.1,
			},
			ResourceType: 1,
		},
	)
	require.NoError(t, errCr)
	require.NotNil(t, res)

	ctx := context.Background()

	taskScheduledAt0, errGetAt0 := res.GetTask(0, 0)
	require.Error(t, errGetAt0)
	require.Nil(t, taskScheduledAt0)

	taskScheduledAt1000, errGetAt1000 := res.GetTask(1000, 0)
	require.Error(t, errGetAt1000)
	require.Nil(t, taskScheduledAt1000)

	overlap, errAddTask := res.AddTask(
		ctx,
		&ParamsTask{
			TimeStart: 1000,
			TimeEnd:   2000,
			GMTOffset: 7200, // 2 hours

			TaskID: 101,
		},
	)
	require.NoError(t, errAddTask)
	require.Empty(t, overlap)
}

package scheduler

import (
	"context"
	"fmt"
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

	ovelapsEmpty, hasAvailabilityEmpty := res.GetAvailability(
		&TimeInterval{
			TimeStart:     0,
			TimeEnd:       2000,
			SecondsOffset: 7200,
		},
	)
	require.True(t, hasAvailabilityEmpty)
	require.Nil(t, ovelapsEmpty)

	taskScheduledAt0, errGetAt0 := res.GetRun(0, 0)
	require.Error(t, errGetAt0)
	require.Nil(t, taskScheduledAt0)

	responseFor1000, errGetAt1000 := res.GetRun(1000, 0)
	require.Error(t, errGetAt1000)
	require.Nil(t, responseFor1000)

	var idRun RunID = 101

	overlapAddTask, errAddTask := res.AddRun(
		ctx,
		&ParamsRun{
			TimeInterval: TimeInterval{
				TimeStart:     1000,
				TimeEnd:       2000,
				SecondsOffset: 7200, // 2 hours
			},

			ID: idRun,
		},
	)
	require.NoError(t, errAddTask)
	require.Empty(t, overlapAddTask)
	require.Len(t,
		res.schedule,
		1,
	)

	fmt.Println(
		res.GetSchedule(),
	)

	overlapFull, noAvailability := res.GetAvailability(
		&TimeInterval{
			TimeStart:     1000,
			TimeEnd:       2000,
			SecondsOffset: 7200,
		},
	)
	require.False(t, noAvailability)
	require.Empty(t, overlapFull)

	overlapsWTask, hasAvailabilityWTask := res.GetAvailability(
		&TimeInterval{
			TimeStart:     0,
			TimeEnd:       3000,
			SecondsOffset: 7200,
		},
	)
	require.False(t, hasAvailabilityWTask)
	require.NotEmpty(t, overlapsWTask)
	require.Len(t,
		overlapsWTask,
		2,
	)
	require.EqualValues(t,
		0,
		overlapsWTask[0].TimeStart,
	)
	require.EqualValues(t,
		1000,
		overlapsWTask[0].TimeEnd,
	)
	require.EqualValues(t,
		2000,
		overlapsWTask[1].TimeStart,
	)
	require.EqualValues(t,
		3000,
		overlapsWTask[1].TimeEnd,
	)

	taskScheduledAt100, errGetAt100 := res.GetRun(100, 0)
	require.Error(t, errGetAt100)
	require.Nil(t, taskScheduledAt100)

	responseTask1000, errGet1000 := res.GetRun(1000, 7200)
	require.NoError(t, errGet1000)
	require.EqualValues(t,
		idRun,
		responseTask1000.ID,
	)

	require.NoError(t,
		res.removeRun(idRun),
	)

	responseRemovedTask1000, errGetRemoved1000 := res.GetRun(1000, 7200)
	require.Error(t, errGetRemoved1000)
	require.Nil(t, responseRemovedTask1000)

	fmt.Println(
		res.GetSchedule(),
	)
}

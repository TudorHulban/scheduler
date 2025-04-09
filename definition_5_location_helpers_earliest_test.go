package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindEarliestSlot(t *testing.T) {
	var now int64 = 10000

	tests := []struct {
		name                   string
		possibilities          map[TimeInterval][]*Resource
		neededCount            int
		offsetDifference       int64
		expectedTime           int64
		expectedResourcesCount int
		expectedCost           float32
	}{
		{
			name: "1. Busy now, cheapest later",
			possibilities: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&Resource{ID: 2, costPerLoadUnit: map[uint8]float32{1: 3.0}}, // High-cost now
				},
				{TimeStart: now + oneHour, TimeEnd: now + 2*oneHour}: {
					&Resource{ID: 1, costPerLoadUnit: map[uint8]float32{1: 2.0}}, // Cheaper later
					&Resource{ID: 2, costPerLoadUnit: map[uint8]float32{1: 3.0}},
				},
			},
			neededCount:      1,
			offsetDifference: 0,

			expectedTime:           now + oneHour,
			expectedResourcesCount: 1,
			expectedCost:           2.0,
		},
		{
			name: "2. Busy now, cheaper later, multiple resources",
			possibilities: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&Resource{ID: 2, costPerLoadUnit: map[uint8]float32{1: 3.0}},
					&Resource{ID: 3, costPerLoadUnit: map[uint8]float32{1: 3.0}},
				},
				{TimeStart: now + oneHour, TimeEnd: now + 2*oneHour}: {
					&Resource{ID: 1, costPerLoadUnit: map[uint8]float32{1: 2.0}},
					&Resource{ID: 2, costPerLoadUnit: map[uint8]float32{1: 3.0}},
				},
			},
			neededCount:      2,
			offsetDifference: 0,

			expectedTime:           now + oneHour,
			expectedResourcesCount: 2,
			expectedCost:           5.0,
		},
		{
			name: "3. Busy now, available next hour",
			possibilities: map[TimeInterval][]*Resource{
				{TimeStart: now + oneHour, TimeEnd: now + 2*oneHour}: {
					&Resource{costPerLoadUnit: map[uint8]float32{1: 2.0}},
					&Resource{costPerLoadUnit: map[uint8]float32{1: 3.0}},
				},
			},
			neededCount:      1,
			offsetDifference: 0,

			expectedTime:           now + oneHour,
			expectedResourcesCount: 1,
			expectedCost:           2.0, // Cheaper resource
		},
		{
			name: "4. Busy now, available next hour, multiple resources",
			possibilities: map[TimeInterval][]*Resource{
				{TimeStart: now + oneHour, TimeEnd: now + 2*oneHour}: {
					&Resource{costPerLoadUnit: map[uint8]float32{1: 2.0}},
					&Resource{costPerLoadUnit: map[uint8]float32{1: 3.0}},
				},
			},
			neededCount:      2,
			offsetDifference: 0,

			expectedTime:           now + oneHour,
			expectedResourcesCount: 2,
			expectedCost:           5.0,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				earliest, resources := findEarliestSlot(
					tt.possibilities,
					tt.neededCount,
					tt.offsetDifference,
				)

				if earliest != tt.expectedTime || len(resources) != tt.expectedResourcesCount {
					t.Errorf(
						"expected {time: %d, count: %d}, got {time: %d, count: %d}",
						tt.expectedTime,
						tt.expectedResourcesCount,
						earliest,
						len(resources),
					)
				}

				require.Len(t,
					resources,
					tt.expectedResourcesCount,

					fmt.Sprintf(
						"expected resource count: %d, got: %d",

						tt.expectedResourcesCount,
						len(resources),
					),
				)

				var totalCost float32

				for _, resource := range resources {
					totalCost = totalCost + resource.costPerLoadUnit[1]
				}

				require.Equal(t,
					tt.expectedCost,
					totalCost,

					fmt.Sprintf(
						"expected cost: %f, got cost: %f",
						tt.expectedCost,
						totalCost,
					),
				)
			},
		)
	}
}

package scheduler

import "testing"

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
				{TimeStart: now, TimeEnd: now + 3600}: {
					&Resource{ID: 2, costPerLoadUnit: map[uint8]float32{1: 3.0}}, // High-cost now
				},
				{TimeStart: now + 3600, TimeEnd: now + 7200}: {
					&Resource{ID: 1, costPerLoadUnit: map[uint8]float32{1: 2.0}}, // Cheapest later
					&Resource{ID: 2, costPerLoadUnit: map[uint8]float32{1: 3.0}},
				},
			},
			neededCount:            1,
			offsetDifference:       0,
			expectedTime:           now + 3600, // 13600
			expectedResourcesCount: 1,
			expectedCost:           2.0,
		},
		{
			name: "2. Busy now, available next hour",
			possibilities: map[TimeInterval][]*Resource{
				{TimeStart: now + 3600, TimeEnd: now + 7200}: {
					&Resource{costPerLoadUnit: map[uint8]float32{1: 2.0}}, // Cheapest
					&Resource{costPerLoadUnit: map[uint8]float32{1: 3.0}},
				},
			},
			neededCount:            1,
			offsetDifference:       0,
			expectedTime:           now + 3600, // 13600
			expectedResourcesCount: 1,
			expectedCost:           2.0, // Cheapest resource
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

				if len(resources) > 0 {
					cost := resources[0].costPerLoadUnit[1] // Assuming LoadUnit 1

					if cost != tt.expectedCost {
						t.Errorf(
							"expected cost: %f, got cost: %f",
							tt.expectedCost,
							cost,
						)
					}
				}
			},
		)
	}
}

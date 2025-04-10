package scheduler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPopulatePossibilities(t *testing.T) {
	tests := []struct {
		name     string
		params   paramsPopulatePossibilities
		expected map[TimeInterval][]*ResourceScheduled
	}{
		{
			name: "1. free, single candidate, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*ResourceScheduled{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{},
					},
				},
			},
		},
		{
			name: "2. multiple candidates with different costs, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{},
						},
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              2,
								CostPerLoadUnit: map[uint8]float32{1: 3.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*ResourceScheduled{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{},
					},
				},
			},
		},
		{
			name: "3. candidate with alternative slots, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now + 3*oneHour, TimeEnd: now + 4*oneHour}: Maintenance,
								{TimeStart: now + 5*oneHour, TimeEnd: now + 6*oneHour}: Maintenance,
							},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*ResourceScheduled{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now + 3*oneHour, TimeEnd: now + 4*oneHour}: Maintenance,
							{TimeStart: now + 5*oneHour, TimeEnd: now + 6*oneHour}: Maintenance,
						},
					},
				},
			},
		},
		{
			name: "4. multiple candidate groups - slide to next hour, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
						},

						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              2,
								CostPerLoadUnit: map[uint8]float32{1: 3.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + 2*oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*ResourceScheduled{
				{TimeStart: now + oneHour, TimeEnd: now + 2*oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
						},
					},
				},
			},
		},
		{
			name: "5. no available candidates",
			params: paramsPopulatePossibilities{
				Candidates:             map[uint8][]*ResourceScheduled{},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               2 * oneHour, // Duration longer than available slots
			},

			expected: map[TimeInterval][]*ResourceScheduled{},
		},
		{
			name: "6. busy resource should not be available, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*ResourceScheduled{},
		},
		{
			name: "7. candidate with partial slot free, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + halfHour}: Maintenance,
							},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour + halfHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*ResourceScheduled{
				{TimeStart: now + halfHour, TimeEnd: now + halfHour + oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + halfHour}: Maintenance,
						},
					},
				},
			},
		},
		{
			name: "8. candidate with partial slot free, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
								{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
							},
						},
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              2,
								CostPerLoadUnit: map[uint8]float32{1: 3.0},
							},

							schedule: map[TimeInterval]RunID{},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 2*oneHour,
				},
				Duration: halfHour,
			},

			expected: ResourcesPerTimeInterval{
				{TimeStart: now, TimeEnd: now + halfHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              2,
							CostPerLoadUnit: map[uint8]float32{1: 3.0},
						},

						schedule: map[TimeInterval]RunID{},
					},
				},
				{TimeStart: now + halfHour, TimeEnd: now + oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              2,
							CostPerLoadUnit: map[uint8]float32{1: 3.0},
						},

						schedule: map[TimeInterval]RunID{},
					},
				},
				{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{},
					},
				},
				{TimeStart: now + oneHour + halfHour, TimeEnd: now + 2*oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{},
					},
				},
			},
		},
		// may fail on batch test due to map oprdering.
		{
			name: "9. multiple candidates with partial slot free, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*ResourceScheduled{
					1: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              1,
								CostPerLoadUnit: map[uint8]float32{1: 2.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
								{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
							},
						},
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              2,
								CostPerLoadUnit: map[uint8]float32{1: 3.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
							},
						},
					},
					2: {
						&ResourceScheduled{
							ResourceInfo: ResourceInfo{
								ID:              3,
								CostPerLoadUnit: map[uint8]float32{1: 1.0},
							},

							schedule: map[TimeInterval]RunID{
								{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
							},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{
					1: 1,
					2: 1,
				},
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 2*oneHour,
				},
				Duration: halfHour,
			},

			expected: map[TimeInterval][]*ResourceScheduled{
				{TimeStart: now, TimeEnd: now + halfHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              2,
							CostPerLoadUnit: map[uint8]float32{1: 3.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
					},
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              3,
							CostPerLoadUnit: map[uint8]float32{1: 1.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
					},
				},
				{TimeStart: now + halfHour, TimeEnd: now + oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              2,
							CostPerLoadUnit: map[uint8]float32{1: 3.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
					},
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              3,
							CostPerLoadUnit: map[uint8]float32{1: 1.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
					},
				},
				{TimeStart: now + oneHour + halfHour, TimeEnd: now + 2*oneHour}: {
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              1,
							CostPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
							{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
						},
					},
					&ResourceScheduled{
						ResourceInfo: ResourceInfo{
							ID:              3,
							CostPerLoadUnit: map[uint8]float32{1: 1.0},
						},

						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				result := populatePossibilities(&tt.params)

				assert.Equal(t,
					len(tt.expected),
					len(result),

					fmt.Sprintf(
						"expected %d time intervals, got %d (%s)",

						len(tt.expected),
						len(result),
						result.String(),
					),
				)

				// Compare each time interval
				for interval, expectedResources := range tt.expected {
					resources, ok := result[interval]
					if !ok {
						t.Errorf("expected interval %v not found in results", interval)

						continue
					}

					if len(resources) != len(expectedResources) {
						t.Errorf(
							"for interval %v, expected %d resources, got %d",
							interval,
							len(expectedResources),
							len(resources),
						)

						continue
					}

					for i := range resources {
						if resources[i].ID != expectedResources[i].ID {
							t.Errorf(
								"for interval %v, resource %d has wrong ID (expected %d, got %d)",
								interval,
								i,
								expectedResources[i].ID,
								resources[i].ID,
							)
						}

						if resources[i].CostPerLoadUnit[1] != expectedResources[i].CostPerLoadUnit[1] {
							t.Errorf(
								"for interval %v, resource %d has wrong cost (expected %f, got %f)",
								interval,
								i,
								expectedResources[i].CostPerLoadUnit[1],
								resources[i].CostPerLoadUnit[1],
							)
						}
					}
				}
			},
		)
	}
}

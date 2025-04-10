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
		expected map[TimeInterval][]*Resource
	}{
		{
			name: "1. free, single candidate, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID:              1,
							schedule:        map[TimeInterval]RunID{},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&Resource{
						ID:              1,
						schedule:        map[TimeInterval]RunID{},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
				},
			},
		},
		{
			name: "2. multiple candidates with different costs, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID:              1,
							schedule:        map[TimeInterval]RunID{},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},
						&Resource{
							ID: 2,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 3.0},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&Resource{
						ID:              1,
						schedule:        map[TimeInterval]RunID{},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
				},
			},
		},
		{
			name: "3. candidate with alternative slots, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID: 1,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now + 3*oneHour, TimeEnd: now + 4*oneHour}: Maintenance,
								{TimeStart: now + 5*oneHour, TimeEnd: now + 6*oneHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					&Resource{
						ID: 1,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now + 3*oneHour, TimeEnd: now + 4*oneHour}: Maintenance,
							{TimeStart: now + 5*oneHour, TimeEnd: now + 6*oneHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
				},
			},
		},
		{
			name: "4. multiple candidate groups - slide to next hour, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID: 1,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},

						&Resource{
							ID: 2,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 3.0},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + 2*oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*Resource{
				{TimeStart: now + oneHour, TimeEnd: now + 2*oneHour}: {
					&Resource{
						ID: 1,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
				},
			},
		},
		{
			name: "5. no available candidates",
			params: paramsPopulatePossibilities{
				Candidates:             map[uint8][]*Resource{},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               2 * oneHour, // Duration longer than available slots
			},

			expected: map[TimeInterval][]*Resource{},
		},
		{
			name: "6. busy resource should not be available, exact interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID: 1,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*Resource{},
		},
		{
			name: "7. candidate with partial slot free, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID: 1,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + halfHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},
					},
				},
				ResourcesNeededPerType: map[uint8]uint16{1: 1},
				TimeInterval:           TimeInterval{TimeStart: now, TimeEnd: now + oneHour + halfHour},
				Duration:               oneHour,
			},

			expected: map[TimeInterval][]*Resource{
				{TimeStart: now + halfHour, TimeEnd: now + halfHour + oneHour}: {
					&Resource{
						ID: 1,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + halfHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
				},
			},
		},
		{
			name: "8. candidate with partial slot free, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID: 1,

							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
								{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},
						&Resource{
							ID: 2,

							schedule:        map[TimeInterval]RunID{},
							costPerLoadUnit: map[uint8]float32{1: 3.0},
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
					&Resource{
						ID:              2,
						schedule:        map[TimeInterval]RunID{},
						costPerLoadUnit: map[uint8]float32{1: 3.0},
					},
				},
				{TimeStart: now + halfHour, TimeEnd: now + oneHour}: {
					&Resource{
						ID:              2,
						schedule:        map[TimeInterval]RunID{},
						costPerLoadUnit: map[uint8]float32{1: 3.0},
					},
				},
				{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: {
					&Resource{
						ID:              1,
						schedule:        map[TimeInterval]RunID{},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
				},
				{TimeStart: now + oneHour + halfHour, TimeEnd: now + 2*oneHour}: {
					&Resource{
						ID:              1,
						schedule:        map[TimeInterval]RunID{},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
				},
			},
		},
		// may fail on batch test due to map oprdering.
		{
			name: "9. multiple candidates with partial slot free, looser interval",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						&Resource{
							ID: 1,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
								{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 2.0},
						},
						&Resource{
							ID: 2,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 3.0},
						},
					},
					2: {
						&Resource{
							ID: 3,
							schedule: map[TimeInterval]RunID{
								{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
							},
							costPerLoadUnit: map[uint8]float32{1: 1.0},
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

			expected: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + halfHour}: {
					&Resource{
						ID: 2,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 3.0},
					},
					&Resource{
						ID: 3,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 1.0},
					},
				},
				{TimeStart: now + halfHour, TimeEnd: now + oneHour}: {
					&Resource{
						ID: 2,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 3.0},
					},
					&Resource{
						ID: 3,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 1.0},
					},
				},
				{TimeStart: now + oneHour + halfHour, TimeEnd: now + 2*oneHour}: {
					&Resource{
						ID: 1,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
							{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 2.0},
					},
					&Resource{
						ID: 3,
						schedule: map[TimeInterval]RunID{
							{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
						},
						costPerLoadUnit: map[uint8]float32{1: 1.0},
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

						if resources[i].costPerLoadUnit[1] != expectedResources[i].costPerLoadUnit[1] {
							t.Errorf(
								"for interval %v, resource %d has wrong cost (expected %f, got %f)",
								interval,
								i,
								expectedResources[i].costPerLoadUnit[1],
								resources[i].costPerLoadUnit[1],
							)
						}
					}
				}
			},
		)
	}
}

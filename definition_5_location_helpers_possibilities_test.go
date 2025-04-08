package scheduler

import (
	"testing"
)

func TestPopulatePossibilities(t *testing.T) {
	var now int64 = 10000

	halfHour := int64(1800)
	oneHour := int64(3600)

	createResource := func(name string, id int, resType uint8, cost map[uint8]float32, schedule map[TimeInterval]RunID) *Resource {
		return &Resource{
			Name:            name,
			ID:              id,
			ResourceType:    resType,
			costPerLoadUnit: cost,
			schedule:        schedule,
		}
	}

	tests := []struct {
		name     string
		params   paramsPopulatePossibilities
		expected map[TimeInterval][]*Resource
	}{
		{
			name: "1. single candidate with exact availability",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						createResource(
							"res1",
							1,
							1,
							map[uint8]float32{1: 1.0},
							map[TimeInterval]RunID{},
						),
					},
				},
				TimeInterval: TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:     oneHour,
			},

			expected: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{}),
				},
			},
		},
		{
			name: "2. multiple candidates with different costs",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						createResource("res1", 1, 1, map[uint8]float32{1: 2.0}, map[TimeInterval]RunID{}),
						createResource("res2", 2, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{}),
					},
				},
				TimeInterval: TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:     oneHour,
			},
			expected: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					createResource("res2", 2, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{}),
				},
			},
		},
		{
			name: "3. candidate with alternative slots",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{
							{TimeStart: now + 3*oneHour, TimeEnd: now + 4*oneHour}: Maintenance,
							{TimeStart: now + 5*oneHour, TimeEnd: now + 6*oneHour}: Maintenance,
						}),
					},
				},
				TimeInterval: TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:     oneHour,
			},
			expected: map[TimeInterval][]*Resource{
				{TimeStart: now, TimeEnd: now + oneHour}: {
					createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{
						{TimeStart: now + 3*oneHour, TimeEnd: now + 4*oneHour}: Maintenance,
						{TimeStart: now + 5*oneHour, TimeEnd: now + 6*oneHour}: Maintenance,
					}),
				},
			},
		},
		{
			name: "4. multiple candidate groups",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
						}),
					},
					2: {
						createResource("res2", 2, 2, map[uint8]float32{1: 2.0}, map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
						}),
					},
				},
				TimeInterval: TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:     oneHour,
			},
			expected: map[TimeInterval][]*Resource{
				{TimeStart: now + oneHour, TimeEnd: now + 2*oneHour}: {
					createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{
						{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
					}),
					createResource("res2", 2, 2, map[uint8]float32{1: 2.0}, map[TimeInterval]RunID{
						{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
					}),
				},
			},
		},
		{
			name: "5. no available candidates",
			params: paramsPopulatePossibilities{
				Candidates:   map[uint8][]*Resource{},
				TimeInterval: TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:     2 * oneHour, // Duration longer than available slots
			},

			expected: map[TimeInterval][]*Resource{},
		},
		{
			name: "6. busy resource should not be available",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + oneHour}: Maintenance, // busy (non-zero RunID)
						}),
					},
				},
				TimeInterval: TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:     oneHour,
			},

			expected: map[TimeInterval][]*Resource{},
		},
		{
			name: "7. candidate with partial slot free",
			params: paramsPopulatePossibilities{
				Candidates: map[uint8][]*Resource{
					1: {
						createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{
							{TimeStart: now, TimeEnd: now + halfHour}: Maintenance,
						}),
					},
				},
				TimeInterval: TimeInterval{TimeStart: now, TimeEnd: now + oneHour},
				Duration:     oneHour,
			},
			expected: map[TimeInterval][]*Resource{
				{TimeStart: now + halfHour, TimeEnd: now + halfHour + oneHour}: {
					createResource("res1", 1, 1, map[uint8]float32{1: 1.0}, map[TimeInterval]RunID{
						{TimeStart: now, TimeEnd: now + halfHour}: Maintenance,
					}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				result := populatePossibilities(&tt.params)

				// Compare lengths first
				if len(result) != len(tt.expected) {
					t.Errorf("expected %d time intervals, got %d", len(tt.expected), len(result))
				}

				// Compare each time interval
				for interval, expectedResources := range tt.expected {
					resources, ok := result[interval]
					if !ok {
						t.Errorf("expected interval %v not found in results", interval)
						continue
					}

					if len(resources) != len(expectedResources) {
						t.Errorf("for interval %v, expected %d resources, got %d", interval, len(expectedResources), len(resources))
						continue
					}

					for i := range resources {
						if resources[i].ID != expectedResources[i].ID {
							t.Errorf("for interval %v, resource %d has wrong ID (expected %d, got %d)",
								interval, i, expectedResources[i].ID, resources[i].ID)
						}

						if resources[i].costPerLoadUnit[1] != expectedResources[i].costPerLoadUnit[1] {
							t.Errorf("for interval %v, resource %d has wrong cost (expected %f, got %f)",
								interval, i, expectedResources[i].costPerLoadUnit[1], resources[i].costPerLoadUnit[1])
						}
					}
				}
			},
		)
	}
}

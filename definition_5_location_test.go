package scheduler

import (
	"testing"
)

func TestCanSchedule(t *testing.T) {
	resourceLowCost := Resource{
		ID:              1,
		Name:            "Low Cost",
		ResourceType:    1,
		costPerLoadUnit: map[uint8]float32{1: 2.0},
		schedule:        make(map[TimeInterval]RunID),
	}

	resourceHighCost := Resource{
		ID:              2,
		Name:            "High Cost",
		ResourceType:    1,
		costPerLoadUnit: map[uint8]float32{1: 3.0},
		schedule:        make(map[TimeInterval]RunID),
	}

	resourceType2 := Resource{
		ID:              3,
		Name:            "Resource Type 2",
		ResourceType:    2,
		costPerLoadUnit: map[uint8]float32{1: 1.0},
		schedule:        make(map[TimeInterval]RunID),
	}

	location := Location{
		Name: "Test Location",
		Resources: []*Resource{
			&resourceLowCost,
			&resourceHighCost,
		},
	}

	tests := []struct {
		name                     string
		scheduleResourceLowCost  map[TimeInterval]RunID
		scheduleResourceHighCost map[TimeInterval]RunID
		scheduleResourceType2    map[TimeInterval]RunID
		params                   ParamsCanRun
		expectedResult           ResponseCanRun
	}{
		{
			name: "1a. Empty schedule - Immediately available of resource low cost",

			scheduleResourceLowCost:  map[TimeInterval]RunID{},
			scheduleResourceHighCost: map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + oneHour,
				},
				TaskRun: &Run{
					ID:                1,
					Name:              "1a.",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: _ScheduledForStart,
				Cost:         2.0,
				WasScheduled: true,
			},
		},
		{
			name: "1b. Empty schedule - Immediately available of resource low cost",

			scheduleResourceLowCost:  map[TimeInterval]RunID{},
			scheduleResourceHighCost: map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + oneHour,
				},
				TaskRun: &Run{
					ID:                1,
					Name:              "1b.",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 2,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: _ScheduledForStart,
				Cost:         5.0,
				WasScheduled: true,
			},
		},
		{
			name: "1c. Empty schedule - Immediately available of resource low cost",

			scheduleResourceLowCost:  map[TimeInterval]RunID{},
			scheduleResourceHighCost: map[TimeInterval]RunID{},
			scheduleResourceType2:    map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + oneHour,
				},
				TaskRun: &Run{
					ID:                1,
					Name:              "1c.(modified 1a.)",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
						{
							ResourceType:     2,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: _ScheduledForStart,
				Cost:         3.0,
				WasScheduled: true,
			},
		},
		{
			name: "1d. Empty schedule - Immediately available of resource low cost",

			scheduleResourceLowCost:  map[TimeInterval]RunID{},
			scheduleResourceHighCost: map[TimeInterval]RunID{},
			scheduleResourceType2:    map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + oneHour,
				},
				TaskRun: &Run{
					ID:                1,
					Name:              "1d.(modified 1b.)",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 2,
						},
						{
							ResourceType:     2,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: _ScheduledForStart,
				Cost:         6.0,
				WasScheduled: true,
			},
		},
		{
			name: "2a. Busy now, available next hour, looser interval",

			scheduleResourceLowCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
			},
			scheduleResourceHighCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
			},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 2*oneHour,
				},
				TaskRun: &Run{
					ID:                3,
					Name:              "2a.",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: now + oneHour,
				Cost:         2.0,
				WasScheduled: false,
			},
		},
		{
			name: "2b. Busy now, available next hour, looser interval",

			scheduleResourceLowCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
			},
			scheduleResourceHighCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
			},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 2*oneHour,
				},
				TaskRun: &Run{
					ID:                3,
					Name:              "2b.",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 2,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: now + oneHour,
				Cost:         5.0,
				WasScheduled: false,
			},
		},
		{
			name: "2c. Busy now, available next hour, looser interval",

			scheduleResourceLowCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
			},
			scheduleResourceHighCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}: Maintenance,
			},
			scheduleResourceType2: map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 2*oneHour,
				},
				TaskRun: &Run{
					ID:                3,
					Name:              "2c.(modified 2b.)",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 2,
						},
						{
							ResourceType:     2,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: now + oneHour,
				Cost:         6.0,
				WasScheduled: false,
			},
		},
		{
			name: "3a. Timezone conversion (task UTC+2, resource UTC)",

			scheduleResourceLowCost: map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart:     now,
					TimeEnd:       now + oneHour,
					SecondsOffset: 2 * oneHour, // UTC+2
				},
				TaskRun: &Run{
					ID:                4,
					Name:              "3a.",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: _ScheduledForStart,
				Cost:         2.0, // Cheapest resource
				WasScheduled: true,
			},
		},
		{
			name: "3b. Timezone conversion (task UTC+2, resource UTC)",

			scheduleResourceLowCost: map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart:     now,
					TimeEnd:       now + oneHour,
					SecondsOffset: 2 * oneHour, // UTC+2
				},
				TaskRun: &Run{
					ID:                4,
					Name:              "3b.",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
						{
							ResourceType:     2,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: _ScheduledForStart,
				Cost:         3.0,
				WasScheduled: true,
			},
		},
		{
			name: "4a. Multiple busy periods, looser interval",

			scheduleResourceLowCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
				{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
			},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 2*oneHour,
				},
				TaskRun: &Run{
					ID:                7,
					Name:              "4a.",
					EstimatedDuration: halfHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: now + oneHour, // Gap between busy periods
				Cost:         2.0,
				WasScheduled: false,
			},
		},
		{
			name: "4b. Multiple busy periods, looser interval",

			scheduleResourceLowCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
				{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
			},
			scheduleResourceType2: map[TimeInterval]RunID{
				{TimeStart: now + oneHour, TimeEnd: now + oneHour + halfHour}: Maintenance,
			},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 2*oneHour,
				},
				TaskRun: &Run{
					ID:                7,
					Name:              "4b.",
					EstimatedDuration: halfHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
						{
							ResourceType:     2,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: now + oneHour + halfHour, // Gap between busy periods
				Cost:         3.0,
				WasScheduled: false,
			},
		},
		{
			name: "5. No availability cheap resource - Exceeds maximum time",

			scheduleResourceLowCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneDay}: Maintenance,
			},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + oneHour,
				},
				TaskRun: &Run{
					ID:                9,
					Name:              "5",
					EstimatedDuration: oneHour,
					Dependencies: []RunDependency{
						{
							ResourceType:     1,
							ResourceQuantity: 1,
						},
					},
					RunLoad: RunLoad{
						Load:     1.0,
						LoadUnit: 1,
					},
				},
			},

			expectedResult: ResponseCanRun{
				WhenCanStart: _ScheduledForStart, // can use higher cost resource
				Cost:         3,
				WasScheduled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				resourceLowCost.schedule = tt.scheduleResourceLowCost
				resourceHighCost.schedule = ternary(
					len(tt.scheduleResourceHighCost) > 0,

					tt.scheduleResourceHighCost,
					make(map[TimeInterval]RunID),
				)
				resourceType2.schedule = ternary(
					len(tt.scheduleResourceType2) > 0,

					tt.scheduleResourceType2,
					make(map[TimeInterval]RunID),
				)

				location.Resources = []*Resource{
					&resourceLowCost,
					&resourceHighCost,
					&resourceType2,
				}

				result, err := location.CanSchedule(
					&tt.params,
				)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if result.WhenCanStart != tt.expectedResult.WhenCanStart ||
					result.Cost != tt.expectedResult.Cost ||
					result.WasScheduled != tt.expectedResult.WasScheduled {
					t.Errorf(
						"expected %+v, got %+v",
						tt.expectedResult,
						*result,
					)

				}
			},
		)
	}
}

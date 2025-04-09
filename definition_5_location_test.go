package scheduler

import (
	"testing"
)

func TestCanSchedule(t *testing.T) {
	resourceLowCost := &Resource{
		ID:              1,
		Name:            "Low Cost",
		ResourceType:    1,
		costPerLoadUnit: map[uint8]float32{1: 2.0},
		schedule:        make(map[TimeInterval]RunID),
	}

	resourceHighCost := &Resource{
		ID:              2,
		Name:            "High Cost",
		ResourceType:    1,
		costPerLoadUnit: map[uint8]float32{1: 3.0},
		schedule:        make(map[TimeInterval]RunID),
	}

	location := Location{
		Name: "Test Location",
		Resources: []*Resource{
			resourceLowCost,
			resourceHighCost,
		},
	}

	tests := []struct {
		name                     string
		scheduleResourceLowCost  map[TimeInterval]RunID
		scheduleResourceHighCost map[TimeInterval]RunID
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
			name: "3. Timezone conversion (task UTC+2, resource UTC)",

			scheduleResourceLowCost: map[TimeInterval]RunID{},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart:     now,
					TimeEnd:       now + oneHour,
					SecondsOffset: 2 * oneHour, // UTC+2
				},
				TaskRun: &Run{
					ID:                4,
					Name:              "3.",
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
			name: "4. Multiple busy periods, looser interval",

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
					Name:              "4.",
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
				// Reset resources and apply schedule to resource1
				resourceLowCost.schedule = tt.scheduleResourceLowCost
				resourceHighCost.schedule = make(map[TimeInterval]RunID)

				location.Resources = []*Resource{
					resourceLowCost,
					resourceHighCost,
				}

				result, err := location.CanSchedule(&tt.params)
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

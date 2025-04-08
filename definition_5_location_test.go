package scheduler

import (
	"testing"
)

func TestCanSchedule(t *testing.T) {
	var now int64 = 10000

	resourceLowCost := &Resource{
		ID:              1,
		ResourceType:    1,
		costPerLoadUnit: map[uint8]float32{1: 2.0},
		schedule:        make(map[TimeInterval]RunID),
	}

	resourceHighCost := &Resource{
		ID:              2,
		ResourceType:    1,
		costPerLoadUnit: map[uint8]float32{1: 3.0},
		schedule:        make(map[TimeInterval]RunID),
	}

	loc := &Location{
		Name: "TestLoc",
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
		// {
		// 	name: "1. Empty schedule - Immediately available of resource low cost",

		// 	scheduleResourceLowCost: map[TimeInterval]RunID{},
		// 	params: ParamsCanRun{
		// 		TimeInterval: TimeInterval{
		// 			TimeStart: now,
		// 			TimeEnd:   now + 3600,
		// 		},
		// 		TaskRun: &Run{
		// 			ID:                1,
		// 			EstimatedDuration: 3600,
		// 			Dependencies: []RunDependency{
		// 				{
		// 					ResourceType:     1,
		// 					ResourceQuantity: 1,
		// 				},
		// 			},
		// 			RunLoad: RunLoad{
		// 				Load:     1.0,
		// 				LoadUnit: 1,
		// 			},
		// 		},
		// 	},

		// 	expectedResult: ResponseCanRun{
		// 		WhenCanStart: 0,
		// 		Cost:         2.0,
		// 		WasScheduled: true,
		// 	},
		// },
		{
			name: "2. Busy now, available next hour",

			scheduleResourceLowCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + 3600}: Maintenance,
			},
			scheduleResourceHighCost: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + 3600}: Maintenance,
			},
			params: ParamsCanRun{
				TimeInterval: TimeInterval{
					TimeStart: now,
					TimeEnd:   now + 3600,
				},
				TaskRun: &Run{
					ID:                3,
					EstimatedDuration: 3600,
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
				WhenCanStart: now + 3600, // Next hour
				Cost:         2.0,
				WasScheduled: false,
			},
		},
		// {
		// 	name: "3. Timezone conversion (task UTC+2, resource UTC)",

		// 	scheduleResourceLowCost: map[TimeInterval]RunID{},
		// 	params: ParamsCanRun{
		// 		TimeInterval: TimeInterval{
		// 			TimeStart:     now,
		// 			TimeEnd:       now + 3600,
		// 			SecondsOffset: 7200, // UTC+2
		// 		},
		// 		TaskRun: &Run{
		// 			ID:                4,
		// 			EstimatedDuration: 3600,
		// 			Dependencies: []RunDependency{
		// 				{
		// 					ResourceType:     1,
		// 					ResourceQuantity: 1,
		// 				},
		// 			},
		// 			RunLoad: RunLoad{
		// 				Load:     1.0,
		// 				LoadUnit: 1,
		// 			},
		// 		},
		// 	},

		// 	expectedResult: ResponseCanRun{
		// 		WhenCanStart: 0,
		// 		Cost:         2.0, // Cheapest resource
		// 		WasScheduled: true,
		// 	},
		// },
		// {
		// 	name: "4. Multiple busy periods",

		// 	scheduleResourceLowCost: map[TimeInterval]RunID{
		// 		{TimeStart: now, TimeEnd: now + 3600}:         5,
		// 		{TimeStart: now + 7200, TimeEnd: now + 10800}: 6,
		// 	},
		// 	params: ParamsCanRun{
		// 		TimeInterval: TimeInterval{
		// 			TimeStart: now,
		// 			TimeEnd:   now + 3600,
		// 		},
		// 		TaskRun: &Run{
		// 			ID:                7,
		// 			EstimatedDuration: 1800, // 30min
		// 			Dependencies: []RunDependency{
		// 				{
		// 					ResourceType:     1,
		// 					ResourceQuantity: 1,
		// 				},
		// 			},
		// 			RunLoad: RunLoad{
		// 				Load:     1.0,
		// 				LoadUnit: 1,
		// 			},
		// 		},
		// 	},

		// 	expectedResult: ResponseCanRun{
		// 		WhenCanStart: now + 3600, // Gap between busy periods
		// 		Cost:         2.0,
		// 		WasScheduled: false,
		// 	},
		// },
		// {
		// 	name: "5. No availability cheap resource- Exceeds maximum time",

		// 	scheduleResourceLowCost: map[TimeInterval]RunID{
		// 		{TimeStart: now, TimeEnd: now + 86400}: Maintenance, // Full day busy
		// 	},
		// 	params: ParamsCanRun{
		// 		TimeInterval: TimeInterval{
		// 			TimeStart: now,
		// 			TimeEnd:   now + 3600,
		// 		},
		// 		TaskRun: &Run{
		// 			ID:                9,
		// 			EstimatedDuration: 3600,
		// 			Dependencies: []RunDependency{
		// 				{
		// 					ResourceType:     1,
		// 					ResourceQuantity: 1,
		// 				},
		// 			},
		// 			RunLoad: RunLoad{
		// 				Load:     1.0,
		// 				LoadUnit: 1,
		// 			},
		// 		},
		// 	},

		// 	expectedResult: ResponseCanRun{
		// 		WhenCanStart: 0, // can use higher cost resource
		// 		Cost:         3,
		// 		WasScheduled: true,
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				// Reset resources and apply schedule to resource1
				resourceLowCost.schedule = tt.scheduleResourceLowCost
				resourceHighCost.schedule = make(map[TimeInterval]RunID)

				loc.Resources = []*Resource{
					resourceLowCost,
					resourceHighCost,
				}

				result, err := loc.CanSchedule(&tt.params)
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

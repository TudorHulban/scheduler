package scheduler

import (
	"testing"
)

func TestFindAvailableTime(t *testing.T) {
	var now int64 = 10000

	tests := []struct {
		name           string
		schedule       map[TimeInterval]RunID
		params         paramsFindAvailableTime
		expectedResult int64
	}{
		{
			name:     "1. Empty schedule - Immediately available",
			schedule: map[TimeInterval]RunID{},
			params: paramsFindAvailableTime{
				MaximumTimeStart:      now + oneDay,
				TimeStart:             now,
				SecondsDuration:       oneHour,
				SecondsOffsetTask:     0, // UTC
				SecondsOffsetLocation: 0, // UTC
			},

			expectedResult: now,
		},
		{
			name: "2. Busy now, available next hour",
			schedule: map[TimeInterval]RunID{
				{
					TimeStart: now,
					TimeEnd:   now + oneHour,
				}: 1,
			},
			params: paramsFindAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + oneDay,
				SecondsDuration:  oneHour,
			},

			expectedResult: now + oneHour, // Next slot after busy period
		},
		{
			name:     "3. Timezone conversion (task UTC+2, resource UTC)",
			schedule: map[TimeInterval]RunID{},
			params: paramsFindAvailableTime{
				TimeStart:             now,
				MaximumTimeStart:      now + 7200, // 2h window
				SecondsDuration:       3600,
				SecondsOffsetTask:     7200, // UTC+2
				SecondsOffsetLocation: 0,    // UTC
			},

			expectedResult: now, // request time
		},
		{
			name: "4. Multiple busy periods - earliest available",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + 3600}:         1,
				{TimeStart: now + 7200, TimeEnd: now + 10800}: 2,
			},
			params: paramsFindAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + 86400,
				SecondsDuration:  1800, // 30min slot
			},

			expectedResult: now + 3600,
		},
		{
			name: "5. Multiple busy periods - latest available",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + 3600}:         1,
				{TimeStart: now + 7200, TimeEnd: now + 10800}: 2,
			},
			params: paramsFindAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + 86400,
				SecondsDuration:  1800, // 30min slot

				IsLatest: true,
			},

			expectedResult: now + 84600,
		},
		{
			name: "6. No availability - Exceeds maximum time",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + 86400}: Maintenance,
			},
			params: paramsFindAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + 3600,
				SecondsDuration:  3600,
			},

			expectedResult: _NoAvailability,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				res := &Resource{
					schedule: tt.schedule,
				}

				result := res.findAvailableTime(&tt.params)
				if result != tt.expectedResult {
					t.Errorf(
						"expected %d, got %d",
						tt.expectedResult,
						result,
					)
				}
			},
		)
	}
}

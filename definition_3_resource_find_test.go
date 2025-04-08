package scheduler

import (
	"testing"
)

func TestFindAvailableTime(t *testing.T) {
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
				TimeStart:             now,
				MaximumTimeStart:      now + oneDay,
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
				}: Maintenance,
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
				MaximumTimeStart:      now + 2*oneHour,
				SecondsDuration:       oneHour,
				SecondsOffsetTask:     2 * oneHour, // UTC+2
				SecondsOffsetLocation: 0,           // UTC
			},

			expectedResult: now, // request time
		},
		{
			name: "4. Multiple busy periods - earliest available",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
				{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
			},
			params: paramsFindAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + oneDay,
				SecondsDuration:  halfHour,
			},

			expectedResult: now + oneHour,
		},
		{
			name: "5. Multiple busy periods - latest available",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneHour}:               Maintenance,
				{TimeStart: now + 2*oneHour, TimeEnd: now + 3*oneHour}: Maintenance,
			},
			params: paramsFindAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + oneDay,
				SecondsDuration:  halfHour,

				IsLatest: true,
			},

			expectedResult: now + oneDay,
		},
		{
			name: "6. No availability - Exceeds maximum time",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + oneDay}: Maintenance,
			},
			params: paramsFindAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + oneHour,
				SecondsDuration:  oneHour,
			},

			expectedResult: _NoAvailability,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				res := Resource{
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

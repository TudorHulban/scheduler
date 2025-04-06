package scheduler

import (
	"testing"
)

func TestFindEarliestAvailableTime(t *testing.T) {
	var now int64 = 10000

	tests := []struct {
		name           string
		schedule       map[TimeInterval]RunID
		params         paramsFindEarliestAvailableTime
		expectedResult int64
	}{
		{
			name:     "1. Empty schedule - Immediately available",
			schedule: map[TimeInterval]RunID{},
			params: paramsFindEarliestAvailableTime{
				MaximumTimeStart: now + 86400, // 24h window
				TimeStart:        now,
				Duration:         3600, // 1h duration
				OffsetTask:       0,    // UTC
				OffsetLocation:   0,    // UTC
			},

			expectedResult: now,
		},
		{
			name: "2. Busy now, available next hour",
			schedule: map[TimeInterval]RunID{
				{
					TimeStart: now,
					TimeEnd:   now + 3600,
				}: 1,
			},
			params: paramsFindEarliestAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + 86400,
				Duration:         3600,
			},

			expectedResult: now + 3600, // Next slot after busy period
		},
		{
			name:     "3. Timezone conversion (task UTC+2, resource UTC)",
			schedule: map[TimeInterval]RunID{},
			params: paramsFindEarliestAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + 7200, // 2h window
				Duration:         3600,
				OffsetTask:       7200, // UTC+2
				OffsetLocation:   0,    // UTC
			},

			expectedResult: now, // request time
		},
		{
			name: "4. Multiple busy periods",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + 3600}:         1,
				{TimeStart: now + 7200, TimeEnd: now + 10800}: 2,
			},
			params: paramsFindEarliestAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + 86400,
				Duration:         1800, // 30min slot
			},

			expectedResult: now + 3600, // First available between busy periods
		},
		{
			name: "5. No availability - Exceeds maximum time",
			schedule: map[TimeInterval]RunID{
				{TimeStart: now, TimeEnd: now + 86400}: 1,
			},
			params: paramsFindEarliestAvailableTime{
				TimeStart:        now,
				MaximumTimeStart: now + 3600,
				Duration:         3600,
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

				result := res.findEarliestAvailableTime(&tt.params)
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

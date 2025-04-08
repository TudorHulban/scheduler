package scheduler

type paramsFindAvailableTime struct {
	TimeStart             int64
	MaximumTimeStart      int64
	SecondsDuration       int64
	SecondsOffsetTask     int64
	SecondsOffsetLocation int64

	IsLatest bool
}

// findAvailableTime returns task time not resource time.
func (res *Resource) findAvailableTime(params *paramsFindAvailableTime) int64 {
	if params.TimeStart > params.MaximumTimeStart {
		return _NoAvailability
	}

	offsetDifference := params.SecondsOffsetTask - params.SecondsOffsetLocation

	intervals, available := res.GetAvailability(
		&TimeInterval{
			TimeStart: params.TimeStart + offsetDifference,
			TimeEnd:   params.MaximumTimeStart + offsetDifference + params.SecondsDuration,
		},
	)
	if available {
		return params.TimeStart // Immediate availability
	}

	if len(intervals) == 0 {
		return _NoAvailability
	}

	if params.IsLatest {
		for i := len(intervals) - 1; i >= 0; i-- {
			interval := intervals[i]

			if interval.TimeEnd-interval.TimeStart >= params.SecondsDuration {
				startTaskTime := min(
					interval.TimeEnd-offsetDifference-params.SecondsDuration,
					params.MaximumTimeStart,
				)

				if startTaskTime >= params.TimeStart && startTaskTime <= params.MaximumTimeStart {
					return startTaskTime
				}
			}
		}
	}

	for _, interval := range intervals {
		if interval.TimeEnd-interval.TimeStart >= params.SecondsDuration {
			startTaskTime := interval.TimeStart - offsetDifference

			if startTaskTime >= params.TimeStart && startTaskTime <= params.MaximumTimeStart {
				if params.IsLatest {
					continue // Skip to find latest
				}

				return startTaskTime
			}
		}
	}

	return _NoAvailability
}

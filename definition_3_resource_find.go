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
	currentUTCStart := params.TimeStart + offsetDifference
	maxUTCStart := params.MaximumTimeStart + offsetDifference

	for currentUTCStart <= maxUTCStart {
		currentUTCEnd := currentUTCStart + params.SecondsDuration
		intervals, available := res.GetAvailability(
			&TimeInterval{
				TimeStart:     currentUTCStart,
				TimeEnd:       currentUTCEnd,
				SecondsOffset: offsetDifference,
			},
		)
		if available {
			return currentUTCStart - offsetDifference
		}
		if len(intervals) > 0 {
			nextUTCStart := intervals[0].TimeStart
			if params.IsLatest {
				nextUTCStart = intervals[len(intervals)-1].TimeStart
			}
			if nextUTCStart > currentUTCStart && nextUTCStart <= maxUTCStart {
				currentUTCStart = nextUTCStart
				continue
			}
		}
		// Find next start based on strategy
		var nextBusyEnd int64
		if params.IsLatest {
			// Latest: Jump to the furthest busy end
			for interval := range res.schedule {
				intervalEnd := interval.GetUTCTimeEnd()
				if intervalEnd > currentUTCStart && intervalEnd > nextBusyEnd {
					nextBusyEnd = intervalEnd
				}
			}
		} else {
			// Earliest: Jump to the earliest end of overlapping intervals
			nextBusyEnd = maxUTCStart + 1 // Default to beyond max
			for interval := range res.schedule {
				intervalStart := interval.GetUTCTimeStart()
				intervalEnd := interval.GetUTCTimeEnd()
				if intervalStart <= currentUTCEnd && intervalEnd > currentUTCStart {
					if intervalEnd < nextBusyEnd {
						nextBusyEnd = intervalEnd
					}
				}
			}
		}
		if nextBusyEnd > currentUTCStart && nextBusyEnd <= maxUTCStart {
			currentUTCStart = nextBusyEnd
		} else {
			currentUTCStart += params.SecondsDuration
		}
	}
	return _NoAvailability
}

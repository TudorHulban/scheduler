package scheduler

import "sort"

// GetAvailability returns:
//   - (nil, true)   = Fully available (no busy intervals or no overlap)
//   - (slots, false) = Partially available (returns available time slots)
//   - (nil, false)  = Completely unavailable (requested interval is fully booked)
func (res *ResourceScheduled) GetAvailability(searchInterval *TimeInterval) ([]TimeInterval, bool) {
	var busyUTCIntervals []TimeInterval

	for scheduledInterval := range res.schedule {
		busyUTCIntervals = append(
			busyUTCIntervals,
			TimeInterval{
				TimeStart:     scheduledInterval.GetUTCTimeStart(),
				TimeEnd:       scheduledInterval.GetUTCTimeEnd(),
				SecondsOffset: scheduledInterval.SecondsOffset,
			},
		)
	}

	sort.Slice(
		busyUTCIntervals,
		func(i, j int) bool {
			return busyUTCIntervals[i].TimeStart < busyUTCIntervals[j].TimeStart
		},
	)

	var availableIntervals []TimeInterval

	currentStart := searchInterval.GetUTCTimeStart()
	searchEnd := searchInterval.GetUTCTimeEnd()

	// Check if any busy interval overlaps
	hasOverlap := false

	for _, busy := range busyUTCIntervals {
		if busy.TimeEnd <= currentStart {
			continue
		}

		if busy.TimeStart >= searchEnd {
			break
		}

		hasOverlap = true

		if busy.TimeStart > currentStart {
			availableIntervals = append(
				availableIntervals,
				TimeInterval{
					TimeStart:     currentStart + searchInterval.SecondsOffset,
					TimeEnd:       busy.TimeStart + searchInterval.SecondsOffset,
					SecondsOffset: searchInterval.SecondsOffset,
				},
			)
		}

		currentStart = max(currentStart, busy.TimeEnd)
	}

	if currentStart < searchEnd {
		availableIntervals = append(
			availableIntervals,
			TimeInterval{
				TimeStart:     currentStart + searchInterval.SecondsOffset,
				TimeEnd:       searchEnd + searchInterval.SecondsOffset,
				SecondsOffset: searchInterval.SecondsOffset,
			},
		)
	}

	if !hasOverlap {
		return nil,
			true // Fully available if no overlap
	}

	return availableIntervals,
		false
}

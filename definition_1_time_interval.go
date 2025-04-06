package scheduler

type TimeInterval struct {
	TimeStart     int64
	TimeEnd       int64
	SecondsOffset int64
}

func (interval *TimeInterval) GetUTCTimeStart() int64 {
	return interval.TimeStart - interval.SecondsOffset
}

func (interval *TimeInterval) GetUTCTimeEnd() int64 {
	return interval.TimeEnd - interval.SecondsOffset
}

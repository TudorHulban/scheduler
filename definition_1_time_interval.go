package scheduler

type TimeInterval struct {
	TimeStart int64
	TimeEnd   int64
	Offset    int64
}

func (interval *TimeInterval) GetUTCTimeStart() int64 {
	return interval.TimeEnd - interval.Offset
}

func (interval *TimeInterval) GetUTCTimeEnd() int64 {
	return interval.TimeEnd - interval.Offset
}

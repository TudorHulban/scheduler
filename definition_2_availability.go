package scheduler

type Availability struct {
	OverlapStart int64
	OverlapEnd   int64
}

func (a *Availability) IsAvailable() bool {
	return a.OverlapStart == 0 && a.OverlapEnd == 0
}

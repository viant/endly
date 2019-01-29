package http

import "sync"

type partialStressTrips struct {
	trips   []*stressTestTrip
	channel chan *stressTestTrip
	*sync.WaitGroup
	index int
}

func (t *partialStressTrips) append(trip *stressTestTrip) {
	if trip == nil {
		return
	}
	if t.index >= len(t.trips) {
		t.flush(len(t.trips))
		t.index = 0
	}
	t.trips[t.index] = trip
	t.index++
}

func (t *partialStressTrips) flush(count int) {
	for i := 0; i < count; i++ {
		if t.trips[i] == nil {
			continue
		}
		t.Add(1)
		t.channel <- t.trips[i]
	}
}

func newPartialStressTrips(capacity int, channel chan *stressTestTrip, waitGroup *sync.WaitGroup) *partialStressTrips {
	return &partialStressTrips{
		channel:   channel,
		WaitGroup: waitGroup,
		trips:     make([]*stressTestTrip, capacity),
	}
}

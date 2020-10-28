package http

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/bridge"
	"sync"
)

//HTTPServerTrips represents http trips
type HTTPServerTrips struct {
	BaseDirectory string
	Rotate        bool
	Trips         map[string]*HTTPResponses
	IndexKeys     []string
	Mutex         *sync.Mutex
}

func (t *HTTPServerTrips) loadTripsIfNeeded() error {
	if t.BaseDirectory != "" {
		t.Trips = make(map[string]*HTTPResponses)
		httpTrips, err := bridge.ReadRecordedHttpTrips(t.BaseDirectory)
		if err != nil {
			return err
		}
		if len(httpTrips) == 0 {
			return fmt.Errorf("http capautre directory was empty %v", t.BaseDirectory)
		}
		for _, trip := range httpTrips {
			key, err := buildKeyValue(t.IndexKeys, trip.Request)
			if err != nil {
				return fmt.Errorf("failed to build request key: %v, %v", trip.Request.URL, err)
			}

			if _, has := t.Trips[key]; !has {
				t.Trips[key] = &HTTPResponses{
					Request:   trip.Request,
					Responses: make([]*bridge.HttpResponse, 0),
				}
			}
			t.Trips[key].Responses = append(t.Trips[key].Responses, trip.Response)
		}
	}
	return nil
}

//Init initialises trips
func (t *HTTPServerTrips) Init() error {
	if t.Mutex == nil {
		t.Mutex = &sync.Mutex{}
	}
	err := t.loadTripsIfNeeded()
	if err != nil {
		return fmt.Errorf("failed to load trips: %w", err)
	}
	if len(t.Trips) == 0 {
		return errors.New("trips were empty")
	}
	return nil
}

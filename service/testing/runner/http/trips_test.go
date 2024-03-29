package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
)

func TestTrips_AddRequest(t *testing.T) {
	trip := newTrips()

	r1 := Request{
		URL:    "http://127.0.0.1:8766/send1",
		Method: "POST",
		Body:   "0123456789",
	}
	r2 := Request{
		When:   "content1-2",
		URL:    "http://127.0.0.1:8766/send2",
		Method: "POST",
		Body:   "xc",
	}

	assert.Nil(t, trip.addRequest(&r1))
	assert.Nil(t, trip.addRequest(&r2))
	assert.NotNil(t, trip[TripRequests])

	requests := toolbox.AsSlice(trip[TripRequests])
	assert.Equal(t, 2, len(requests))

	trip1 := toolbox.AsMap(requests[0])
	assert.Equal(t, r1.URL, trip1["URL"])

	trip2 := toolbox.AsMap(requests[1])
	assert.Equal(t, r2.URL, trip2["URL"])
}

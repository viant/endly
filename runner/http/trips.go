package http

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

// Identifier to access in context
const Trips = "trips"
const TripRequests = "requests"
const TripResponses = "responses"

// Internal structure for managing all requests and responses
type trips data.Map

// Create new HTTP trips
func newTrips() trips {
	t := make(map[string]interface{})
	t[TripRequests] = make([]map[string]interface{}, 0)
	t[TripResponses] = make([]map[string]interface{}, 0)
	return t
}

// Add HTTP Request to trips
func (t trips) addRequest(r *Request) error {
	return t.add(TripRequests, r)
}

// Add HTTP Response to trips
func (t trips) addResponse(r *Response) error {
	return t.add(TripResponses, r)
}

// Internal method to generically handle request and response
func (t trips) add(index string, r interface{}) error {
	if r == nil {
		return errors.New("http trip data to be added is nil")
	}
	// Converting data to standard endly structured data format and any additional processing
	var data = make(map[string]interface{}, 0)
	err := toolbox.DefaultConverter.AssignConverted(&data, r)
	if err != nil {
		return fmt.Errorf("error converting to http trip map:%v", err)
	}
	// Append new data to trip
	t[index] = append(toolbox.AsSlice(t[index]), data)
	return nil
}

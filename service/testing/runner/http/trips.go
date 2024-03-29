package http

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

// Identifier to access in context
const TripsKey = "httpTrips"
const TripRequests = "Request"
const TripResponses = "Response"
const TripData = "Data"

// Internal structure for managing all requests and responses
type Trips data.Map

// Create new HTTP Trips
func newTrips() Trips {
	t := make(map[string]interface{})
	t[TripRequests] = make([]map[string]interface{}, 0)
	t[TripResponses] = make([]map[string]interface{}, 0)
	return t
}

// addRequest add HTTP Request to Trips
func (t Trips) addRequest(request *Request) error {
	return t.add(TripRequests, request)
}

// Set sets trip data
func (t Trips) setData(data data.Map) {
	t[TripData] = data
}

// addResponse addd HTTP Response to Trips
func (t Trips) addResponse(response *Response) error {
	return t.add(TripResponses, response)
}

// Internal method to generically handle request and response
func (t Trips) add(index string, value interface{}) error {
	if value == nil {
		return errors.New("http trip data to be added is nil")
	}
	// Converting data to standard endly structured data format and any additional processing
	var data = make(map[string]interface{}, 0)
	err := toolbox.DefaultConverter.AssignConverted(&data, value)
	if err != nil {
		return fmt.Errorf("error converting to http trip map:%v", err)
	}
	// Append new data to trip
	t[index] = append(toolbox.AsSlice(t[index]), data)
	return nil
}

package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PubsubProxyEvent
type PubsubProxyEvent struct {
	Data       []byte `json:"data"`
	Attributes *Attributes
}


type Attributes struct {
	NotificationConfig string `json:"notificationConfig"`
	ObjectGeneration string `json:"objectGeneration"`
	ObjectId string  `json:"objectId"`
	BucketId string `json:"bucketId"`
	EventTime  *time.Time `json:"eventTime"`
	EventType string `json:"eventTime"`
	OverwroteGeneration  string `json:"overwroteGeneration"`
}


//Hello prints meta and events details
func Hello(ctx context.Context, event *PubsubProxyEvent) error {
	{
		JSON, _ := json.Marshal(event)
		fmt.Printf("META: %s\n", JSON)
	}
	return nil
}

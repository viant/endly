package logs

import "github.com/aws/aws-sdk-go/service/cloudwatchlogs"

//FilterLogEventMessagesInput represents fetch log request
type FilterLogEventMessagesInput struct {
	cloudwatchlogs.FilterLogEventsInput `yaml:",inline" json:",inline"`
	Exclude                             []string //exclude log event where message contains exclude fragment
	Include                             []string //include log event where message contains include fragment
}

//FilterLogEventMessagesOutput represents response
type FilterLogEventMessagesOutput struct {
	Messages           []interface{}
	SearchedLogStreams []*cloudwatchlogs.SearchedLogStream
}

type SetupLogGroupInput cloudwatchlogs.CreateLogGroupInput

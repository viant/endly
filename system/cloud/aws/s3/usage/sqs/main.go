package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
)

var logger = log.New(os.Stdout, "INFO ", log.Llongfile)

func handleEvent(ctx context.Context, sqsEvent events.SQSEvent) (err error) {
	if len(sqsEvent.Records) == 0 {
		return err
	}
	for _, record := range sqsEvent.Records {
		s3Event := &events.S3Event{}
		if err = json.Unmarshal([]byte(record.Body), s3Event); err != nil {
			return err
		}
		for _, record := range s3Event.Records {
			URL := fmt.Sprintf("s3://%s/%s", record.S3.Bucket.Name, record.S3.Object.Key)
			logger.Printf("got notification: %v", URL)
		}
	}
	return err
}



func main() {
	lambda.Start(handleEvent)
}

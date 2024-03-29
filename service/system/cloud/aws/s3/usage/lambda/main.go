package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"os"
)

var logger = log.New(os.Stdout, "INFO ", log.Llongfile)

func handleEvent(ctx context.Context, s3Event events.S3Event) {
	if len(s3Event.Records) == 0 {
		return
	}
	for _, record := range s3Event.Records {
		URL := fmt.Sprintf("s3://%s/%s", record.S3.Bucket.Name, record.S3.Object.Key)
		logger.Printf("got notification: %v", URL)
	}

}

func main() {
	lambda.Start(handleEvent)
}

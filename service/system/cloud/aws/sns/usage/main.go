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

func handleEvent(ctx context.Context, sqsEvent events.SNSEvent) (err error) {
	if len(sqsEvent.Records) == 0 {
		return err
	}
	for _, record := range sqsEvent.Records {
		fmt.Printf("%v\n", record.SNS.Message)
	}
	return err
}

func main() {
	lambda.Start(handleEvent)
}

package sqs

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	currentAccount = "${aws.accountID}"
	allActions     = "*"
)

//SetupPermissionInput creates a permission if it does not exists
type SetupPermissionInput struct {
	sqs.AddPermissionInput
	Everybody bool
	Queue     string
	SourceArn string
}

func (i *SetupPermissionInput) Init() error {
	if i.Label == nil {
		source := ""
		if ARN, err := arn.Parse(i.SourceArn); err == nil {
			source = ARN.Resource + "-"
		}
		i.Label = aws.String(fmt.Sprintf("%s%vPermission", source, i.Queue))
	}
	if len(i.AWSAccountIds) == 0 {
		i.AWSAccountIds = []*string{aws.String(currentAccount)}
	}

	if len(i.Actions) == 0 {
		i.Actions = []*string{aws.String(allActions)}
	}
	return nil

}

func (i *SetupPermissionInput) Validate() error {
	if i.Queue == "" && i.QueueUrl == nil {
		return errors.New("queue was empty")
	}
	return nil
}

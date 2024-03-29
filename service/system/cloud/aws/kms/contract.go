package kms

import (
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"
)

// SetupKeyInput setup key with alias
type SetupKeyInput struct {
	kms.CreateKeyInput   `json:",inline" yaml:",inline"`
	kms.CreateAliasInput `json:",inline" yaml:",inline"`
}

func (i *SetupKeyInput) Validate() error {
	if i.AliasName == nil {
		return errors.New("alias was empty")
	}
	return nil
}

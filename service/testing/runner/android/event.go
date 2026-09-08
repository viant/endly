package android

import (
	"fmt"

	"github.com/viant/endly/model/msg"
)

func (r *DoctorResponse) Messages() []*msg.Message {
	status := msg.MessageStyleSuccess
	if !r.Ready {
		status = msg.MessageStyleError
	}
	result := []*msg.Message{
		msg.NewMessage(
			msg.NewStyled("Android runner", msg.MessageStyleGeneric),
			msg.NewStyled(fmt.Sprintf("ready=%t sdk=%s devices=%d avds=%d", r.Ready, r.AndroidSDKRoot, len(r.Devices), len(r.AVDs)), status),
		),
	}
	for _, check := range r.Checks {
		style := msg.MessageStyleSuccess
		if check.Status != "ok" {
			style = msg.MessageStyleError
		}
		detail := check.Version
		if detail == "" {
			detail = check.Detail
		}
		result = append(result, msg.NewMessage(
			msg.NewStyled(check.Name, msg.MessageStyleGeneric),
			msg.NewStyled(check.Status, style),
			msg.NewStyled(detail, msg.MessageStyleOutput),
			msg.NewStyled(check.Remediation, msg.MessageStyleGeneric),
		))
	}
	return result
}

func (r *DoctorResponse) IsOutput() bool { return true }

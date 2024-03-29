package iam

import (
	"encoding/json"
	"net/url"
)

func getPolicyDocument(document string) (*PolicyDocument, error) {
	policyDocument := &PolicyDocument{}
	document, _ = url.QueryUnescape(document)
	return policyDocument, json.Unmarshal([]byte(document), &policyDocument)
}

func getPolicies(attachedList []*Policy) []*PolicyEvenInfo {
	if len(attachedList) == 0 {
		return nil
	}
	var result = make([]*PolicyEvenInfo, 0)
	for _, attached := range attachedList {
		result = append(result, &PolicyEvenInfo{
			Policy:   attached.PolicyName,
			Arn:      attached.PolicyArn,
			Document: attached.PolicyInfo(),
		})
	}
	return result
}

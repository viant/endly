package sns

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws/iam"
	"github.com/viant/toolbox"
	"os"
	"path"
	"testing"
)

func Test_Policy(t *testing.T) {

	policyData := `{"Version":"2008-10-17","Id":"__default_policy_ID","Statement":[{"Sid":"__default_statement_ID","Effect":"Allow","Principal":{"AWS":"*"},"Action":["SNS:GetTopicAttributes","SNS:SetTopicAttributes","SNS:AddPermission","SNS:RemovePermission","SNS:DeleteTopic","SNS:Subscribe","SNS:ListSubscriptionsByTopic","SNS:Publish","SNS:Receive"],"Resource":"arn:aws:sns:us-west-1:458197927229:ms-dataflowStorageMirrorTopic","Condition":{"StringEquals":{"AWS:SourceOwner":"458197927229"}}},{"Sid":"ms-dataflowStorageMirrorTopicPermission","Effect":"Allow","Principal":{"AWS":"arn:aws:iam::458197927229:root"},"Action":"SNS:publish","Resource":"arn:aws:sns:us-west-1:458197927229:ms-dataflowStorageMirrorTopic"}]}`
	policy := &iam.PolicyDocument{}
	err := json.Unmarshal([]byte(policyData), policy)
	assert.Nil(t, err)
	toolbox.DumpIndent(policy, true)
	data, err := json.Marshal(policy)
	assert.Nil(t, err)
	fmt.Printf("%s\n", data)
}

func TestClient(t *testing.T) {
	context := endly.New().NewContext(nil)
	err := setClient(context, map[string]interface{}{
		"Credentials": "4234234dasdasde",
	})
	assert.NotNil(t, err)
	_, err = getClient(context)
	assert.NotNil(t, err)
	if !toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/aws.json")) {
		return
	}

	err = setClient(context, map[string]interface{}{
		"Credentials": "aws",
	})
	assert.Nil(t, err)
	client, err := getClient(context)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	_, ok := client.(*sns.SNS)
	assert.True(t, ok)
}

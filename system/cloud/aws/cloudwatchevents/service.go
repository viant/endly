package cloudwatchevents

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	ciam "github.com/viant/endly/system/cloud/aws/iam"
	"log"
)

const (
	//ServiceID aws Cloudwatch service id.
	ServiceID = "aws/cloudwatchevents"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func getRuleTargets(context *endly.Context, name string) ([]*cloudwatchevents.Target, error) {
	var result = make([]*cloudwatchevents.Target, 0)
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	var pageToken *string
	for {
		output, err := client.ListTargetsByRule(&cloudwatchevents.ListTargetsByRuleInput{
			Rule: &name,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, output.Targets...)
		pageToken = output.NextToken
		if pageToken == nil {
			break
		}
	}
	return result, nil
}

func getRule(context *endly.Context, name string) (*cloudwatchevents.Rule, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	var pageToken *string
	for {
		output, err := client.ListRules(&cloudwatchevents.ListRulesInput{NextToken: pageToken})
		if err != nil {
			return nil, err
		}
		for _, candidate := range output.Rules {
			if *candidate.Name == name {
				return candidate, nil
			}
		}
		pageToken = output.NextToken
		if pageToken == nil {
			break
		}
	}
	return nil, nil
}

func (s *service) deleteRuleTargets(context *endly.Context, rule string) error {
	targets, _ := getRuleTargets(context, rule)
	if len(targets) == 0 {
		return nil
	}
	client, err := GetClient(context)
	if err != nil {
		return err
	}
	var ids = make([]*string, 0)
	for _, target := range targets {
		ids = append(ids, target.Id)
	}
	_, err = client.RemoveTargets(&cloudwatchevents.RemoveTargetsInput{
		Rule: &rule,
		Ids:  ids,
	})
	return err
}

func (s *service) deployRule(context *endly.Context, input *DeployRuleInput) (*DeployRuleOutput, error) {
	var output = &DeployRuleOutput{}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	rule, err := getRule(context, *input.Name)
	if err != nil {
		return nil, err
	}
	if input.RoleName != nil {
		output.RoleInfo = &ciam.GetRoleInfoOutput{}
		if err = endly.Run(context, &input.SetupRolePolicyInput, &output.RoleInfo); err != nil {
			return nil, err
		}
	}
	if err = initTargets(context, input); err != nil {
		return nil, err
	}

	if hasRuleChanged(rule, &input.PutRuleInput) {
		if _, err = client.PutRule(&input.PutRuleInput); err != nil {
			return nil, err
		}
		rule, err = getRule(context, *input.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to PutRule: %s", input)
		}
	}
	output.Rule, err = getRule(context, *input.Name)
	if err != nil {
		return nil, err
	}
	targets, _ := getRuleTargets(context, *input.Name)
	if input.Targets.hasChanged(targets) {
		if err = s.deleteRuleTargets(context, *input.Name); err != nil {
			return nil, err
		}
		putInput := &cloudwatchevents.PutTargetsInput{
			Rule:    input.Name,
			Targets: input.Targets.targets(),
		}
		if _, err = client.PutTargets(putInput); err != nil {
			return nil, errors.Wrapf(err, "failed to PutTargets: %s", putInput)
		}
	}
	output.Targets, _ = getRuleTargets(context, *input.Name)
	return output, nil
}

func initTargets(context *endly.Context, input *DeployRuleInput) error {
	if len(input.Targets) == 0 {
		return nil
	}
	for i, target := range input.Targets {
		if target.Function != nil && target.Arn == nil {
			configuration, err := aws.GetFunctionConfiguration(context, *target.Function)
			if err != nil {
				return err
			}
			input.Targets[i].Arn = configuration.FunctionArn
		}
		if target.Id == nil {
			UUID, err := aws.NextID()
			if err != nil {
				return err
			}
			input.Targets[i].Id = &UUID
		}
	}
	return nil
}

func (s *service) getRule(context *endly.Context, input *GetRuleInput) (*GetRuleOutput, error) {
	output := &GetRuleOutput{}
	var err error
	if output.Rule, err = getRule(context, *input.Name); err != nil {
		return nil, err
	}
	if output.Rule != nil && output.Rule.RoleArn != nil {
		ruleName, _ := aws.ArnName(*output.Rule.RoleArn)
		output.RoleInfo = &ciam.GetRoleInfoOutput{}
		if err = endly.Run(context, &ciam.GetRoleInfoInput{
			RoleName: &ruleName,
		}, &output.RoleInfo); err != nil {
			return nil, err
		}
	}
	output.Targets, _ = getRuleTargets(context, *input.Name)
	return output, nil
}

func (s *service) getRuleNamesByTarget(context *endly.Context, targetARN *string) ([]*string, error) {
	var result = make([]*string, 0)
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	var pageToken *string
	for {
		output, err := client.ListRuleNamesByTarget(&cloudwatchevents.ListRuleNamesByTargetInput{
			TargetArn: targetARN,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, output.RuleNames...)
		pageToken = output.NextToken
		if pageToken == nil {
			break
		}

	}
	return result, nil
}

func (s *service) deleteRule(context *endly.Context, name *string) (*cloudwatchevents.DeleteRuleOutput, error) {
	output := &cloudwatchevents.DeleteRuleOutput{}
	if name == nil {
		return output, nil
	}
	err := s.deleteRuleTargets(context, *name)
	if err != nil {
		return nil, err
	}

	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	req := &cloudwatchevents.DeleteRuleInput{
		Name: name,
	}
	return client.DeleteRule(req)
}

func (s *service) DeleteRule(context *endly.Context, input *DeleteRuleInput) (*cloudwatchevents.DeleteRuleOutput, error) {
	output := &cloudwatchevents.DeleteRuleOutput{}
	if input.TargetArn != nil {
		names, err := s.getRuleNamesByTarget(context, input.TargetArn)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			output, err = s.deleteRule(context, name)
			if err != nil {
				return nil, err
			}
		}
	}
	if input.Name != nil {
		return s.deleteRule(context, input.Name)
	}
	return output, nil
}

func (s *service) registerRoutes() {
	client := &cloudwatchevents.CloudWatchEvents{}
	routes, err := aws.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = setClient
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action: "deployRule",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "deployRule", &DeployRuleInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &DeployRuleOutput{}),
		},
		RequestProvider: func() interface{} {
			return &DeployRuleInput{}
		},
		ResponseProvider: func() interface{} {
			return &DeployRuleOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeployRuleInput); ok {
				output, err := s.deployRule(context, req)
				if err == nil {
					context.Publish(aws.NewOutputEvent("...", "deployRule", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getRule",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getRule", &GetRuleInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetRuleOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetRuleInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetRuleOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetRuleInput); ok {
				output, err := s.getRule(context, req)
				if err == nil {
					context.Publish(aws.NewOutputEvent("...", "getRule", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "DeleteRule",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "DeleteRule", &DeleteRuleInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &cloudwatchevents.DeleteRuleOutput{}),
		},
		RequestProvider: func() interface{} {
			return &DeleteRuleInput{}
		},
		ResponseProvider: func() interface{} {
			return &cloudwatchevents.DeleteRuleOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeleteRuleInput); ok {
				output, err := s.DeleteRule(context, req)
				if err == nil {
					context.Publish(aws.NewOutputEvent("...", "DeleteRule", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// New creates a new AWS Cloudwatch events service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

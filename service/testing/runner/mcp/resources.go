package mcp

import (
	"context"
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/mcp/client"
)

type ListResourcesRequest struct {
	Request
	Cursor *string `json:"cursor,omitempty" yaml:"cursor,omitempty"`
}
type ListSkillsRequest struct {
	Request
	Cursor *string `json:"cursor,omitempty" yaml:"cursor,omitempty"`
}
type ReadResourceRequest struct {
	Request
	URI                string `json:"uri" yaml:"uri"`
	AllowProtocolError bool   `json:"allowProtocolError,omitempty" yaml:"allowProtocolError,omitempty"`
}
type GetSkillRequest struct {
	Request
	URI                string `json:"uri" yaml:"uri"`
	AllowProtocolError bool   `json:"allowProtocolError,omitempty" yaml:"allowProtocolError,omitempty"`
}

func (r *ReadResourceRequest) Validate() error {
	if err := r.Request.Validate(); err != nil {
		return err
	}
	if r.URI == "" {
		return fmt.Errorf("URI is required")
	}
	return nil
}
func (r *GetSkillRequest) Validate() error {
	return (&ReadResourceRequest{Request: r.Request, URI: r.URI}).Validate()
}

type ListResourcesResponse struct {
	Resources  []schema.Resource
	NextCursor *string
}
type ListSkillsResponse struct {
	Skills     []schema.Skill
	NextCursor *string
}
type ReadResourceResponse struct {
	Contents          []schema.ReadResourceResultContentsElem
	IsError           bool
	ProtocolErrorCode *int `json:",omitempty"`
}
type GetSkillResponse struct {
	Skill             *schema.Skill `json:",omitempty"`
	IsError           bool
	ProtocolErrorCode *int `json:",omitempty"`
}

func (s *service) registerResourceRoutes() {
	s.Register(&endly.Route{Action: "listResources", RequestInfo: &endly.ActionInfo{Description: "List native MCP resources without fetching contents"}, RequestProvider: func() interface{} { return &ListResourcesRequest{} }, ResponseProvider: func() interface{} { return &ListResourcesResponse{} }, Handler: func(ctx *endly.Context, v interface{}) (interface{}, error) {
		return s.listResources(ctx, v.(*ListResourcesRequest))
	}})
	s.Register(&endly.Route{Action: "readResource", RequestInfo: &endly.ActionInfo{Description: "Read one native MCP resource"}, RequestProvider: func() interface{} { return &ReadResourceRequest{} }, ResponseProvider: func() interface{} { return &ReadResourceResponse{} }, Handler: func(ctx *endly.Context, v interface{}) (interface{}, error) {
		return s.readResource(ctx, v.(*ReadResourceRequest))
	}})
	s.Register(&endly.Route{Action: "listSkills", RequestInfo: &endly.ActionInfo{Description: "List native MCP skills without fetching contents"}, RequestProvider: func() interface{} { return &ListSkillsRequest{} }, ResponseProvider: func() interface{} { return &ListSkillsResponse{} }, Handler: func(ctx *endly.Context, v interface{}) (interface{}, error) {
		return s.listSkills(ctx, v.(*ListSkillsRequest))
	}})
	s.Register(&endly.Route{Action: "getSkill", RequestInfo: &endly.ActionInfo{Description: "Read one native MCP skill"}, RequestProvider: func() interface{} { return &GetSkillRequest{} }, ResponseProvider: func() interface{} { return &GetSkillResponse{} }, Handler: func(ctx *endly.Context, v interface{}) (interface{}, error) {
		return s.getSkill(ctx, v.(*GetSkillRequest))
	}})
}
func (s *service) listResources(ctx *endly.Context, r *ListResourcesRequest) (*ListResourcesResponse, error) {
	var response *ListResourcesResponse
	err := s.withClient(ctx, &r.Request, func(op context.Context, c *client.Client) error {
		v, err := c.ListResources(op, r.Cursor)
		if err != nil {
			return safeError("MCP listResources failed", op, err)
		}
		response = &ListResourcesResponse{Resources: v.Resources, NextCursor: v.NextCursor}
		return nil
	})
	return response, err
}
func (s *service) listSkills(ctx *endly.Context, r *ListSkillsRequest) (*ListSkillsResponse, error) {
	var response *ListSkillsResponse
	err := s.withClient(ctx, &r.Request, func(op context.Context, c *client.Client) error {
		v, err := c.ListSkills(op, r.Cursor)
		if err != nil {
			return safeError("MCP listSkills failed", op, err)
		}
		response = &ListSkillsResponse{Skills: v.Skills, NextCursor: v.NextCursor}
		return nil
	})
	return response, err
}
func protocolCode(ctx context.Context, err error, allow bool) *int {
	var rpc *jsonrpc.Error
	if allow && ctx.Err() == nil && errors.As(err, &rpc) {
		code := rpc.Code
		return &code
	}
	return nil
}
func (s *service) readResource(ctx *endly.Context, r *ReadResourceRequest) (*ReadResourceResponse, error) {
	var response *ReadResourceResponse
	err := s.withClient(ctx, &r.Request, func(op context.Context, c *client.Client) error {
		v, err := c.ReadResource(op, &schema.ReadResourceRequestParams{Uri: r.URI})
		if err != nil {
			if code := protocolCode(op, err, r.AllowProtocolError); code != nil {
				response = &ReadResourceResponse{IsError: true, ProtocolErrorCode: code}
				return nil
			}
			return safeError("MCP readResource failed", op, err)
		}
		response = &ReadResourceResponse{Contents: v.Contents}
		return nil
	})
	return response, err
}
func (s *service) getSkill(ctx *endly.Context, r *GetSkillRequest) (*GetSkillResponse, error) {
	var response *GetSkillResponse
	err := s.withClient(ctx, &r.Request, func(op context.Context, c *client.Client) error {
		v, err := c.GetSkill(op, r.URI)
		if err != nil {
			if code := protocolCode(op, err, r.AllowProtocolError); code != nil {
				response = &GetSkillResponse{IsError: true, ProtocolErrorCode: code}
				return nil
			}
			return safeError("MCP getSkill failed", op, err)
		}
		response = &GetSkillResponse{Skill: &v.Skill}
		return nil
	})
	return response, err
}

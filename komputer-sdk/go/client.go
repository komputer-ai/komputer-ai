// Package client provides a convenience wrapper for the komputer.ai API.
//
// Hand-maintained convenience wrapper. Scaffolded by generate_client.py.
//
// Quick start:
//
//	client := client.New("http://localhost:8080")
//	agent, _, err := client.CreateAgent(ctx, "my-agent", "Say hello",
//	    client.CreateAgentOpts{Model: client.PtrString("claude-sonnet-4-6")})
package client

import (
	"context"
	"net/http"

	komputer "github.com/komputer-ai/komputer-ai/komputer-sdk/go/internal"
)

// Client wraps the generated komputer API client with convenience methods.
type Client struct {
	api     *komputer.APIClient
	baseURL string
}

// New creates a new Client for the given base URL.
func New(baseURL string) *Client {
	cfg := komputer.NewConfiguration()
	cfg.Servers = komputer.ServerConfigurations{
		{URL: baseURL + "/api/v1"},
	}
	return &Client{api: komputer.NewAPIClient(cfg), baseURL: baseURL}
}

// --- Agents ---

func (c *Client) ListAgents(ctx context.Context) (*komputer.AgentListResponse, *http.Response, error) {
	return c.api.AgentsAPI.ListAgents(ctx).Execute()
}

type CreateAgentOpts struct {
	Connectors []string
	Lifecycle *string
	Memories []string
	Model *string
	Namespace *string
	OfficeManager *string
	Role *string
	SecretRefs []string
	Skills []string
	SystemPrompt *string
	TemplateRef *string
}

func (c *Client) CreateAgent(ctx context.Context, name string, instructions string, opts ...CreateAgentOpts) (*komputer.AgentResponse, *http.Response, error) {
	req := komputer.CreateAgentRequest{
		Name: name,
		Instructions: instructions,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Connectors != nil {
			req.Connectors = o.Connectors
		}
		if o.Lifecycle != nil {
			req.Lifecycle = o.Lifecycle
		}
		if o.Memories != nil {
			req.Memories = o.Memories
		}
		if o.Model != nil {
			req.Model = o.Model
		}
		if o.Namespace != nil {
			req.Namespace = o.Namespace
		}
		if o.OfficeManager != nil {
			req.OfficeManager = o.OfficeManager
		}
		if o.Role != nil {
			req.Role = o.Role
		}
		if o.SecretRefs != nil {
			req.SecretRefs = o.SecretRefs
		}
		if o.Skills != nil {
			req.Skills = o.Skills
		}
		if o.SystemPrompt != nil {
			req.SystemPrompt = o.SystemPrompt
		}
		if o.TemplateRef != nil {
			req.TemplateRef = o.TemplateRef
		}
	}
	resp, httpResp, err := c.api.AgentsAPI.CreateAgent(ctx).Request(req).Execute()
	if httpResp != nil && httpResp.StatusCode == 409 {
		var patchOpts PatchAgentOpts
		if len(opts) > 0 {
			o := opts[0]
			patchOpts = PatchAgentOpts{
				Connectors:   o.Connectors,
				Instructions: &instructions,
				Lifecycle:    o.Lifecycle,
				Memories:     o.Memories,
				Model:        o.Model,
				SecretRefs:   o.SecretRefs,
				Skills:       o.Skills,
				SystemPrompt: o.SystemPrompt,
				TemplateRef:  o.TemplateRef,
			}
		} else {
			patchOpts = PatchAgentOpts{Instructions: &instructions}
		}
		return c.PatchAgent(ctx, name, patchOpts)
	}
	return resp, httpResp, err
}

func (c *Client) GetAgent(ctx context.Context, name string) (*komputer.AgentResponse, *http.Response, error) {
	return c.api.AgentsAPI.GetAgent(ctx, name).Execute()
}

type PatchAgentOpts struct {
	Connectors []string
	Instructions *string
	Lifecycle *string
	Memories []string
	Model *string
	SecretRefs []string
	Skills []string
	SystemPrompt *string
	TemplateRef *string
}

func (c *Client) PatchAgent(ctx context.Context, name string, opts ...PatchAgentOpts) (*komputer.AgentResponse, *http.Response, error) {
	req := komputer.PatchAgentRequest{
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Connectors != nil {
			req.Connectors = o.Connectors
		}
		if o.Instructions != nil {
			req.Instructions = o.Instructions
		}
		if o.Lifecycle != nil {
			req.Lifecycle = o.Lifecycle
		}
		if o.Memories != nil {
			req.Memories = o.Memories
		}
		if o.Model != nil {
			req.Model = o.Model
		}
		if o.SecretRefs != nil {
			req.SecretRefs = o.SecretRefs
		}
		if o.Skills != nil {
			req.Skills = o.Skills
		}
		if o.SystemPrompt != nil {
			req.SystemPrompt = o.SystemPrompt
		}
		if o.TemplateRef != nil {
			req.TemplateRef = o.TemplateRef
		}
	}
	return c.api.AgentsAPI.PatchAgent(ctx, name).Request(req).Execute()
}

func (c *Client) DeleteAgent(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.AgentsAPI.DeleteAgent(ctx, name).Execute()
}

func (c *Client) CancelAgentTask(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.AgentsAPI.CancelAgentTask(ctx, name).Execute()
}

func (c *Client) GetAgentEvents(ctx context.Context, name string) (map[string]interface{}, *http.Response, error) {
	return c.api.AgentsAPI.GetAgentEvents(ctx, name).Execute()
}

// --- Memories ---

func (c *Client) ListMemories(ctx context.Context) (map[string]interface{}, *http.Response, error) {
	return c.api.MemoriesAPI.ListMemories(ctx).Execute()
}

type CreateMemoryOpts struct {
	Description *string
	Namespace *string
}

func (c *Client) CreateMemory(ctx context.Context, name string, content string, opts ...CreateMemoryOpts) (*komputer.MemoryResponse, *http.Response, error) {
	req := komputer.CreateMemoryRequest{
		Name: name,
		Content: content,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Description != nil {
			req.Description = o.Description
		}
		if o.Namespace != nil {
			req.Namespace = o.Namespace
		}
	}
	resp, httpResp, err := c.api.MemoriesAPI.CreateMemory(ctx).Request(req).Execute()
	if httpResp != nil && httpResp.StatusCode == 409 {
		var patchOpts PatchMemoryOpts
		patchOpts.Content = &content
		if len(opts) > 0 {
			patchOpts.Description = opts[0].Description
		}
		return c.PatchMemory(ctx, name, patchOpts)
	}
	return resp, httpResp, err
}

func (c *Client) GetMemory(ctx context.Context, name string) (*komputer.MemoryResponse, *http.Response, error) {
	return c.api.MemoriesAPI.GetMemory(ctx, name).Execute()
}

type PatchMemoryOpts struct {
	Content *string
	Description *string
}

func (c *Client) PatchMemory(ctx context.Context, name string, opts ...PatchMemoryOpts) (*komputer.MemoryResponse, *http.Response, error) {
	req := komputer.PatchMemoryRequest{
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Content != nil {
			req.Content = o.Content
		}
		if o.Description != nil {
			req.Description = o.Description
		}
	}
	return c.api.MemoriesAPI.PatchMemory(ctx, name).Request(req).Execute()
}

func (c *Client) DeleteMemory(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.MemoriesAPI.DeleteMemory(ctx, name).Execute()
}

// --- Skills ---

func (c *Client) ListSkills(ctx context.Context) (map[string]interface{}, *http.Response, error) {
	return c.api.SkillsAPI.ListSkills(ctx).Execute()
}

type CreateSkillOpts struct {
	Namespace *string
}

func (c *Client) CreateSkill(ctx context.Context, name string, content string, description string, opts ...CreateSkillOpts) (*komputer.SkillResponse, *http.Response, error) {
	req := komputer.CreateSkillRequest{
		Name: name,
		Content: content,
		Description: description,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Namespace != nil {
			req.Namespace = o.Namespace
		}
	}
	resp, httpResp, err := c.api.SkillsAPI.CreateSkill(ctx).Request(req).Execute()
	if httpResp != nil && httpResp.StatusCode == 409 {
		return c.PatchSkill(ctx, name, PatchSkillOpts{
			Content:     &content,
			Description: &description,
		})
	}
	return resp, httpResp, err
}

func (c *Client) GetSkill(ctx context.Context, name string) (*komputer.SkillResponse, *http.Response, error) {
	return c.api.SkillsAPI.GetSkill(ctx, name).Execute()
}

type PatchSkillOpts struct {
	Content *string
	Description *string
}

func (c *Client) PatchSkill(ctx context.Context, name string, opts ...PatchSkillOpts) (*komputer.SkillResponse, *http.Response, error) {
	req := komputer.PatchSkillRequest{
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Content != nil {
			req.Content = o.Content
		}
		if o.Description != nil {
			req.Description = o.Description
		}
	}
	return c.api.SkillsAPI.PatchSkill(ctx, name).Request(req).Execute()
}

func (c *Client) DeleteSkill(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.SkillsAPI.DeleteSkill(ctx, name).Execute()
}

// --- Schedules ---

func (c *Client) ListSchedules(ctx context.Context) (*komputer.ScheduleListResponse, *http.Response, error) {
	return c.api.SchedulesAPI.ListSchedules(ctx).Execute()
}

type CreateScheduleOpts struct {
	Agent *komputer.CreateScheduleAgentSpec
	AgentName *string
	AutoDelete *bool
	KeepAgents *bool
	Namespace *string
	Timezone *string
}

func (c *Client) CreateSchedule(ctx context.Context, name string, instructions string, schedule string, opts ...CreateScheduleOpts) (*komputer.ScheduleResponse, *http.Response, error) {
	req := komputer.CreateScheduleRequest{
		Name: name,
		Instructions: instructions,
		Schedule: schedule,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Agent != nil {
			req.Agent = o.Agent
		}
		if o.AgentName != nil {
			req.AgentName = o.AgentName
		}
		if o.AutoDelete != nil {
			req.AutoDelete = o.AutoDelete
		}
		if o.KeepAgents != nil {
			req.KeepAgents = o.KeepAgents
		}
		if o.Namespace != nil {
			req.Namespace = o.Namespace
		}
		if o.Timezone != nil {
			req.Timezone = o.Timezone
		}
	}
	resp, httpResp, err := c.api.SchedulesAPI.CreateSchedule(ctx).Request(req).Execute()
	if httpResp != nil && httpResp.StatusCode == 409 {
		return c.PatchSchedule(ctx, name, PatchScheduleOpts{Schedule: &schedule})
	}
	return resp, httpResp, err
}

func (c *Client) GetSchedule(ctx context.Context, name string) (*komputer.ScheduleResponse, *http.Response, error) {
	return c.api.SchedulesAPI.GetSchedule(ctx, name).Execute()
}

type PatchScheduleOpts struct {
	Schedule *string
}

func (c *Client) PatchSchedule(ctx context.Context, name string, opts ...PatchScheduleOpts) (*komputer.ScheduleResponse, *http.Response, error) {
	req := komputer.PatchScheduleRequest{
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Schedule != nil {
			req.Schedule = o.Schedule
		}
	}
	return c.api.SchedulesAPI.PatchSchedule(ctx, name).Request(req).Execute()
}

func (c *Client) DeleteSchedule(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.SchedulesAPI.DeleteSchedule(ctx, name).Execute()
}

// --- Secrets ---

func (c *Client) ListSecrets(ctx context.Context) (*komputer.SecretListResponse, *http.Response, error) {
	return c.api.SecretsAPI.ListSecrets(ctx).Execute()
}

type CreateSecretOpts struct {
	Namespace *string
}

func (c *Client) CreateSecret(ctx context.Context, name string, data map[string]string, opts ...CreateSecretOpts) (*komputer.SecretResponse, *http.Response, error) {
	req := komputer.CreateSecretRequest{
		Name: name,
		Data: data,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Namespace != nil {
			req.Namespace = o.Namespace
		}
	}
	resp, httpResp, err := c.api.SecretsAPI.CreateSecret(ctx).Request(req).Execute()
	if httpResp != nil && httpResp.StatusCode == 409 {
		_, httpResp2, err2 := c.UpdateSecret(ctx, name, data)
		return nil, httpResp2, err2
	}
	return resp, httpResp, err
}

type UpdateSecretOpts struct {
	Namespace *string
}

func (c *Client) UpdateSecret(ctx context.Context, name string, data map[string]string, opts ...UpdateSecretOpts) (map[string]string, *http.Response, error) {
	req := komputer.UpdateSecretRequest{
		Data: data,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Namespace != nil {
			req.Namespace = o.Namespace
		}
	}
	return c.api.SecretsAPI.UpdateSecret(ctx, name).Request(req).Execute()
}

func (c *Client) DeleteSecret(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.SecretsAPI.DeleteSecret(ctx, name).Execute()
}

// --- Connectors ---

func (c *Client) ListConnectors(ctx context.Context) (map[string]interface{}, *http.Response, error) {
	return c.api.ConnectorsAPI.ListConnectors(ctx).Execute()
}

type CreateConnectorOpts struct {
	AuthSecretKey *string
	AuthSecretName *string
	AuthType *string
	DisplayName *string
	Namespace *string
	OauthClientId *string
	OauthClientSecret *string
	Type *string
}

func (c *Client) CreateConnector(ctx context.Context, name string, service string, url string, opts ...CreateConnectorOpts) (*komputer.ConnectorResponse, *http.Response, error) {
	req := komputer.CreateConnectorRequest{
		Name: name,
		Service: service,
		Url: url,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.AuthSecretKey != nil {
			req.AuthSecretKey = o.AuthSecretKey
		}
		if o.AuthSecretName != nil {
			req.AuthSecretName = o.AuthSecretName
		}
		if o.AuthType != nil {
			req.AuthType = o.AuthType
		}
		if o.DisplayName != nil {
			req.DisplayName = o.DisplayName
		}
		if o.Namespace != nil {
			req.Namespace = o.Namespace
		}
		if o.OauthClientId != nil {
			req.OauthClientId = o.OauthClientId
		}
		if o.OauthClientSecret != nil {
			req.OauthClientSecret = o.OauthClientSecret
		}
		if o.Type != nil {
			req.Type = o.Type
		}
	}
	return c.api.ConnectorsAPI.CreateConnector(ctx).Request(req).Execute()
}

func (c *Client) GetConnector(ctx context.Context, name string) (*komputer.ConnectorResponse, *http.Response, error) {
	return c.api.ConnectorsAPI.GetConnector(ctx, name).Execute()
}

func (c *Client) DeleteConnector(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.ConnectorsAPI.DeleteConnector(ctx, name).Execute()
}

func (c *Client) ListConnectorTools(ctx context.Context, name string) (map[string]interface{}, *http.Response, error) {
	return c.api.ConnectorsAPI.ListConnectorTools(ctx, name).Execute()
}

// --- Offices ---

func (c *Client) ListOffices(ctx context.Context) (*komputer.OfficeListResponse, *http.Response, error) {
	return c.api.OfficesAPI.ListOffices(ctx).Execute()
}

func (c *Client) GetOffice(ctx context.Context, name string) (*komputer.OfficeResponse, *http.Response, error) {
	return c.api.OfficesAPI.GetOffice(ctx, name).Execute()
}

func (c *Client) DeleteOffice(ctx context.Context, name string) (map[string]string, *http.Response, error) {
	return c.api.OfficesAPI.DeleteOffice(ctx, name).Execute()
}

func (c *Client) GetOfficeEvents(ctx context.Context, name string) (map[string]interface{}, *http.Response, error) {
	return c.api.OfficesAPI.GetOfficeEvents(ctx, name).Execute()
}

// --- Templates ---

func (c *Client) ListTemplates(ctx context.Context) (map[string]interface{}, *http.Response, error) {
	return c.api.TemplatesAPI.ListTemplates(ctx).Execute()
}


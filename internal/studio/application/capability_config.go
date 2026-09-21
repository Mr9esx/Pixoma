package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	platformcrypto "github.com/Mr9esx/Pixoma/internal/platform/crypto"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type CapabilityConfigRepository interface {
	CreateSkill(context.Context, *domain.Skill) error
	UpdateSkill(context.Context, *domain.Skill) error
	GetSkill(context.Context, string, string) (*domain.Skill, error)
	ListSkills(context.Context, string) ([]*domain.Skill, error)
	CreateMCPConnector(context.Context, *domain.MCPConnector) error
	UpdateMCPConnector(context.Context, *domain.MCPConnector) error
	GetMCPConnector(context.Context, string, string) (*domain.MCPConnector, error)
	ListMCPConnectors(context.Context, string) ([]*domain.MCPConnector, error)
	UpsertAgentWorkflowSetting(context.Context, *domain.AgentWorkflowSetting) error
	ListAgentWorkflowSettings(context.Context, string) ([]*domain.AgentWorkflowSetting, error)
}

type WorkflowCatalog interface {
	List(context.Context, catalogdomain.ListQuery) ([]*catalogdomain.Case, error)
}

// MCPConnectorProber performs the authenticated MCP handshake and returns only
// display-safe tool metadata. Credentials stay inside the infrastructure layer.
type MCPConnectorProber interface {
	Probe(ctx context.Context, endpoint, credential string) ([]domain.MCPTool, error)
}

type CapabilityConfigService struct {
	Repo            CapabilityConfigRepository
	EncryptionKey   []byte
	IDs             func() string
	Now             func() time.Time
	WorkflowCatalog WorkflowCatalog
	MCPProber       MCPConnectorProber
}

type CreateSkillInput struct {
	AccountID, Name, Description, Prompt string
	Enabled                              bool
}
type UpdateSkillInput struct {
	AccountID, SkillID, Name, Description, Prompt string
	Enabled                                       bool
}
type SkillView struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Prompt      string    `json:"prompt"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type CreateConnectorInput struct {
	AccountID, Name, URL, Credential string
	Enabled                          bool
	Policy                           domain.ConnectorPolicy
}
type UpdateConnectorInput struct {
	AccountID, ConnectorID, Name, URL, Credential string
	Enabled                                       bool
	Policy                                        domain.ConnectorPolicy
}
type MCPConnectorView struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	URL              string                 `json:"url"`
	Enabled          bool                   `json:"enabled"`
	Policy           domain.ConnectorPolicy `json:"policy"`
	CredentialMasked string                 `json:"credential_masked"`
	Tools            []domain.MCPTool       `json:"tools"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}
type AgentWorkflowView struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	WorkflowEnabled bool   `json:"workflow_enabled"`
	AgentEnabled    bool   `json:"agent_enabled"`
	Inputs          int    `json:"inputs"`
	Outputs         int    `json:"outputs"`
}

func (s *CapabilityConfigService) CreateSkill(ctx context.Context, input CreateSkillInput) (*SkillView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	skill, err := domain.NewSkill(s.nextID(), input.AccountID, input.Name, input.Description, input.Prompt, s.now())
	if err != nil {
		return nil, err
	}
	skill.Enabled = input.Enabled
	if err := s.Repo.CreateSkill(ctx, skill); err != nil {
		return nil, err
	}
	return skillView(skill), nil
}

func (s *CapabilityConfigService) ListSkills(ctx context.Context, accountID string) ([]*SkillView, error) {
	skills, err := s.Repo.ListSkills(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*SkillView, 0, len(skills))
	for _, skill := range skills {
		out = append(out, skillView(skill))
	}
	return out, nil
}

func (s *CapabilityConfigService) UpdateSkill(ctx context.Context, input UpdateSkillInput) (*SkillView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	existing, err := s.Repo.GetSkill(ctx, input.AccountID, input.SkillID)
	if err != nil {
		return nil, err
	}
	updated, err := domain.NewSkill(existing.ID, existing.AccountID, input.Name, input.Description, input.Prompt, s.now())
	if err != nil {
		return nil, err
	}
	updated.Enabled = input.Enabled
	updated.CreatedAt = existing.CreatedAt
	if err := s.Repo.UpdateSkill(ctx, updated); err != nil {
		return nil, err
	}
	return skillView(updated), nil
}

func (s *CapabilityConfigService) CreateConnector(ctx context.Context, input CreateConnectorInput) (*MCPConnectorView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	if strings.TrimSpace(input.Credential) == "" {
		return nil, fmt.Errorf("%w: connector credential is required", domain.ErrInvalid)
	}
	cipherText, err := platformcrypto.Encrypt(s.EncryptionKey, input.Credential)
	if err != nil {
		return nil, err
	}
	connector, err := domain.NewMCPConnector(s.nextID(), input.AccountID, input.Name, input.URL, cipherText, input.Policy, s.now())
	if err != nil {
		return nil, err
	}
	connector.Enabled = input.Enabled
	if err := s.Repo.CreateMCPConnector(ctx, connector); err != nil {
		return nil, err
	}
	return connectorView(connector), nil
}

func (s *CapabilityConfigService) ListConnectors(ctx context.Context, accountID string) ([]*MCPConnectorView, error) {
	connectors, err := s.Repo.ListMCPConnectors(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*MCPConnectorView, 0, len(connectors))
	for _, connector := range connectors {
		out = append(out, connectorView(connector))
	}
	return out, nil
}

func (s *CapabilityConfigService) UpdateConnector(ctx context.Context, input UpdateConnectorInput) (*MCPConnectorView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	existing, err := s.Repo.GetMCPConnector(ctx, input.AccountID, input.ConnectorID)
	if err != nil {
		return nil, err
	}
	cipherText := existing.CredentialCipher
	if strings.TrimSpace(input.Credential) != "" {
		cipherText, err = platformcrypto.Encrypt(s.EncryptionKey, input.Credential)
		if err != nil {
			return nil, err
		}
	}
	updated, err := domain.NewMCPConnector(existing.ID, existing.AccountID, input.Name, input.URL, cipherText, input.Policy, s.now())
	if err != nil {
		return nil, err
	}
	updated.Enabled = input.Enabled
	updated.CreatedAt = existing.CreatedAt
	updated.DiscoveredTools = existing.DiscoveredTools
	if err := s.Repo.UpdateMCPConnector(ctx, updated); err != nil {
		return nil, err
	}
	return connectorView(updated), nil
}

// ProbeConnector discovers the tools published by a configured Streamable HTTP
// endpoint. It is deliberately available while disabled so an administrator can
// verify a connector before making it available to Agent.
func (s *CapabilityConfigService) ProbeConnector(ctx context.Context, accountID, connectorID string) (*MCPConnectorView, error) {
	if s == nil || s.Repo == nil || s.MCPProber == nil {
		return nil, fmt.Errorf("studio: MCP connector prober is not configured")
	}
	connector, err := s.Repo.GetMCPConnector(ctx, accountID, connectorID)
	if err != nil {
		return nil, err
	}
	credential, err := platformcrypto.Decrypt(s.EncryptionKey, connector.CredentialCipher)
	if err != nil {
		return nil, err
	}
	probeContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tools, err := s.MCPProber.Probe(probeContext, connector.URL, credential)
	if err != nil {
		return nil, sanitizeConnectorProbeError(err, credential)
	}
	connector.DiscoveredTools = tools
	connector.UpdatedAt = s.now()
	if err := s.Repo.UpdateMCPConnector(ctx, connector); err != nil {
		return nil, err
	}
	return connectorView(connector), nil
}

func sanitizeConnectorProbeError(err error, credential string) error {
	message := strings.ReplaceAll(err.Error(), credential, "[REDACTED]")
	if len(message) > 512 {
		message = message[:512]
	}
	return fmt.Errorf("MCP connector probe failed: %s", message)
}

func (s *CapabilityConfigService) ListAgentWorkflows(ctx context.Context, accountID string) ([]*AgentWorkflowView, error) {
	if s == nil || s.Repo == nil || s.WorkflowCatalog == nil {
		return nil, fmt.Errorf("studio: workflow catalog is not configured")
	}
	cases, err := s.WorkflowCatalog.List(ctx, catalogdomain.ListQuery{})
	if err != nil {
		return nil, err
	}
	settings, err := s.Repo.ListAgentWorkflowSettings(ctx, accountID)
	if err != nil {
		return nil, err
	}
	enabled := make(map[string]bool, len(settings))
	for _, setting := range settings {
		enabled[setting.WorkflowID] = setting.AgentEnabled
	}
	workflows := make([]*AgentWorkflowView, 0, len(cases))
	for _, workflow := range cases {
		if workflow == nil {
			continue
		}
		id := strconv.FormatUint(uint64(workflow.Document.ID), 10)
		workflows = append(workflows, &AgentWorkflowView{ID: id, Name: workflow.Document.Name, Description: workflow.Document.Description, WorkflowEnabled: workflow.Enabled, AgentEnabled: enabled[id], Inputs: len(workflow.Document.Inputs), Outputs: len(workflow.Document.Outputs)})
	}
	return workflows, nil
}

func (s *CapabilityConfigService) SetAgentWorkflowEnabled(ctx context.Context, accountID, workflowID string, agentEnabled bool) (*AgentWorkflowView, error) {
	if s == nil || s.Repo == nil || s.WorkflowCatalog == nil {
		return nil, fmt.Errorf("studio: workflow catalog is not configured")
	}
	workflowID = strings.TrimSpace(workflowID)
	if accountID == "" || workflowID == "" {
		return nil, fmt.Errorf("%w: workflow id is required", domain.ErrInvalid)
	}
	workflows, err := s.ListAgentWorkflows(ctx, accountID)
	if err != nil {
		return nil, err
	}
	for _, workflow := range workflows {
		if workflow.ID != workflowID {
			continue
		}
		if err := s.Repo.UpsertAgentWorkflowSetting(ctx, &domain.AgentWorkflowSetting{AccountID: accountID, WorkflowID: workflowID, AgentEnabled: agentEnabled, UpdatedAt: s.now()}); err != nil {
			return nil, err
		}
		workflow.AgentEnabled = agentEnabled
		return workflow, nil
	}
	return nil, domain.ErrNotFound
}

func skillView(skill *domain.Skill) *SkillView {
	return &SkillView{ID: skill.ID, Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt, Enabled: skill.Enabled, CreatedAt: skill.CreatedAt, UpdatedAt: skill.UpdatedAt}
}
func connectorView(connector *domain.MCPConnector) *MCPConnectorView {
	return &MCPConnectorView{ID: connector.ID, Name: connector.Name, URL: connector.URL, Enabled: connector.Enabled, Policy: connector.Policy, CredentialMasked: "••••••••", Tools: connector.DiscoveredTools, CreatedAt: connector.CreatedAt, UpdatedAt: connector.UpdatedAt}
}
func (s *CapabilityConfigService) nextID() string {
	if s.IDs != nil {
		return s.IDs()
	}
	return uuid.NewString()
}
func (s *CapabilityConfigService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

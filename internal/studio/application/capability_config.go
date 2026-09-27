package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	UpdateSkillIfUnchanged(context.Context, *domain.Skill, time.Time, string) error
	SetSkillEnabled(context.Context, string, string, bool) error
	GetSkill(context.Context, string, string) (*domain.Skill, error)
	ListSkills(context.Context, string) ([]*domain.Skill, error)
	ListSkillSummaries(context.Context, string) ([]*domain.Skill, error)
	ListSkillVersions(context.Context, string, string) ([]*domain.SkillVersion, error)
	GetSkillVersion(context.Context, string, string, string) (*domain.SkillVersion, error)
	CreateMCPConnector(context.Context, *domain.MCPConnector) error
	UpdateMCPConnector(context.Context, *domain.MCPConnector) error
	GetMCPConnector(context.Context, string, string) (*domain.MCPConnector, error)
	ListMCPConnectors(context.Context, string) ([]*domain.MCPConnector, error)
	UpsertAgentWorkflowSetting(context.Context, *domain.AgentWorkflowSetting) error
	ListAgentWorkflowSettings(context.Context, string) ([]*domain.AgentWorkflowSetting, error)
}

type SkillCreator interface {
	CreateSkill(context.Context, CreateSkillInput) (*SkillView, error)
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

const (
	maxRuntimeMCPTools       = 100
	maxRuntimeToolDescBytes  = 8 << 10
	maxRuntimeToolSchemaByte = 64 << 10
)

type CreateSkillInput struct {
	AccountID, Name, Description, Prompt string
	Files                                []domain.SkillFile
	Enabled                              bool
	Version                              string `json:"version"`
}
type UpdateSkillInput struct {
	AccountID, SkillID, Name, Description, Prompt string
	Files                                         []domain.SkillFile
	Enabled                                       bool
	UpdatedAt                                     time.Time `json:"updated_at"`
	Version                                       string    `json:"version"`
}
type SkillSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type SkillView struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Prompt      string             `json:"prompt"`
	Files       []domain.SkillFile `json:"files,omitempty"`
	Version     string             `json:"version"`
	Enabled     bool               `json:"enabled"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}
type SkillVersionSummary struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}
type SkillVersionView struct {
	Version     string             `json:"version"`
	CreatedAt   time.Time          `json:"created_at"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Prompt      string             `json:"prompt"`
	Files       []domain.SkillFile `json:"files"`
}
type CreateConnectorInput struct {
	AccountID, Name, URL, Credential string
	Enabled                          bool
	Policy                           domain.ConnectorPolicy
}
type DiscoverConnectorInput struct {
	AccountID   string `json:"-"`
	ConnectorID string `json:"connector_id"`
	URL         string `json:"url"`
	Credential  string `json:"credential"`
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

// ResolvedMCPConnector is a short-lived, server-only connector view used by
// the Agent runtime. It must never be serialized into an HTTP response,
// persisted in a Run event, or logged because Credential is plaintext.
type ResolvedMCPConnector struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	URL        string                 `json:"url"`
	Credential string                 `json:"-"`
	Policy     domain.ConnectorPolicy `json:"policy"`
	Tools      []domain.MCPTool       `json:"tools"`
}
type AgentWorkflowView struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Description     string                      `json:"description"`
	WorkflowEnabled bool                        `json:"workflow_enabled"`
	AgentEnabled    bool                        `json:"agent_enabled"`
	Inputs          int                         `json:"inputs"`
	Outputs         int                         `json:"outputs"`
	Preview         string                      `json:"preview,omitempty"`
	InputSchema     map[string]any              `json:"input_schema"`
	InputFields     []domain.WorkflowInputField `json:"input_fields"`
}

func (s *CapabilityConfigService) CreateSkill(ctx context.Context, input CreateSkillInput) (*SkillView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	if len(input.Files) > 0 {
		metadata, prompt, err := ValidateSkillFiles(input.Files)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
		}
		input.Name, input.Description, input.Prompt = metadata.Name, metadata.Description, prompt
	}
	skill, err := domain.NewSkill(s.nextID(), input.AccountID, input.Name, input.Description, input.Prompt, s.now())
	if err != nil {
		return nil, err
	}
	skill.Enabled = input.Enabled
	skill.Files = input.Files
	if input.Version != "" {
		if !domain.ValidSkillVersion(input.Version) {
			return nil, fmt.Errorf("%w: invalid Skill version", domain.ErrInvalid)
		}
		skill.Version = input.Version
	}
	if err := s.Repo.CreateSkill(ctx, skill); err != nil {
		return nil, err
	}
	return s.GetSkill(ctx, input.AccountID, skill.ID)
}

func (s *CapabilityConfigService) ListSkills(ctx context.Context, accountID string) ([]*SkillSummary, error) {
	skills, err := s.Repo.ListSkillSummaries(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*SkillSummary, 0, len(skills))
	for _, skill := range skills {
		out = append(out, &SkillSummary{
			ID: skill.ID, Name: skill.Name, Description: skill.Description,
			Version: skill.Version, Enabled: skill.Enabled, CreatedAt: skill.CreatedAt, UpdatedAt: skill.UpdatedAt,
		})
	}
	return out, nil
}

func (s *CapabilityConfigService) GetSkill(ctx context.Context, accountID, skillID string) (*SkillView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	skill, err := s.Repo.GetSkill(ctx, accountID, skillID)
	if err != nil {
		return nil, err
	}
	return skillView(skill), nil
}

func (s *CapabilityConfigService) ListSkillVersions(ctx context.Context, accountID, skillID string) ([]*SkillVersionSummary, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	versions, err := s.Repo.ListSkillVersions(ctx, accountID, skillID)
	if err != nil {
		return nil, err
	}
	out := make([]*SkillVersionSummary, 0, len(versions))
	for _, version := range versions {
		out = append(out, &SkillVersionSummary{Version: version.Version, CreatedAt: version.CreatedAt})
	}
	return out, nil
}

func (s *CapabilityConfigService) GetSkillVersion(ctx context.Context, accountID, skillID, version string) (*SkillVersionView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	if !domain.ValidSkillVersion(version) {
		return nil, fmt.Errorf("%w: invalid Skill version", domain.ErrInvalid)
	}
	snapshot, err := s.Repo.GetSkillVersion(ctx, accountID, skillID, version)
	if err != nil {
		return nil, err
	}
	files := snapshot.Files
	if files == nil {
		files = []domain.SkillFile{}
	}
	return &SkillVersionView{Version: snapshot.Version, CreatedAt: snapshot.CreatedAt, Name: snapshot.Name, Description: snapshot.Description, Prompt: snapshot.Prompt, Files: files}, nil
}

func (s *CapabilityConfigService) SetSkillEnabled(ctx context.Context, accountID, skillID string, enabled bool) (*SkillView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	if err := s.Repo.SetSkillEnabled(ctx, accountID, skillID, enabled); err != nil {
		return nil, err
	}
	return s.GetSkill(ctx, accountID, skillID)
}

func (s *CapabilityConfigService) UpdateSkill(ctx context.Context, input UpdateSkillInput) (*SkillView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	existing, err := s.Repo.GetSkill(ctx, input.AccountID, input.SkillID)
	if err != nil {
		return nil, err
	}
	if input.UpdatedAt.IsZero() {
		return nil, fmt.Errorf("%w: updated_at is required", domain.ErrInvalid)
	}
	if !existing.UpdatedAt.Equal(input.UpdatedAt) {
		return nil, domain.ErrConflict
	}
	if !domain.ValidSkillVersion(input.Version) {
		return nil, fmt.Errorf("%w: invalid Skill version", domain.ErrInvalid)
	}
	if domain.CompareSkillVersions(input.Version, existing.Version) <= 0 {
		return nil, domain.ErrConflict
	}
	if input.Files == nil {
		input.Files = existing.Files
	}
	if len(input.Files) > 0 {
		metadata, prompt, err := ValidateSkillFiles(input.Files)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
		}
		input.Name, input.Description, input.Prompt = metadata.Name, metadata.Description, prompt
	}
	updated, err := domain.NewSkill(existing.ID, existing.AccountID, input.Name, input.Description, input.Prompt, s.now())
	if err != nil {
		return nil, err
	}
	updated.Enabled = existing.Enabled
	updated.Files = input.Files
	updated.Version = input.Version
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = updated.UpdatedAt.Truncate(time.Millisecond)
	if !updated.UpdatedAt.After(existing.UpdatedAt) {
		updated.UpdatedAt = existing.UpdatedAt.Truncate(time.Millisecond).Add(time.Millisecond)
	}
	if err := s.Repo.UpdateSkillIfUnchanged(ctx, updated, input.UpdatedAt, existing.Version); err != nil {
		return nil, err
	}
	return s.GetSkill(ctx, input.AccountID, input.SkillID)
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

// ResolveMCPConnectors returns only connectors that are available to Agent.
// A connector must be explicitly enabled, have an executable policy, and have
// completed tool discovery before its encrypted credential is resolved.
func (s *CapabilityConfigService) ResolveMCPConnectors(ctx context.Context, accountID string) ([]ResolvedMCPConnector, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: capability config service is not configured")
	}
	connectors, err := s.Repo.ListMCPConnectors(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("studio: list MCP connectors: %w", err)
	}
	resolved := make([]ResolvedMCPConnector, 0, len(connectors))
	for _, connector := range connectors {
		if connector == nil || !connector.Enabled || connector.Policy == domain.ConnectorPolicyForbidden {
			continue
		}
		tools := runtimeToolSnapshots(connector.DiscoveredTools)
		if len(tools) == 0 {
			continue
		}
		credential, err := platformcrypto.Decrypt(s.EncryptionKey, connector.CredentialCipher)
		if err != nil {
			return nil, fmt.Errorf("studio: decrypt MCP connector %s: %w", connector.ID, err)
		}
		resolved = append(resolved, ResolvedMCPConnector{
			ID: connector.ID, Name: connector.Name, URL: connector.URL,
			Credential: credential, Policy: connector.Policy,
			Tools: tools,
		})
	}
	return resolved, nil
}

func runtimeToolSnapshots(discovered []domain.MCPTool) []domain.MCPTool {
	tools := make([]domain.MCPTool, 0, len(discovered))
	for _, tool := range discovered {
		if len(tools) == maxRuntimeMCPTools {
			break
		}
		if strings.TrimSpace(tool.Name) == "" || len(tool.Description) > maxRuntimeToolDescBytes || len(tool.InputSchema) > maxRuntimeToolSchemaByte || !json.Valid(tool.InputSchema) {
			continue
		}
		tools = append(tools, domain.MCPTool{
			Name: tool.Name, Description: tool.Description,
			InputSchema: append([]byte(nil), tool.InputSchema...),
		})
	}
	return tools
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
	if updated.URL == existing.URL && strings.TrimSpace(input.Credential) == "" {
		updated.DiscoveredTools = existing.DiscoveredTools
	}
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
		connector.DiscoveredTools = nil
		connector.UpdatedAt = s.now()
		if updateErr := s.Repo.UpdateMCPConnector(ctx, connector); updateErr != nil {
			return nil, updateErr
		}
		return nil, sanitizeConnectorProbeError(err, credential)
	}
	connector.DiscoveredTools = tools
	connector.UpdatedAt = s.now()
	if err := s.Repo.UpdateMCPConnector(ctx, connector); err != nil {
		return nil, err
	}
	return connectorView(connector), nil
}

var ErrConnectorProbe = errors.New("studio: MCP connector discovery failed")

func (s *CapabilityConfigService) DiscoverConnector(ctx context.Context, input DiscoverConnectorInput) ([]domain.MCPTool, error) {
	if s == nil || s.Repo == nil || s.MCPProber == nil {
		return nil, fmt.Errorf("studio: MCP connector prober is not configured")
	}
	endpoint := strings.TrimSpace(input.URL)
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("%w: connector URL must be HTTP(S)", domain.ErrInvalid)
	}
	credential := input.Credential
	if strings.TrimSpace(credential) == "" {
		if input.ConnectorID == "" {
			return nil, fmt.Errorf("%w: connector credential is required", domain.ErrInvalid)
		}
		connector, err := s.Repo.GetMCPConnector(ctx, input.AccountID, input.ConnectorID)
		if err != nil {
			return nil, err
		}
		credential, err = platformcrypto.Decrypt(s.EncryptionKey, connector.CredentialCipher)
		if err != nil {
			return nil, err
		}
	}
	probeContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tools, err := s.MCPProber.Probe(probeContext, endpoint, credential)
	if err != nil {
		return nil, sanitizeConnectorProbeError(err, credential)
	}
	if tools == nil {
		return []domain.MCPTool{}, nil
	}
	return tools, nil
}

func sanitizeConnectorProbeError(err error, credential string) error {
	message := err.Error()
	if credential != "" {
		message = strings.ReplaceAll(message, credential, "[REDACTED]")
	}
	if len(message) > 512 {
		message = message[:512]
	}
	return fmt.Errorf("%w: %s", ErrConnectorProbe, message)
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
		workflows = append(workflows, &AgentWorkflowView{ID: id, Name: workflow.Document.Name, Description: workflow.Document.Description, WorkflowEnabled: workflow.Enabled, AgentEnabled: enabled[id], Inputs: len(workflow.Document.Inputs), Outputs: len(workflow.Document.Outputs), Preview: workflow.Document.Preview, InputSchema: workflow.Document.InputSchema, InputFields: workflowInputFields(workflow.Document.Inputs)})
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
	return &SkillView{ID: skill.ID, Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt, Files: skill.Files, Version: skill.Version, Enabled: skill.Enabled, CreatedAt: skill.CreatedAt, UpdatedAt: skill.UpdatedAt}
}
func connectorView(connector *domain.MCPConnector) *MCPConnectorView {
	tools := connector.DiscoveredTools
	if tools == nil {
		tools = []domain.MCPTool{}
	}
	return &MCPConnectorView{ID: connector.ID, Name: connector.Name, URL: connector.URL, Enabled: connector.Enabled, Policy: connector.Policy, CredentialMasked: "••••••••", Tools: tools, CreatedAt: connector.CreatedAt, UpdatedAt: connector.UpdatedAt}
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

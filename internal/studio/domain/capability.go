package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Skill struct {
	ID          string
	AccountID   string
	Name        string
	Description string
	Prompt      string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewSkill(id, accountID, name, description, prompt string, now time.Time) (*Skill, error) {
	if anyBlank(id, accountID, name, prompt) {
		return nil, fmt.Errorf("%w: invalid skill", ErrInvalid)
	}
	now = now.UTC()
	return &Skill{
		ID: id, AccountID: accountID, Name: strings.TrimSpace(name),
		Description: strings.TrimSpace(description), Prompt: strings.TrimSpace(prompt),
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

type ConnectorPolicy string

const (
	ConnectorPolicyAuto      ConnectorPolicy = "auto"
	ConnectorPolicyApproval  ConnectorPolicy = "approval"
	ConnectorPolicyForbidden ConnectorPolicy = "forbidden"
)

func (p ConnectorPolicy) Valid() bool {
	return p == ConnectorPolicyAuto || p == ConnectorPolicyApproval || p == ConnectorPolicyForbidden
}

type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

type MCPConnector struct {
	ID               string
	AccountID        string
	Name             string
	URL              string
	CredentialCipher string
	Enabled          bool
	Policy           ConnectorPolicy
	DiscoveredTools  []MCPTool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// AgentWorkflowSetting controls whether a catalog workflow is exposed to one
// Studio account. It never changes the workflow's global enabled state.
type AgentWorkflowSetting struct {
	AccountID    string
	WorkflowID   string
	AgentEnabled bool
	UpdatedAt    time.Time
}

func NewMCPConnector(id, accountID, name, rawURL, credentialCipher string, policy ConnectorPolicy, now time.Time) (*MCPConnector, error) {
	if anyBlank(id, accountID, name, rawURL, credentialCipher) || !policy.Valid() {
		return nil, fmt.Errorf("%w: invalid MCP connector", ErrInvalid)
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("%w: connector URL must be HTTP(S)", ErrInvalid)
	}
	now = now.UTC()
	return &MCPConnector{
		ID: id, AccountID: accountID, Name: strings.TrimSpace(name),
		URL: strings.TrimRight(strings.TrimSpace(rawURL), "/"), CredentialCipher: credentialCipher,
		Enabled: true, Policy: policy, CreatedAt: now, UpdatedAt: now,
	}, nil
}

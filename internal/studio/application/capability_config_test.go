package application_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type connectorProber func(context.Context, string, string) ([]domain.MCPTool, error)

func (f connectorProber) Probe(ctx context.Context, endpoint, credential string) ([]domain.MCPTool, error) {
	return f(ctx, endpoint, credential)
}

func TestCapabilityConfigCreatesEnabledSkill(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.CapabilityConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		IDs:           (&idSequence{}).Next,
	}

	skill, err := service.CreateSkill(context.Background(), studioapp.CreateSkillInput{
		AccountID:   "account-a",
		Name:        "漫画分镜",
		Description: "把故事拆成镜头",
		Prompt:      "先输出镜头表",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if !skill.Enabled || skill.Prompt != "先输出镜头表" {
		t.Fatalf("skill = %#v", skill)
	}

	listed, err := service.ListSkills(context.Background(), "account-a")
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != skill.ID {
		t.Fatalf("listed = %#v", listed)
	}
}

type workflowCatalog struct{ cases []*catalogdomain.Case }

func (c workflowCatalog) List(_ context.Context, _ catalogdomain.ListQuery) ([]*catalogdomain.Case, error) {
	return c.cases, nil
}

func TestCapabilityConfigPersistsAgentWorkflowAvailability(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.CapabilityConfigService{
		Repo: repo,
		WorkflowCatalog: workflowCatalog{cases: []*catalogdomain.Case{{
			Document: catalogdomain.CaseDocument{ID: sharedkernel.CaseID(12), Name: "角色三视图", Description: "生成角色设定图"}, Enabled: true,
		}}},
	}

	workflows, err := service.ListAgentWorkflows(context.Background(), "account-a")
	if err != nil || len(workflows) != 1 || workflows[0].AgentEnabled {
		t.Fatalf("initial workflows = %#v, err=%v", workflows, err)
	}
	updated, err := service.SetAgentWorkflowEnabled(context.Background(), "account-a", "12", true)
	if err != nil || !updated.AgentEnabled {
		t.Fatalf("SetAgentWorkflowEnabled() = %#v, err=%v", updated, err)
	}
	workflows, err = service.ListAgentWorkflows(context.Background(), "account-a")
	if err != nil || !workflows[0].AgentEnabled {
		t.Fatalf("persisted workflows = %#v, err=%v", workflows, err)
	}
}

func TestCapabilityConfigUpdatesSkillEnabledState(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.CapabilityConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		IDs:           (&idSequence{}).Next,
	}
	created, err := service.CreateSkill(context.Background(), studioapp.CreateSkillInput{
		AccountID: "account-a", Name: "角色设定", Prompt: "保持角色一致", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}

	updated, err := service.UpdateSkill(context.Background(), studioapp.UpdateSkillInput{
		AccountID: "account-a", SkillID: created.ID, Name: "角色设定", Prompt: "保持角色一致", Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateSkill() error = %v", err)
	}
	if updated.Enabled {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestCapabilityConfigMasksConnectorCredential(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.CapabilityConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		IDs:           (&idSequence{}).Next,
	}

	connector, err := service.CreateConnector(context.Background(), studioapp.CreateConnectorInput{
		AccountID:  "account-a",
		Name:       "资料库",
		URL:        "https://mcp.example.test/mcp",
		Credential: "connector-secret",
		Policy:     domain.ConnectorPolicyApproval,
	})
	if err != nil {
		t.Fatalf("CreateConnector() error = %v", err)
	}
	if connector.CredentialMasked == "" || strings.Contains(connector.CredentialMasked, "connector-secret") {
		t.Fatalf("connector = %#v", connector)
	}

	stored, err := repo.GetMCPConnector(context.Background(), "account-a", connector.ID)
	if err != nil {
		t.Fatalf("GetMCPConnector() error = %v", err)
	}
	if stored.CredentialCipher == "connector-secret" || stored.CredentialCipher == "" {
		t.Fatalf("stored connector leaked plaintext = %#v", stored)
	}
}

func TestCapabilityConfigUpdatesConnectorWithoutReplacingCredential(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.CapabilityConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		IDs:           (&idSequence{}).Next,
	}
	created, err := service.CreateConnector(context.Background(), studioapp.CreateConnectorInput{
		AccountID: "account-a", Name: "资料库", URL: "https://mcp.example.test/mcp", Credential: "connector-secret", Enabled: true, Policy: domain.ConnectorPolicyApproval,
	})
	if err != nil {
		t.Fatalf("CreateConnector() error = %v", err)
	}
	stored, err := repo.GetMCPConnector(context.Background(), "account-a", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.DiscoveredTools = []domain.MCPTool{{Name: "search_reference", InputSchema: json.RawMessage(`{"type":"object","properties":{}}`)}}
	if err := repo.UpdateMCPConnector(context.Background(), stored); err != nil {
		t.Fatal(err)
	}

	updated, err := service.UpdateConnector(context.Background(), studioapp.UpdateConnectorInput{
		AccountID: "account-a", ConnectorID: created.ID, Name: "资料库", URL: "https://mcp.example.test/updated", Enabled: false, Policy: domain.ConnectorPolicyForbidden,
	})
	if err != nil {
		t.Fatalf("UpdateConnector() error = %v", err)
	}
	if updated.Enabled || updated.Policy != domain.ConnectorPolicyForbidden || updated.CredentialMasked == "" || len(updated.Tools) != 0 {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestCapabilityConfigProbeConnectorPersistsSafeToolMetadata(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.CapabilityConfigService{
		Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: (&idSequence{}).Next,
		MCPProber: connectorProber(func(_ context.Context, endpoint, credential string) ([]domain.MCPTool, error) {
			if endpoint != "https://mcp.example.test/mcp" || credential != "connector-secret" {
				t.Fatalf("probe input endpoint=%q credential=%q", endpoint, credential)
			}
			return []domain.MCPTool{{Name: "search_reference", Description: "Search reference material", InputSchema: json.RawMessage(`{"type":"object","properties":{}}`)}}, nil
		}),
	}
	created, err := service.CreateConnector(context.Background(), studioapp.CreateConnectorInput{
		AccountID: "account-a", Name: "Reference", URL: "https://mcp.example.test/mcp", Credential: "connector-secret", Enabled: true, Policy: domain.ConnectorPolicyApproval,
	})
	if err != nil {
		t.Fatal(err)
	}
	probed, err := service.ProbeConnector(context.Background(), "account-a", created.ID)
	if err != nil || len(probed.Tools) != 1 || probed.Tools[0].Name != "search_reference" || strings.Contains(fmt.Sprintf("%#v", probed), "connector-secret") {
		t.Fatalf("ProbeConnector() = %#v, %v", probed, err)
	}
}

func TestCapabilityConfigResolvesOnlyCallableMCPConnectorsForAgent(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.CapabilityConfigService{
		Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: (&idSequence{}).Next,
	}
	callable, err := service.CreateConnector(context.Background(), studioapp.CreateConnectorInput{
		AccountID: "account-a", Name: "Reference", URL: "https://mcp.example.test/mcp", Credential: "connector-secret", Enabled: true, Policy: domain.ConnectorPolicyApproval,
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetMCPConnector(context.Background(), "account-a", callable.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.DiscoveredTools = []domain.MCPTool{{Name: "search_reference", Description: "Search reference material", InputSchema: json.RawMessage(`{"type":"object","properties":{}}`)}}
	if err := repo.UpdateMCPConnector(context.Background(), stored); err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateConnector(context.Background(), studioapp.CreateConnectorInput{
		AccountID: "account-a", Name: "Disabled", URL: "https://mcp.example.test/disabled", Credential: "disabled-secret", Enabled: false, Policy: domain.ConnectorPolicyAuto,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateConnector(context.Background(), studioapp.CreateConnectorInput{
		AccountID: "account-a", Name: "Forbidden", URL: "https://mcp.example.test/forbidden", Credential: "forbidden-secret", Enabled: true, Policy: domain.ConnectorPolicyForbidden,
	})
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := service.ResolveMCPConnectors(context.Background(), "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 || resolved[0].ID != callable.ID || resolved[0].Credential != "connector-secret" {
		t.Fatalf("ResolveMCPConnectors() = %#v", resolved)
	}
	encoded, err := json.Marshal(resolved[0])
	if err != nil || strings.Contains(string(encoded), "connector-secret") {
		t.Fatalf("resolved connector JSON leaked credential: %s, %v", encoded, err)
	}
}

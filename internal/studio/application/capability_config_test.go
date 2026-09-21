package application_test

import (
	"context"
	"strings"
	"testing"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

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

	updated, err := service.UpdateConnector(context.Background(), studioapp.UpdateConnectorInput{
		AccountID: "account-a", ConnectorID: created.ID, Name: "资料库", URL: "https://mcp.example.test/mcp", Enabled: false, Policy: domain.ConnectorPolicyForbidden,
	})
	if err != nil {
		t.Fatalf("UpdateConnector() error = %v", err)
	}
	if updated.Enabled || updated.Policy != domain.ConnectorPolicyForbidden || updated.CredentialMasked == "" {
		t.Fatalf("updated = %#v", updated)
	}
}

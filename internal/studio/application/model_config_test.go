package application_test

import (
	"context"
	"strings"
	"testing"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestModelConfigEncryptsSecretAndReturnsMaskedView(t *testing.T) {
	repo := openRepository(t)
	key := []byte(strings.Repeat("k", 32))
	ids := &idSequence{}
	service := &studioapp.ModelConfigService{Repo: repo, EncryptionKey: key, IDs: ids.Next}

	created, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID:    "account-a",
		Name:         "Ark DeepSeek",
		Protocol:     domain.ModelProtocolOpenAIChat,
		BaseURL:      "https://ark.example.com/api/coding/v3",
		Model:        "deepseek-v4-flash",
		APIKey:       "temporary-secret",
		Enabled:      true,
		AgentEnabled: true,
		Thinking:     domain.ThinkingConfig{Enabled: true, BudgetTokens: 2048},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created.HasAPIKey || created.APIKeyMasked == "" || strings.Contains(created.APIKeyMasked, "temporary-secret") {
		t.Fatalf("public secret view = %#v", created)
	}
	stored, err := repo.GetModelConfig(context.Background(), "account-a", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.APIKeyCipher == "temporary-secret" || stored.APIKeyCipher == "" {
		t.Fatalf("API key was not encrypted: %q", stored.APIKeyCipher)
	}
	resolved, err := service.Resolve(context.Background(), "account-a", created.ID)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.APIKey != "temporary-secret" {
		t.Fatalf("resolved API key = %q", resolved.APIKey)
	}
}

func TestModelConfigDefaultIsUniquePerAccount(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.ModelConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: (&idSequence{}).Next}
	first, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Model A", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "http://model-a.test/v1", Model: "a", APIKey: "a-key", Enabled: true, AgentEnabled: true, Default: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Model B", Protocol: domain.ModelProtocolAnthropic,
		BaseURL: "http://model-b.test", Model: "b", APIKey: "b-key", Enabled: true, AgentEnabled: true, Default: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := service.List(context.Background(), "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("models = %#v", list)
	}
	defaults := 0
	for _, item := range list {
		if item.Default {
			defaults++
			if item.ID != second.ID {
				t.Fatalf("default model = %q, want %q", item.ID, second.ID)
			}
		}
	}
	if defaults != 1 {
		t.Fatalf("default count = %d", defaults)
	}
	gotFirst, _ := repo.GetModelConfig(context.Background(), "account-a", first.ID)
	if gotFirst.Default {
		t.Fatal("previous default was not cleared")
	}
}

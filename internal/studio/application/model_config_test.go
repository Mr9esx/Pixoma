package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type modelConnectionTester func(context.Context, domain.ResolvedModelConfig) error

func (f modelConnectionTester) Test(ctx context.Context, config domain.ResolvedModelConfig) error {
	return f(ctx, config)
}

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
		Limits:       domain.ModelLimits{ContextWindowTokens: 131072, MaxInputTokens: 120000, MaxOutputTokens: 8192},
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
		Limits: domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Model B", Protocol: domain.ModelProtocolAnthropic,
		BaseURL: "http://model-b.test", Model: "b", APIKey: "b-key", Enabled: true, AgentEnabled: true, Default: true,
		Limits: domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
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

func TestModelConfigConnectionTestUsesStoredSecretWithoutLeakingIt(t *testing.T) {
	repo := openRepository(t)
	key := []byte(strings.Repeat("k", 32))
	service := &studioapp.ModelConfigService{
		Repo:          repo,
		EncryptionKey: key,
		IDs:           (&idSequence{}).Next,
		Tester: modelConnectionTester(func(_ context.Context, config domain.ResolvedModelConfig) error {
			if config.APIKey != "test-secret" || config.Model != "model-test" {
				t.Fatalf("connection config = %#v", config)
			}
			return nil
		}),
	}
	created, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Test", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "http://model.test/v1", Model: "model-test", APIKey: "test-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.TestConnection(context.Background(), "account-a", created.ID)
	if err != nil || !result.Success || result.LatencyMS < 0 {
		t.Fatalf("TestConnection() = %#v, %v", result, err)
	}

	service.Tester = modelConnectionTester(func(_ context.Context, _ domain.ResolvedModelConfig) error {
		return fmt.Errorf("provider rejected test-secret")
	})
	_, err = service.TestConnection(context.Background(), "account-a", created.ID)
	if err == nil || strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("connection failure leaked secret: %v", err)
	}
}

func TestModelConfigTransientConnectionTestDoesNotPersist(t *testing.T) {
	repo := openRepository(t)
	keySeen := ""
	service := &studioapp.ModelConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		Tester: modelConnectionTester(func(_ context.Context, config domain.ResolvedModelConfig) error {
			keySeen = config.APIKey
			if config.Model != "model-test" || config.BaseURL != "https://model.test/v1" {
				t.Fatalf("transient config = %#v", config)
			}
			return nil
		}),
	}
	result, err := service.TestConnectionConfig(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Test", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://model.test/v1", Model: "model-test", APIKey: "temporary-secret",
	})
	if err != nil || !result.Success || result.LatencyMS < 0 {
		t.Fatalf("TestConnectionConfig() = %#v, %v", result, err)
	}
	if keySeen != "temporary-secret" {
		t.Fatalf("tester key = %q", keySeen)
	}
	models, err := repo.ListModelConfigs(context.Background(), "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 0 {
		t.Fatalf("transient test persisted %d model(s)", len(models))
	}
}

func TestModelConfigTransientConnectionTestCanReuseStoredKeyWithOverrides(t *testing.T) {
	repo := openRepository(t)
	keySeen := ""
	service := &studioapp.ModelConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		Tester: modelConnectionTester(func(_ context.Context, config domain.ResolvedModelConfig) error {
			keySeen = config.APIKey
			if config.Model != "overridden-model" {
				t.Fatalf("override config = %#v", config)
			}
			return nil
		}),
	}
	created, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Stored", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://stored.example/v1", Model: "stored-model", APIKey: "stored-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.TestConnectionConfig(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", ExistingModelID: created.ID, Name: "Edited", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://edited.example/v1", Model: "overridden-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if keySeen != "stored-secret" {
		t.Fatalf("stored key = %q", keySeen)
	}
}

func TestModelConfigUpdateKeepsExistingKeyWhenInputIsBlank(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.ModelConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: (&idSequence{}).Next}
	created, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Before", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://before.example/v1", Model: "before", APIKey: "keep-secret",
		Limits: domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(context.Background(), "account-a", created.ID, studioapp.UpdateModelConfigInput{
		Name: "After", Protocol: domain.ModelProtocolOpenAIResponses,
		BaseURL: "https://after.example/v1", Model: "after", APIKey: "",
		Enabled: true, AgentEnabled: true, Default: true,
		Limits: domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "After" || updated.Protocol != domain.ModelProtocolOpenAIResponses || !updated.Default {
		t.Fatalf("updated model = %#v", updated)
	}
	resolved, err := service.Resolve(context.Background(), "account-a", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.APIKey != "keep-secret" {
		t.Fatalf("existing key changed to %q", resolved.APIKey)
	}
}

func TestModelConnectionErrorsAreSanitizedAndClassified(t *testing.T) {
	service := &studioapp.ModelConfigService{
		Tester: modelConnectionTester(func(_ context.Context, _ domain.ResolvedModelConfig) error {
			return errors.New("provider rejected keep-secret")
		}),
	}
	_, err := service.TestConnectionConfig(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Test", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://model.example/v1", Model: "model", APIKey: "keep-secret",
	})
	if err == nil || !errors.Is(err, studioapp.ErrModelConnectionTest) || strings.Contains(err.Error(), "keep-secret") {
		t.Fatalf("connection error = %v", err)
	}
}

func TestStoredModelConnectionDecryptErrorsAreClassified(t *testing.T) {
	repo := openRepository(t)
	createdBy := &studioapp.ModelConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: (&idSequence{}).Next}
	created, err := createdBy.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Stored", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://model.example/v1", Model: "model", APIKey: "stored-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	service := &studioapp.ModelConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("z", 32)),
		Tester:        modelConnectionTester(func(context.Context, domain.ResolvedModelConfig) error { return nil }),
	}
	_, err = service.TestConnection(context.Background(), "account-a", created.ID)
	if err == nil || !errors.Is(err, studioapp.ErrModelConnectionTest) || strings.Contains(err.Error(), "stored-secret") {
		t.Fatalf("decrypt error = %v", err)
	}
}

func TestModelLimitsRejectMissingOrOversizedValues(t *testing.T) {
	tests := []struct {
		name   string
		limits domain.ModelLimits
	}{
		{name: "missing context window", limits: domain.ModelLimits{MaxInputTokens: 1000, MaxOutputTokens: 1000}},
		{name: "missing input limit", limits: domain.ModelLimits{ContextWindowTokens: 4096, MaxOutputTokens: 1000}},
		{name: "missing output limit", limits: domain.ModelLimits{ContextWindowTokens: 4096, MaxInputTokens: 3000}},
		{name: "input exceeds window", limits: domain.ModelLimits{ContextWindowTokens: 4096, MaxInputTokens: 8192, MaxOutputTokens: 1000}},
		{name: "output exceeds window", limits: domain.ModelLimits{ContextWindowTokens: 4096, MaxInputTokens: 3000, MaxOutputTokens: 8192}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.limits.Validate(); err == nil {
				t.Fatalf("Validate() error = nil for %#v", tt.limits)
			}
		})
	}
}

func TestAgentModelCreationRejectsMissingLimits(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.ModelConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		IDs:           (&idSequence{}).Next,
	}
	_, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Unbounded", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://model.example/v1", Model: "model", APIKey: "secret",
		Enabled: true, AgentEnabled: false,
	})
	if err != nil {
		t.Fatal("Create() should allow storing an incomplete non-Agent model only")
	}

	_, err = service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Unbounded Agent", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://model.example/v1", Model: "model", APIKey: "secret",
		Enabled: true, AgentEnabled: true,
	})
	if err == nil {
		t.Fatal("Create() error = nil for Agent-enabled model without limits")
	}
}

func TestModelLimitsRoundTripThroughModelConfig(t *testing.T) {
	repo := openRepository(t)
	service := &studioapp.ModelConfigService{
		Repo:          repo,
		EncryptionKey: []byte(strings.Repeat("k", 32)),
		IDs:           (&idSequence{}).Next,
	}
	want := domain.ModelLimits{ContextWindowTokens: 131072, MaxInputTokens: 120000, MaxOutputTokens: 8192}
	created, err := service.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "Bounded", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: "https://model.example/v1", Model: "model", APIKey: "secret",
		Enabled: true, AgentEnabled: true, Limits: want,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Limits != want {
		t.Fatalf("created limits = %#v, want %#v", created.Limits, want)
	}
	resolved, err := service.Resolve(context.Background(), "account-a", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Limits != want {
		t.Fatalf("resolved limits = %#v, want %#v", resolved.Limits, want)
	}
}

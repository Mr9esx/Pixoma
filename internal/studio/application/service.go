package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type TitleGenerator interface {
	GenerateTitle(ctx context.Context, firstMessage string) (string, error)
}

type RunRef struct {
	AccountID string
	RunID     string
}

type RunQueue interface {
	Enqueue(ref RunRef) error
}

type Service struct {
	Repo   domain.Repository
	IDs    func() string
	Now    func() time.Time
	Titles TitleGenerator
	Queue  RunQueue
}

type SendMessageInput struct {
	AccountID        string
	SessionID        string
	RequestID        string
	Text             string
	Parts            []MessagePart
	ModelConfigID    string
	PermissionMode   domain.PermissionMode
	SkillIDs         []string
	SelectedAssetIDs []string
	SelectedAssets   []domain.AssetReference
}

type SendMessageResult struct {
	Session *domain.Session
	Message *domain.Message
	Run     *domain.Run
}

func (s *Service) CreateSession(ctx context.Context, accountID string) (*domain.Session, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: repository is required")
	}
	session, err := domain.NewSession(s.nextID(), strings.TrimSpace(accountID), s.now())
	if err != nil {
		return nil, err
	}
	if err := s.Repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

type MessagePart struct {
	Type           string `json:"type"`
	Text           string `json:"text,omitempty"`
	SkillID        string `json:"skill_id,omitempty"`
	AssetID        string `json:"asset_id,omitempty"`
	AssetVersionID string `json:"asset_version_id,omitempty"`
	Name           string `json:"name,omitempty"`
}

func normalizeMessageInput(input SendMessageInput) (SendMessageInput, error) {
	if len(input.Parts) == 0 {
		return input, nil
	}
	text, err := messagePartsText(input.Parts)
	if err != nil {
		return input, err
	}
	if strings.TrimSpace(input.Text) != strings.TrimSpace(text) {
		return input, fmt.Errorf("%w: message text and references differ", domain.ErrInvalid)
	}
	input.Text = strings.TrimSpace(text)
	input.SkillIDs = nil
	input.SelectedAssetIDs = nil
	input.SelectedAssets = nil
	seenSkills := make(map[string]bool)
	seenAssets := make(map[string]string)
	for _, part := range input.Parts {
		switch part.Type {
		case "skill_ref":
			if !seenSkills[part.SkillID] {
				input.SkillIDs = append(input.SkillIDs, part.SkillID)
				seenSkills[part.SkillID] = true
			}
		case "asset_ref":
			if version, exists := seenAssets[part.AssetID]; exists {
				if version != part.AssetVersionID {
					return input, fmt.Errorf("%w: one asset has multiple selected versions", domain.ErrInvalid)
				}
				continue
			}
			seenAssets[part.AssetID] = part.AssetVersionID
			input.SelectedAssets = append(input.SelectedAssets, domain.AssetReference{AssetID: part.AssetID, AssetVersionID: part.AssetVersionID})
		}
	}
	return input, nil
}

func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (*SendMessageResult, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: repository is required")
	}
	input.AccountID = strings.TrimSpace(input.AccountID)
	input.Text = strings.TrimSpace(input.Text)
	input.RequestID = strings.TrimSpace(input.RequestID)
	var err error
	input, err = normalizeMessageInput(input)
	if err != nil {
		return nil, err
	}
	if input.AccountID == "" || input.Text == "" {
		return nil, fmt.Errorf("%w: account and message text are required", domain.ErrInvalid)
	}
	if input.RequestID == "" {
		input.RequestID = uuid.NewString()
	}
	if input.SessionID != "" {
		previous, err := s.Repo.GetRunByRequestID(ctx, input.AccountID, input.SessionID, input.RequestID)
		if err == nil {
			return s.existingTurn(ctx, previous, input.Text)
		}
		if err != domain.ErrNotFound {
			return nil, err
		}
	}
	now := s.now()

	session, created, err := s.resolveSession(ctx, input, now)
	if err != nil {
		return nil, err
	}
	parts := input.Parts
	if len(parts) == 0 {
		parts = []MessagePart{{Type: "text", Text: input.Text}}
	}
	content, err := json.Marshal(parts)
	if err != nil {
		return nil, err
	}
	message := &domain.Message{
		ID: s.nextID(), SessionID: session.ID, AccountID: input.AccountID,
		Role: domain.MessageRoleUser, ContentJSON: content, CreatedAt: now,
	}
	run, err := domain.NewRun(s.nextID(), session.ID, input.AccountID, message.ID, now)
	if err != nil {
		return nil, err
	}
	run.ModelConfigID = session.ModelConfigID
	run.RequestID = input.RequestID
	skillSnapshot, err := s.snapshotSkills(ctx, input.AccountID)
	if err != nil {
		return nil, err
	}
	skillIDs, err := resolveSkillIDs(input.SkillIDs, skillSnapshot)
	if err != nil {
		return nil, err
	}
	run.SkillIDs = skillIDs
	run.SkillSnapshot = skillSnapshot
	assetReferences, err := s.resolveAssetReferences(ctx, input.AccountID, session.ID, input.SelectedAssets, input.SelectedAssetIDs)
	if err != nil {
		return nil, err
	}
	run.AssetReferences = assetReferences
	run.AssetIDs = assetIDsFromReferences(assetReferences)
	message.RunID = run.ID

	stored, createdTurn, err := s.Repo.CreateRunTurn(ctx, message, run)
	if err != nil {
		return nil, err
	}
	if !createdTurn {
		return s.existingTurn(ctx, stored, input.Text)
	}
	if s.Queue == nil {
		return nil, fmt.Errorf("studio: run queue is required")
	}
	if err := s.Queue.Enqueue(RunRef{AccountID: input.AccountID, RunID: run.ID}); err != nil {
		_ = run.Fail("enqueue_failed", err.Error(), s.now())
		_ = s.Repo.UpdateRun(context.WithoutCancel(ctx), run)
		return nil, err
	}
	_ = created // retained to make the create-vs-existing lifecycle explicit.
	return &SendMessageResult{Session: session, Message: message, Run: run}, nil
}

func (s *Service) existingTurn(ctx context.Context, run *domain.Run, requestedText string) (*SendMessageResult, error) {
	session, err := s.Repo.GetSession(ctx, run.AccountID, run.SessionID)
	if err != nil {
		return nil, err
	}
	message, err := s.Repo.GetMessage(ctx, run.AccountID, run.TriggerMessageID)
	if err != nil {
		return nil, err
	}
	storedText, err := messageText(message.ContentJSON)
	if err != nil {
		return nil, err
	}
	if storedText != requestedText {
		return nil, fmt.Errorf("%w: request id belongs to a different message", domain.ErrInvalid)
	}
	return &SendMessageResult{Session: session, Message: message, Run: run}, nil
}

func (s *Service) resolveAssetReferences(ctx context.Context, accountID, sessionID string, references []domain.AssetReference, legacyIDs []string) ([]domain.AssetReference, error) {
	if len(references) == 0 {
		references = make([]domain.AssetReference, 0, len(legacyIDs))
		for _, assetID := range legacyIDs {
			references = append(references, domain.AssetReference{AssetID: assetID})
		}
	}
	seen := make(map[string]struct{}, len(references))
	selected := make([]domain.AssetReference, 0, len(references))
	for _, reference := range references {
		reference.AssetID = strings.TrimSpace(reference.AssetID)
		reference.AssetVersionID = strings.TrimSpace(reference.AssetVersionID)
		if reference.AssetID == "" {
			continue
		}
		if _, ok := seen[reference.AssetID]; ok {
			continue
		}
		asset, err := s.Repo.GetAsset(ctx, accountID, reference.AssetID)
		if err != nil {
			return nil, err
		}
		if asset.SessionID != sessionID && asset.LibrarySavedAt.IsZero() {
			return nil, fmt.Errorf("%w: asset is not available in session", domain.ErrInvalid)
		}
		if len(asset.Versions) == 0 {
			return nil, fmt.Errorf("%w: asset has no versions", domain.ErrInvalid)
		}
		if reference.AssetVersionID == "" {
			reference.AssetVersionID = asset.Versions[len(asset.Versions)-1].ID
		}
		if !assetHasVersion(asset, reference.AssetVersionID) {
			return nil, fmt.Errorf("%w: asset version is not available", domain.ErrInvalid)
		}
		seen[reference.AssetID] = struct{}{}
		selected = append(selected, reference)
	}
	return selected, nil
}

func assetHasVersion(asset *domain.Asset, versionID string) bool {
	for _, version := range asset.Versions {
		if version.ID == versionID {
			return true
		}
	}
	return false
}

func assetIDsFromReferences(references []domain.AssetReference) []string {
	assetIDs := make([]string, 0, len(references))
	for _, reference := range references {
		assetIDs = append(assetIDs, reference.AssetID)
	}
	return assetIDs
}

func (s *Service) RetryRun(ctx context.Context, accountID, runID string) (*domain.Run, error) {
	if s == nil || s.Repo == nil || s.Queue == nil {
		return nil, fmt.Errorf("studio: service is not configured")
	}
	previous, err := s.Repo.GetRun(ctx, accountID, runID)
	if err != nil {
		return nil, err
	}
	if !previous.Status.Terminal() || previous.Status == domain.RunSucceeded {
		return nil, domain.ErrInvalidTransition
	}
	retried, err := domain.NewRun(s.nextID(), previous.SessionID, previous.AccountID, previous.TriggerMessageID, s.now())
	if err != nil {
		return nil, err
	}
	retried.ModelConfigID = previous.ModelConfigID
	retried.SkillIDs = append([]string(nil), previous.SkillIDs...)
	if previous.SkillSnapshot != nil {
		retried.SkillSnapshot = append([]domain.RunSkill{}, previous.SkillSnapshot...)
	}
	retried.AssetIDs = append([]string(nil), previous.AssetIDs...)
	retried.AssetReferences = append([]domain.AssetReference(nil), previous.AssetReferences...)
	if err := s.Repo.CreateRun(ctx, retried); err != nil {
		return nil, err
	}
	if err := s.Queue.Enqueue(RunRef{AccountID: accountID, RunID: retried.ID}); err != nil {
		_ = retried.Fail("enqueue_failed", err.Error(), s.now())
		_ = s.Repo.UpdateRun(context.WithoutCancel(ctx), retried)
		return nil, err
	}
	return retried, nil
}

func (s *Service) snapshotSkills(ctx context.Context, accountID string) ([]domain.RunSkill, error) {
	skills, err := s.Repo.ListSkills(ctx, accountID)
	if err != nil {
		return nil, err
	}
	snapshot := make([]domain.RunSkill, 0, len(skills))
	for _, skill := range skills {
		if skill.Enabled {
			snapshot = append(snapshot, domain.RunSkill{
				ID: skill.ID, Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt,
			})
		}
	}
	return snapshot, nil
}

func resolveSkillIDs(ids []string, snapshot []domain.RunSkill) ([]string, error) {
	available := make(map[string]bool, len(snapshot))
	for _, skill := range snapshot {
		available[skill.ID] = true
	}
	seen := make(map[string]struct{}, len(ids))
	selected := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if !available[id] {
			return nil, fmt.Errorf("%w: selected Skill is unavailable", domain.ErrInvalid)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		selected = append(selected, id)
	}
	return selected, nil
}

func (s *Service) resolveSession(ctx context.Context, input SendMessageInput, now time.Time) (*domain.Session, bool, error) {
	if input.SessionID != "" {
		session, err := s.Repo.GetSession(ctx, input.AccountID, input.SessionID)
		if err != nil {
			return nil, false, err
		}
		needsUpdate := false
		if session.Title == domain.DefaultSessionTitle {
			if err := session.Rename(s.generateTitle(ctx, input.Text), now); err != nil {
				return nil, false, err
			}
			needsUpdate = true
		}
		if input.PermissionMode.Valid() || input.ModelConfigID != "" {
			mode := session.PermissionMode
			if input.PermissionMode.Valid() {
				mode = input.PermissionMode
			}
			modelID := session.ModelConfigID
			if input.ModelConfigID != "" {
				modelID = input.ModelConfigID
			}
			if err := session.Configure(modelID, mode, now); err != nil {
				return nil, false, err
			}
			needsUpdate = true
		}
		if needsUpdate {
			if err := s.Repo.UpdateSession(ctx, session); err != nil {
				return nil, false, err
			}
		}
		return session, false, nil
	}

	session, err := domain.NewSession(s.nextID(), input.AccountID, now)
	if err != nil {
		return nil, false, err
	}
	mode := session.PermissionMode
	if input.PermissionMode.Valid() {
		mode = input.PermissionMode
	}
	if err := session.Configure(input.ModelConfigID, mode, now); err != nil {
		return nil, false, err
	}
	title := s.generateTitle(ctx, input.Text)
	if err := session.Rename(title, now); err != nil {
		return nil, false, err
	}
	if err := s.Repo.CreateSession(ctx, session); err != nil {
		return nil, false, err
	}
	return session, true, nil
}

func (s *Service) generateTitle(ctx context.Context, text string) string {
	if s.Titles != nil {
		if title, err := s.Titles.GenerateTitle(ctx, text); err == nil {
			if normalized := normalizeTitle(title); normalized != "" {
				return normalized
			}
		}
	}
	return normalizeTitle(text)
}

func normalizeTitle(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return domain.DefaultSessionTitle
	}
	if utf8.RuneCountInString(value) <= 20 {
		return value
	}
	return string([]rune(value)[:20])
}

func (s *Service) nextID() string {
	if s.IDs != nil {
		return s.IDs()
	}
	return uuid.NewString()
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

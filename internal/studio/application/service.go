package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type TitleGenerator interface {
	GenerateTitle(ctx context.Context, accountID, modelConfigID, firstMessage string) (string, error)
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
	ProjectID        string
	RequestID        string
	Text             string
	Locale           string
	Parts            []MessagePart
	ModelConfigID    string
	PermissionMode   domain.PermissionMode
	SkillIDs         []string
	SelectedAssetIDs []string
	SelectedAssets   []domain.AssetReference
	createOnSend     bool
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

// CreateSessionWithRequestID makes a retried create request resolve to the
// original session even when its successful HTTP response was lost. The
// account is included in the deterministic ID so keys cannot collide across
// accounts; callers without a key keep the legacy random-ID behavior.
func (s *Service) CreateSessionWithRequestID(ctx context.Context, accountID, requestID string) (*domain.Session, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return s.CreateSession(ctx, accountID)
	}
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: repository is required")
	}
	parsed, err := uuid.Parse(requestID)
	if err != nil {
		return nil, fmt.Errorf("%w: request_id must be a UUID", domain.ErrInvalid)
	}
	accountID = strings.TrimSpace(accountID)
	sessionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(accountID+"\x00"+parsed.String())).String()
	previous, err := s.Repo.GetSession(ctx, accountID, sessionID)
	if err == nil {
		return previous, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	session, err := domain.NewSession(sessionID, accountID, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.Repo.CreateSession(ctx, session); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			if existing, getErr := s.Repo.GetSession(ctx, accountID, sessionID); getErr == nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return session, nil
}

type messagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type MessagePart struct {
	Type           string `json:"type"`
	Text           string `json:"text,omitempty"`
	SkillID        string `json:"skill_id,omitempty"`
	AssetID        string `json:"asset_id,omitempty"`
	AssetVersionID string `json:"asset_version_id,omitempty"`
	WorkflowID     string `json:"workflow_id,omitempty"`
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
	input.Locale = strings.TrimSpace(input.Locale)
	if input.Locale == "" {
		input.Locale = "zh"
	}
	if input.Locale != "zh" && input.Locale != "en" {
		return nil, fmt.Errorf("%w: unsupported studio locale", domain.ErrInvalid)
	}
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
	if input.SessionID == "" {
		input.createOnSend = true
		input.SessionID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(input.AccountID+"\x00message\x00"+input.RequestID)).String()
		if _, err := s.Repo.GetSession(ctx, input.AccountID, input.SessionID); err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		} else if errors.Is(err, domain.ErrNotFound) && input.ProjectID != "" {
			if _, err := s.Repo.GetProject(ctx, input.AccountID, input.ProjectID); err != nil {
				return nil, err
			}
		}
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
	run.Locale = input.Locale
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
	for _, reference := range assetReferences {
		asset, err := s.Repo.GetAsset(ctx, input.AccountID, reference.AssetID)
		if err != nil {
			return nil, err
		}
		placement := &domain.ProjectAsset{
			ID: s.nextID(), AccountID: input.AccountID, ProjectID: session.ProjectID,
			AssetID: asset.ID, AssetVersionID: reference.AssetVersionID, DisplayName: asset.Name,
			AddedAt: now, UpdatedAt: now,
		}
		usage := &domain.SessionAssetUsage{
			ID: s.nextID(), AccountID: input.AccountID, SessionID: session.ID,
			AssetID: asset.ID, AssetVersionID: reference.AssetVersionID,
			UsageKind: "referenced", OperationKey: run.ID, RunID: run.ID,
			MessageID: message.ID, CreatedAt: now,
		}
		if err := s.Repo.ReferenceAssetInSession(ctx, placement, usage); err != nil {
			_ = run.Fail("asset_reference_failed", err.Error(), s.now())
			_ = s.Repo.UpdateRun(context.WithoutCancel(ctx), run)
			return nil, err
		}
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
	retried.Locale = previous.Locale
	if retried.Locale == "" {
		retried.Locale = "zh"
	}
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
			prompt := skill.Prompt
			if len(skill.Files) > 0 {
				prompt = ""
			}
			snapshot = append(snapshot, domain.RunSkill{
				ID: skill.ID, Name: skill.Name, Description: skill.Description, Prompt: prompt,
				Files: append([]domain.SkillFile(nil), skill.Files...),
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
		created := false
		if errors.Is(err, domain.ErrNotFound) && input.createOnSend {
			session, err = domain.NewSession(input.SessionID, input.AccountID, now)
			if err != nil {
				return nil, false, err
			}
			session.ProjectID = input.ProjectID
			if err := s.Repo.CreateSession(ctx, session); err != nil {
				if !errors.Is(err, domain.ErrAlreadyExists) {
					return nil, false, err
				}
				session, err = s.Repo.GetSession(ctx, input.AccountID, input.SessionID)
				if err != nil {
					return nil, false, err
				}
			} else {
				created = true
			}
		} else if err != nil {
			return nil, false, err
		}
		needsUpdate := false
		if session.Title == domain.DefaultSessionTitle {
			modelConfigID := input.ModelConfigID
			if modelConfigID == "" {
				modelConfigID = session.ModelConfigID
			}
			if err := session.Rename(s.generateTitle(ctx, input.AccountID, modelConfigID, input.Text), now); err != nil {
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
		return session, created, nil
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
	title := s.generateTitle(ctx, input.AccountID, input.ModelConfigID, input.Text)
	if err := session.Rename(title, now); err != nil {
		return nil, false, err
	}
	if err := s.Repo.CreateSession(ctx, session); err != nil {
		return nil, false, err
	}
	return session, true, nil
}

func (s *Service) generateTitle(ctx context.Context, accountID, modelConfigID, text string) string {
	if s.Titles != nil {
		titleCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		title, err := s.Titles.GenerateTitle(titleCtx, accountID, modelConfigID, text)
		if err == nil && strings.TrimSpace(title) != "" {
			return normalizeTitle(title)
		}
		if err != nil {
			slog.Warn("studio title generation failed", "error", err)
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

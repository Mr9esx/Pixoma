package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/application"
	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

type memStore struct {
	mu   sync.Mutex
	docs map[string]domain.MenuDocument
}

func newMemStore(t *testing.T) *memStore {
	t.Helper()
	s := &memStore{docs: map[string]domain.MenuDocument{}}
	seed := domain.DefaultSeed()
	seed.UpdatedAt = time.Now().UTC()
	s.docs[domain.DocumentIDDefault] = seed
	return s
}

func (s *memStore) Get(_ context.Context, id string) (domain.MenuDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[id]
	if !ok {
		return domain.MenuDocument{}, domain.ErrNotFound
	}
	return doc, nil
}

func (s *memStore) Replace(_ context.Context, doc domain.MenuDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc.UpdatedAt = time.Now().UTC()
	s.docs[doc.ID] = doc
	return nil
}

func (s *memStore) EnsureDefault(ctx context.Context) (domain.MenuDocument, error) {
	doc, err := s.Get(ctx, domain.DocumentIDDefault)
	if err == nil {
		return doc, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.MenuDocument{}, err
	}
	seed := domain.DefaultSeed()
	if err := s.Replace(ctx, seed); err != nil {
		return domain.MenuDocument{}, err
	}
	return s.Get(ctx, domain.DocumentIDDefault)
}

type alwaysMissing struct{}

func (alwaysMissing) CaseExists(context.Context, string) (bool, error) { return false, nil }

type alwaysExists struct{}

func (alwaysExists) CaseExists(context.Context, string) (bool, error) { return true, nil }

func TestService_ReplaceRejectsInvalidWithoutWriting(t *testing.T) {
	store := newMemStore(t)
	svc := &application.Service{Store: store, Cases: alwaysMissing{}}
	_, err := svc.Replace(context.Background(), []domain.MenuItem{{
		ID: "x", Label: "X", Action: domain.ActionOpenCase, CaseID: "nope", Enabled: true,
	}})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err=%v", err)
	}
	doc, _ := store.Get(context.Background(), domain.DocumentIDDefault)
	if len(doc.Items) != 6 {
		t.Fatalf("store mutated: %d", len(doc.Items))
	}
}

func TestService_GetAndReplaceHappyPath(t *testing.T) {
	store := newMemStore(t)
	svc := &application.Service{Store: store, Cases: alwaysExists{}}
	got, err := svc.Get(context.Background())
	if err != nil || len(got.Items) != 6 {
		t.Fatalf("get: %v %+v", err, got)
	}
	items := domain.DefaultSeed().Items
	items[0].Label = "🖼 图生图"
	out, err := svc.Replace(context.Background(), items)
	if err != nil {
		t.Fatal(err)
	}
	if out.Items[0].Label != "🖼 图生图" {
		t.Fatalf("replace: %+v", out.Items[0])
	}
}

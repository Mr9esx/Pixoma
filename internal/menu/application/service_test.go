package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/application"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

type memStore struct {
	mu    sync.Mutex
	trees map[string]domain.MenuTree
}

func newMemStore(t *testing.T) *memStore {
	t.Helper()
	s := &memStore{trees: map[string]domain.MenuTree{}}
	seed := domain.DefaultSeedTree("tg-default")
	seed.UpdatedAt = time.Now().UTC()
	s.trees["tg-default"] = seed
	return s
}

func (s *memStore) GetTree(_ context.Context, id string) (domain.MenuTree, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tree, ok := s.trees[id]
	if !ok {
		return domain.MenuTree{}, domain.ErrNotFound
	}
	return tree, nil
}

func (s *memStore) ReplaceTree(_ context.Context, tree domain.MenuTree) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tree.UpdatedAt = time.Now().UTC()
	s.trees[tree.ChannelID] = tree
	return nil
}

func (s *memStore) ListPlacementsByCase(_ context.Context, caseID string) ([]domain.MenuPlacement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.MenuPlacement
	for _, tree := range s.trees {
		flat := domain.Flatten(tree.Items)
		for _, it := range flat {
			for _, cid := range domain.CaseIDsOf(domain.MenuNode{CapabilityID: it.CapabilityID, Params: it.Params}) {
				if cid == caseID {
					out = append(out, domain.MenuPlacement{
						ChannelID: tree.ChannelID,
						ItemID:    it.ID,
						Path:      []domain.PlacementStep{{ID: it.ID, Label: it.Label}},
					})
				}
			}
		}
	}
	return out, nil
}

func (s *memStore) ListExtras(context.Context, string) (map[string][]domain.Extra, error) {
	return map[string][]domain.Extra{}, nil
}

func (s *memStore) SaveExtras(context.Context, string, map[string][]domain.Extra) error {
	return nil
}

func (s *memStore) EnsureDefault(
	ctx context.Context,
	channelID string,
	listImageCaseIDs func(context.Context) ([]string, error),
) (domain.MenuTree, error) {
	tree, err := s.GetTree(ctx, channelID)
	if err == nil {
		return tree, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.MenuTree{}, err
	}
	seed := domain.DefaultSeedTree(channelID)
	if listImageCaseIDs != nil {
		ids, err := listImageCaseIDs(ctx)
		if err != nil {
			return domain.MenuTree{}, err
		}
		for i := range seed.Items {
			if seed.Items[i].ID == "btn-image" {
				params := map[string]any{}
				if seed.Items[i].Params != nil {
					for k, v := range seed.Items[i].Params {
						params[k] = v
					}
				}
				anyIDs := make([]any, 0, len(ids))
				for _, id := range ids {
					anyIDs = append(anyIDs, id)
				}
				params["case_ids"] = anyIDs
				seed.Items[i].Params = params
				break
			}
		}
	}
	if err := s.ReplaceTree(ctx, seed); err != nil {
		return domain.MenuTree{}, err
	}
	return s.GetTree(ctx, channelID)
}

type alwaysMissing struct{}

func (alwaysMissing) CaseExists(context.Context, string) (bool, error) { return false, nil }

type alwaysExists struct{}

func (alwaysExists) CaseExists(context.Context, string) (bool, error) { return true, nil }

type allCapabilities struct{}

func (allCapabilities) CapabilityExists(context.Context, string) (bool, error) { return true, nil }
func (allCapabilities) ValidateParams(context.Context, string, map[string]any) error { return nil }

func TestService_ReplaceRejectsInvalidFolderCaseWithoutWriting(t *testing.T) {
	store := newMemStore(t)
	svc := &application.Service{Store: store, Cases: alwaysMissing{}, Capabilities: allCapabilities{}}

	invalid := domain.DefaultSeedTree("tg-default")
	invalid.Items[0].Params = map[string]any{"case_ids": []any{"nope"}}

	_, err := svc.Replace(context.Background(), "tg-default", invalid)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err=%v", err)
	}

	got, _ := store.GetTree(context.Background(), "tg-default")
	if len(got.Items) != 6 {
		t.Fatalf("store mutated: %d items", len(got.Items))
	}
}

func TestService_GetAndReplaceHappyPath(t *testing.T) {
	store := newMemStore(t)
	svc := &application.Service{Store: store, Cases: alwaysExists{}, Capabilities: allCapabilities{}}

	got, err := svc.Get(context.Background(), "tg-default")
	if err != nil || len(got.Items) != 6 {
		t.Fatalf("get: %v %+v", err, got)
	}

	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[0].Label = "🖼 图生图"
	tree.Items[0].Params = map[string]any{"case_ids": []any{"case-a"}}

	out, err := svc.Replace(context.Background(), "tg-default", tree)
	if err != nil {
		t.Fatal(err)
	}
	if out.Items[0].Label != "🖼 图生图" {
		t.Fatalf("replace: %+v", out.Items[0])
	}

	again, err := svc.Get(context.Background(), "tg-default")
	if err != nil {
		t.Fatal(err)
	}
	if again.Items[0].Label != "🖼 图生图" {
		t.Fatalf("get after replace: %+v", again.Items[0])
	}
}

func TestService_ReplaceForcesChannelID(t *testing.T) {
	store := newMemStore(t)
	svc := &application.Service{Store: store, Cases: alwaysExists{}, Capabilities: allCapabilities{}}

	tree := domain.DefaultSeedTree("tg-default")
	tree.ChannelID = "other"
	tree.Items[0].Params = map[string]any{"case_ids": []any{"case-a"}}

	out, err := svc.Replace(context.Background(), "tg-default", tree)
	if err != nil {
		t.Fatal(err)
	}
	if out.ChannelID != "tg-default" {
		t.Fatalf("channel id not forced: %q", out.ChannelID)
	}
}

func TestService_ListPlacementsByCase(t *testing.T) {
	store := newMemStore(t)
	svc := &application.Service{Store: store, Cases: alwaysExists{}, Capabilities: allCapabilities{}}

	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[0].Params = map[string]any{"case_ids": []any{"img-1", "img-2"}}
	if _, err := svc.Replace(context.Background(), "tg-default", tree); err != nil {
		t.Fatal(err)
	}

	placements, err := svc.ListPlacementsByCase(context.Background(), "img-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(placements) != 1 || placements[0].ItemID != "btn-image" {
		t.Fatalf("placements=%+v", placements)
	}
}

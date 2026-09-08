package application

import (
	"context"

	"github.com/Mr9esx/Pixoma/internal/topics/domain"
	"testing"
	"time"
)

type fakeRepo struct {
	topics map[string]domain.Topic
}

func (f *fakeRepo) List(_ context.Context, _ *bool) ([]domain.Topic, error) {
	out := make([]domain.Topic, 0, len(f.topics))
	for _, t := range f.topics {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, key string) (*domain.Topic, error) {
	t, ok := f.topics[key]
	if !ok {
		return nil, domain.ErrTopicNotFound
	}
	return &t, nil
}

func (f *fakeRepo) Create(_ context.Context, t domain.Topic) error {
	if _, ok := f.topics[t.Key]; ok {
		return domain.ErrTopicConflict
	}
	f.topics[t.Key] = t
	return nil
}

func (f *fakeRepo) Update(_ context.Context, t domain.Topic) error {
	f.topics[t.Key] = t
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, key string) error {
	delete(f.topics, key)
	return nil
}

func TestEnsureDefaultTopic_CreatesOnce(t *testing.T) {
	repo := &fakeRepo{topics: map[string]domain.Topic{}}
	ctx := context.Background()

	if err := EnsureDefaultTopic(ctx, repo); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	got, err := repo.Get(ctx, domain.DefaultKey)
	if err != nil {
		t.Fatalf("default topic missing: %v", err)
	}
	if !got.Enabled || got.Name == "" {
		t.Fatalf("default topic invalid: %+v", got)
	}
	if err := EnsureDefaultTopic(ctx, repo); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if len(repo.topics) != 1 {
		t.Fatalf("ensure created duplicates: %d topics", len(repo.topics))
	}
}

func TestEnsureDefaultTopic_RenamesLegacyEnglishName(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeRepo{topics: map[string]domain.Topic{
		domain.DefaultKey: {Key: domain.DefaultKey, Name: "Default", Enabled: true, CreatedAt: now, UpdatedAt: now},
	}}
	ctx := context.Background()

	if err := EnsureDefaultTopic(ctx, repo); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got, err := repo.Get(ctx, domain.DefaultKey)
	if err != nil {
		t.Fatalf("default topic missing: %v", err)
	}
	if got.Name != DefaultName {
		t.Fatalf("default name not renamed: got %q, want %q", got.Name, DefaultName)
	}
}

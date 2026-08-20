package topic

import (
	"context"
	"testing"
)

type fakeRepo struct {
	topics map[string]Topic
}

func (f *fakeRepo) List(_ context.Context, _ *bool) ([]Topic, error) {
	out := make([]Topic, 0, len(f.topics))
	for _, t := range f.topics {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, key string) (*Topic, error) {
	t, ok := f.topics[key]
	if !ok {
		return nil, ErrTopicNotFound
	}
	return &t, nil
}

func (f *fakeRepo) Create(_ context.Context, t Topic) error {
	if _, ok := f.topics[t.Key]; ok {
		return ErrTopicConflict
	}
	f.topics[t.Key] = t
	return nil
}

func (f *fakeRepo) Update(_ context.Context, t Topic) error {
	f.topics[t.Key] = t
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, key string) error {
	delete(f.topics, key)
	return nil
}

func TestEnsureDefaultTopic_CreatesOnce(t *testing.T) {
	repo := &fakeRepo{topics: map[string]Topic{}}
	ctx := context.Background()

	if err := EnsureDefaultTopic(ctx, repo); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	got, err := repo.Get(ctx, DefaultKey)
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

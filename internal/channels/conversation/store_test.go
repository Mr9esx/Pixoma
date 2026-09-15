package conversation

import (
	"testing"

	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
)

func TestActionStoreConsumesOneShotToken(t *testing.T) {
	store := NewActionStore()
	token := store.Put(protocol.CapabilityInvoke{CapabilityID: "open_case"})
	if token == "" {
		t.Fatal("empty token")
	}
	if _, ok := store.Get(token); !ok {
		t.Fatal("stored token missing")
	}
	if _, ok := store.Take(token); !ok {
		t.Fatal("token was not consumed")
	}
	if _, ok := store.Get(token); ok {
		t.Fatal("consumed token retained")
	}
}

func TestBackStackReturnsNestedCardSource(t *testing.T) {
	stack := NewBackStack()
	stack.Push("tg-1:123", "root")
	stack.Push("tg-1:123", "card-1")
	if got, ok := stack.Top("tg-1:123"); !ok || got != "card-1" {
		t.Fatalf("top = %q, %v", got, ok)
	}
	if got, ok := stack.Pop("tg-1:123"); !ok || got != "card-1" {
		t.Fatalf("pop = %q, %v", got, ok)
	}
	if got, ok := stack.Top("tg-1:123"); !ok || got != "root" {
		t.Fatalf("top after pop = %q, %v", got, ok)
	}
}

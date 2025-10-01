package models

import (
	"errors"
	"testing"
)

func TestLedgerStoreUndoRedo(t *testing.T) {
	store := NewLedgerStore()

	if store.CanUndo() {
		t.Fatalf("expected no undo available initially")
	}

	if _, err := store.CreateEntry(LedgerTypeDevice, LedgerEntry{Name: "Gateway"}, "tester"); err != nil {
		t.Fatalf("create entry failed: %v", err)
	}
	if !store.CanUndo() {
		t.Fatalf("expected undo available after create")
	}
	if got := len(store.ListEntries(LedgerTypeDevice)); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}

	if err := store.Undo(); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if store.CanRedo() == false {
		t.Fatalf("expected redo available after undo")
	}
	if got := len(store.ListEntries(LedgerTypeDevice)); got != 0 {
		t.Fatalf("expected entries cleared after undo, got %d", got)
	}
	if err := store.Redo(); err != nil {
		t.Fatalf("redo failed: %v", err)
	}
	if got := len(store.ListEntries(LedgerTypeDevice)); got != 1 {
		t.Fatalf("expected entries restored after redo, got %d", got)
	}
}

func TestLedgerStoreLoginChallengeLifecycle(t *testing.T) {
	store := NewLedgerStore()
	challenge := store.CreateLoginChallenge()
	if challenge.Nonce == "" {
		t.Fatalf("expected nonce to be generated")
	}
	if challenge.Message == "" {
		t.Fatalf("expected message to be populated")
	}
	consumed, err := store.ConsumeLoginChallenge(challenge.Nonce)
	if err != nil {
		t.Fatalf("consume challenge: %v", err)
	}
	if consumed.Nonce != challenge.Nonce {
		t.Fatalf("expected consumed nonce to match original")
	}
	if _, err := store.ConsumeLoginChallenge(challenge.Nonce); !errors.Is(err, ErrLoginChallengeNotFound) {
		t.Fatalf("expected challenge to be single-use, got %v", err)
	}
}

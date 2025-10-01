package models

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestLedgerStoreUndoRedo(t *testing.T) {
	store := NewLedgerStore([]byte("secret"))

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

func TestLedgerStoreEnrollmentAndLogin(t *testing.T) {
	store := NewLedgerStore([]byte("secret"))

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	enrollment, err := store.StartEnrollment("alice", "Laptop", pub)
	if err != nil {
		t.Fatalf("start enrollment: %v", err)
	}
	sig := ed25519.Sign(priv, []byte(enrollment.Nonce))
	if _, err := store.CompleteEnrollment("alice", enrollment.DeviceID, enrollment.Nonce, sig, "fingerprint-1"); err != nil {
		t.Fatalf("complete enrollment: %v", err)
	}
	if _, err := store.AppendAllowlist(&IPAllowlistEntry{CIDR: "0.0.0.0/0", Label: "any"}, "system"); err != nil {
		t.Fatalf("allowlist: %v", err)
	}

	challenge, err := store.RequestLoginNonce("alice", enrollment.DeviceID)
	if err != nil {
		t.Fatalf("request nonce: %v", err)
	}
	loginSig := ed25519.Sign(priv, []byte(challenge.Nonce))
	if _, err := store.ValidateLogin("alice", enrollment.DeviceID, challenge.Nonce, loginSig, "fingerprint-1", "192.168.1.10"); err != nil {
		t.Fatalf("validate login: %v", err)
	}

	challenge2, err := store.RequestLoginNonce("alice", enrollment.DeviceID)
	if err != nil {
		t.Fatalf("request nonce 2: %v", err)
	}
	loginSig2 := ed25519.Sign(priv, []byte(challenge2.Nonce))
	if _, err := store.ValidateLogin("alice", enrollment.DeviceID, challenge2.Nonce, loginSig2, "fingerprint-bad", "192.168.1.10"); err == nil {
		t.Fatalf("expected fingerprint mismatch error")
	}
}

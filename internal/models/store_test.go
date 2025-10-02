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

	if _, err := store.CreateEntry(LedgerTypeSystem, LedgerEntry{Name: "审批平台"}, "tester"); err != nil {
		t.Fatalf("create entry failed: %v", err)
	}
	if !store.CanUndo() {
		t.Fatalf("expected undo available after create")
	}
	if got := len(store.ListEntries(LedgerTypeSystem)); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}

	if err := store.Undo(); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if store.CanRedo() == false {
		t.Fatalf("expected redo available after undo")
	}
	if got := len(store.ListEntries(LedgerTypeSystem)); got != 0 {
		t.Fatalf("expected entries cleared after undo, got %d", got)
	}
	if err := store.Redo(); err != nil {
		t.Fatalf("redo failed: %v", err)
	}
	if got := len(store.ListEntries(LedgerTypeSystem)); got != 1 {
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

func TestWorkspaceLifecycle(t *testing.T) {
	store := NewLedgerStore()

	created, err := store.CreateWorkspace("需求汇总", []WorkspaceColumn{{Title: "事项"}}, nil, "<p>初始说明</p>", "tester")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected workspace id")
	}
	if got := len(created.Columns); got != 1 {
		t.Fatalf("expected 1 column, got %d", got)
	}
	colID := created.Columns[0].ID
	if colID == "" {
		t.Fatalf("expected column id to be populated")
	}

	update := WorkspaceUpdate{
		SetRows: true,
		Rows: []WorkspaceRow{{
			ID:    "",
			Cells: map[string]string{colID: "整理 VPN 账号"},
		}},
	}
	updated, err := store.UpdateWorkspace(created.ID, update, "tester")
	if err != nil {
		t.Fatalf("update workspace rows: %v", err)
	}
	if got := len(updated.Rows); got != 1 {
		t.Fatalf("expected 1 row after update, got %d", got)
	}
	if value := updated.Rows[0].Cells[colID]; value != "整理 VPN 账号" {
		t.Fatalf("unexpected cell value: %q", value)
	}

	headers := []string{"负责人", "计划"}
	records := [][]string{{"王五", "本周内完成"}}
	replaced, err := store.ReplaceWorkspaceData(created.ID, headers, records, "tester")
	if err != nil {
		t.Fatalf("replace workspace data: %v", err)
	}
	if got := len(replaced.Columns); got != 2 {
		t.Fatalf("expected 2 columns after import, got %d", got)
	}
	if got := replaced.Columns[0].Title; got != "负责人" {
		t.Fatalf("unexpected column title: %q", got)
	}
	if got := len(replaced.Rows); got != 1 {
		t.Fatalf("expected 1 row after import, got %d", got)
	}
	if value := replaced.Rows[0].Cells[replaced.Columns[1].ID]; value != "本周内完成" {
		t.Fatalf("unexpected imported cell value: %q", value)
	}

	if err := store.DeleteWorkspace(created.ID, "tester"); err != nil {
		t.Fatalf("delete workspace: %v", err)
	}
	if _, err := store.GetWorkspace(created.ID); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("expected workspace to be removed, got %v", err)
	}
}

package models

import (
	"bytes"
	"encoding/json"
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

	created, err := store.CreateWorkspace("需求汇总", WorkspaceKindSheet, "", []WorkspaceColumn{{Title: "事项"}}, nil, "<p>初始说明</p>", "tester")
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

func TestWorkspaceHierarchy(t *testing.T) {
	store := NewLedgerStore()

	folder, err := store.CreateWorkspace("专项项目", WorkspaceKindFolder, "", nil, nil, "", "tester")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	doc, err := store.CreateWorkspace("项目说明", WorkspaceKindDocument, folder.ID, nil, nil, "<p>说明</p>", "tester")
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	sheet, err := store.CreateWorkspace(
		"任务清单",
		WorkspaceKindSheet,
		folder.ID,
		[]WorkspaceColumn{{Title: "任务"}},
		nil,
		"",
		"tester",
	)
	if err != nil {
		t.Fatalf("create sheet: %v", err)
	}

	if _, err := store.UpdateWorkspace(doc.ID, WorkspaceUpdate{SetParent: true, ParentID: ""}, "tester"); err != nil {
		t.Fatalf("move document to root: %v", err)
	}

	if _, err := store.UpdateWorkspace(doc.ID, WorkspaceUpdate{SetColumns: true, Columns: []WorkspaceColumn{}}, "tester"); !errors.Is(err, ErrWorkspaceKindUnsupported) {
		t.Fatalf("expected unsupported kind error when updating columns, got %v", err)
	}
	if _, err := store.ReplaceWorkspaceData(doc.ID, []string{"A"}, [][]string{{"1"}}, "tester"); !errors.Is(err, ErrWorkspaceKindUnsupported) {
		t.Fatalf("expected unsupported kind error when importing document data, got %v", err)
	}

	if err := store.DeleteWorkspace(folder.ID, "tester"); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	if _, err := store.GetWorkspace(folder.ID); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("expected folder removed, got %v", err)
	}
	if _, err := store.GetWorkspace(sheet.ID); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("expected sheet removed with folder, got %v", err)
	}
	if _, err := store.GetWorkspace(doc.ID); err != nil {
		t.Fatalf("expected document to remain after folder deletion, got %v", err)
	}

	if _, err := store.UpdateWorkspace(doc.ID, WorkspaceUpdate{SetParent: true, ParentID: doc.ID}, "tester"); !errors.Is(err, ErrWorkspaceParentInvalid) {
		t.Fatalf("expected invalid parent error when assigning self, got %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	store := NewLedgerStore()

	const firstNewPassword = "StrongerPwd1!"
	if err := store.ChangePassword(defaultAdminUsername, defaultAdminPassword, firstNewPassword, defaultAdminUsername); err != nil {
		t.Fatalf("change password failed: %v", err)
	}

	if _, err := store.AuthenticateUser(defaultAdminUsername, defaultAdminPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected old password to be rejected, got %v", err)
	}

	if _, err := store.AuthenticateUser(defaultAdminUsername, firstNewPassword); err != nil {
		t.Fatalf("expected new password to succeed, got %v", err)
	}

	if err := store.ChangePassword(defaultAdminUsername, "WrongPassword1!", "AnotherPwd2@", defaultAdminUsername); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials error for wrong old password, got %v", err)
	}

	if err := store.ChangePassword(defaultAdminUsername, firstNewPassword, "short", defaultAdminUsername); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected password length validation error, got %v", err)
	}

	if err := store.ChangePassword(defaultAdminUsername, firstNewPassword, "FinalPwd3#", defaultAdminUsername); err != nil {
		t.Fatalf("change password to final value failed: %v", err)
	}

	if _, err := store.AuthenticateUser(defaultAdminUsername, "FinalPwd3#"); err != nil {
		t.Fatalf("expected final password to succeed, got %v", err)
	}
}

func TestWriteSnapshotJSONRoundTrip(t *testing.T) {
	store := NewLedgerStore()
	if _, err := store.CreateEntry(LedgerTypeSystem, LedgerEntry{Name: "日志平台", Description: "集中收集日志"}, "tester"); err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := store.AppendAllowlist(&IPAllowlistEntry{Label: "总部办公网", CIDR: "192.168.0.0/24"}, "tester"); err != nil {
		t.Fatalf("append allowlist: %v", err)
	}
	columns := []WorkspaceColumn{{ID: "col_task", Title: "任务"}, {ID: "col_owner", Title: "负责人"}}
	rows := []WorkspaceRow{{Cells: map[string]string{"col_task": "梳理资产", "col_owner": "刘伟"}}}
	if _, err := store.CreateWorkspace("安全周报", WorkspaceKindSheet, "", columns, rows, "<p>本周重点</p>", "tester"); err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	expected := store.ExportSnapshot()
	var buf bytes.Buffer
	if err := store.WriteSnapshotJSON(&buf); err != nil {
		t.Fatalf("write snapshot json: %v", err)
	}

	var decoded Snapshot
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}

	expectedJSON := marshalSnapshot(t, expected)
	decodedJSON := marshalSnapshot(t, &decoded)
	if expectedJSON != decodedJSON {
		t.Fatalf("snapshot mismatch after round trip")
	}
}
func marshalSnapshot(t *testing.T, snap *Snapshot) string {
	t.Helper()
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	return string(data)
}

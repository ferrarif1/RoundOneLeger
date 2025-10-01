package api

import (
	"testing"

	"ledger/internal/models"
	"ledger/internal/xlsx"
)

func TestParseLedgerSheetDetectsIP(t *testing.T) {
	sheet := xlsx.Sheet{
		Name: "IP",
		Rows: [][]string{
			{"", ""},
			{"", "192.168.0.10"},
		},
	}
	entries := parseLedgerSheet(models.LedgerTypeIP, sheet)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Attributes["address"] != "192.168.0.10" {
		t.Fatalf("expected address attribute to be detected")
	}
	if entries[0].Name != "192.168.0.10" {
		t.Fatalf("expected entry name to default to IP value, got %s", entries[0].Name)
	}
}

func TestBuildCartesianRows(t *testing.T) {
	store := models.NewLedgerStore()
	ip, err := store.CreateEntry(models.LedgerTypeIP, models.LedgerEntry{Name: "Gateway IP"}, "tester")
	if err != nil {
		t.Fatalf("create ip: %v", err)
	}
	device, err := store.CreateEntry(models.LedgerTypeDevice, models.LedgerEntry{Name: "Firewall"}, "tester")
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	person, err := store.CreateEntry(models.LedgerTypePersonnel, models.LedgerEntry{Name: "Alice"}, "tester")
	if err != nil {
		t.Fatalf("create personnel: %v", err)
	}
	system, err := store.CreateEntry(models.LedgerTypeSystem, models.LedgerEntry{Name: "ERP"}, "tester")
	if err != nil {
		t.Fatalf("create system: %v", err)
	}
	_, err = store.UpdateEntry(models.LedgerTypeDevice, device.ID, models.LedgerEntry{Links: map[models.LedgerType][]string{
		models.LedgerTypeIP:        {ip.ID},
		models.LedgerTypePersonnel: {person.ID},
		models.LedgerTypeSystem:    {system.ID},
	}}, "tester")
	if err != nil {
		t.Fatalf("update device links: %v", err)
	}
	server := &Server{Store: store}
	rows := server.buildCartesianRows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	expected := []string{"Gateway IP", "Firewall", "Alice", "ERP"}
	for i, value := range expected {
		if rows[0][i] != value {
			t.Fatalf("unexpected value at column %d: %s", i, rows[0][i])
		}
	}
}

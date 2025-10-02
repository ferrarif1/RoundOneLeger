package xlsx

import "testing"

func TestEncodeDecodeWorkbook(t *testing.T) {
	wb := Workbook{Sheets: []Sheet{{Name: "Systems", Rows: [][]string{{"ID", "Name"}, {"1", "审批台账"}}}}}
	data, err := Encode(wb)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(decoded.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(decoded.Sheets))
	}
	if decoded.Sheets[0].Name != "Systems" {
		t.Fatalf("unexpected sheet name: %s", decoded.Sheets[0].Name)
	}
	if got := decoded.Sheets[0].Rows[1][1]; got != "审批台账" {
		t.Fatalf("unexpected cell value: %s", got)
	}
}

func TestSheetByName(t *testing.T) {
	wb := Workbook{Sheets: []Sheet{{Name: "IP"}, {Name: "Matrix"}}}
	if _, ok := wb.SheetByName("ip"); !ok {
		t.Fatalf("expected to find sheet by case-insensitive name")
	}
}

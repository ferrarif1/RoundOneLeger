package docx

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := `<p>第一段落</p><p>第二段<br />换行</p>`
	encoded, err := EncodeFromHTML(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatalf("expected encoded payload")
	}
	decoded, err := DecodeToHTML(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Contains([]byte(decoded), []byte("第一段落")) {
		t.Fatalf("expected first paragraph in decoded content, got %q", decoded)
	}
	if !bytes.Contains([]byte(decoded), []byte("第二段")) {
		t.Fatalf("expected second paragraph in decoded content, got %q", decoded)
	}
}

func TestDecodeInvalidDocument(t *testing.T) {
	if _, err := DecodeToHTML([]byte("not a docx")); err == nil {
		t.Fatalf("expected decode to fail for invalid archive")
	}
}

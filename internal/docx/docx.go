package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
)

const (
	relationshipContent = `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	contentTypes = `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
)

var (
	brPattern           = regexp.MustCompile(`(?i)<br\s*/?>`)
	closingBlockPattern = regexp.MustCompile(`(?i)</(p|div|section|li|h[1-6]|tr)>`)
	openingBlockPattern = regexp.MustCompile(`(?i)<(p|div|section|li|h[1-6]|tr)[^>]*>`)
	genericTagPattern   = regexp.MustCompile(`(?s)<[^>]+>`)
)

// DecodeToHTML converts a DOCX payload into a lightweight HTML string comprised of paragraph blocks.
func DecodeToHTML(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	reader := bytes.NewReader(data)
	archive, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("docx_open: %w", err)
	}
	var documentFile *zip.File
	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			documentFile = file
			break
		}
	}
	if documentFile == nil {
		return "", fmt.Errorf("docx_document_missing")
	}
	rc, err := documentFile.Open()
	if err != nil {
		return "", fmt.Errorf("docx_read: %w", err)
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	paragraphs := make([]string, 0, 8)
	var builder strings.Builder
	var inParagraph bool
	var inText bool
	preserveSpace := false

	for {
		token, decodeErr := decoder.Token()
		if decodeErr == io.EOF {
			break
		}
		if decodeErr != nil {
			return "", fmt.Errorf("docx_decode: %w", decodeErr)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "p":
				if inParagraph {
					if text := strings.TrimSpace(builder.String()); text != "" {
						paragraphs = append(paragraphs, text)
					}
					builder.Reset()
				}
				inParagraph = true
			case "t":
				if !inParagraph {
					continue
				}
				inText = true
				preserveSpace = false
				for _, attr := range tok.Attr {
					if attr.Name.Space == "xml" && attr.Name.Local == "space" && attr.Value == "preserve" {
						preserveSpace = true
						break
					}
				}
			case "br":
				if inParagraph {
					builder.WriteString("\n")
				}
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "p":
				if inParagraph {
					text := builder.String()
					text = strings.TrimSpace(strings.ReplaceAll(text, "\r", ""))
					if text != "" {
						paragraphs = append(paragraphs, text)
					}
					builder.Reset()
				}
				inParagraph = false
			case "t":
				inText = false
			}
		case xml.CharData:
			if inParagraph && inText {
				chunk := string([]byte(tok))
				if !preserveSpace {
					chunk = strings.ReplaceAll(chunk, "\r", "")
				}
				builder.WriteString(chunk)
			}
		}
	}

	if len(paragraphs) == 0 {
		return "", nil
	}
	htmlParts := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		escaped := html.EscapeString(paragraph)
		escaped = strings.ReplaceAll(escaped, "\n", "<br />")
		htmlParts = append(htmlParts, "<p>"+escaped+"</p>")
	}
	return strings.Join(htmlParts, ""), nil
}

// EncodeFromHTML converts a HTML payload into a DOCX document.
func EncodeFromHTML(content string) ([]byte, error) {
	paragraphs := htmlToParagraphs(content)
	if len(paragraphs) == 0 {
		paragraphs = []string{""}
	}
	documentXML, err := buildDocumentXML(paragraphs)
	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}
	archive := zip.NewWriter(buffer)

	if err := writeZipFile(archive, "[Content_Types].xml", []byte(contentTypes)); err != nil {
		return nil, err
	}
	if err := writeZipFile(archive, "_rels/.rels", []byte(relationshipContent)); err != nil {
		return nil, err
	}
	if err := writeZipFile(archive, "word/document.xml", []byte(documentXML)); err != nil {
		return nil, err
	}

	if err := archive.Close(); err != nil {
		return nil, fmt.Errorf("docx_close: %w", err)
	}
	return buffer.Bytes(), nil
}

func writeZipFile(archive *zip.Writer, name string, data []byte) error {
	writer, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("docx_zip_entry: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("docx_zip_write: %w", err)
	}
	return nil
}

func buildDocumentXML(paragraphs []string) (string, error) {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	builder.WriteString(`<w:body>`)
	for _, paragraph := range paragraphs {
		text := strings.ReplaceAll(paragraph, "\r", "")
		if text == "" {
			builder.WriteString(`<w:p/>`)
			continue
		}
		builder.WriteString(`<w:p><w:r>`)
		segments := strings.Split(text, "\n")
		for idx, segment := range segments {
			builder.WriteString(`<w:t>`)
			if err := xml.EscapeText(&builder, []byte(segment)); err != nil {
				return "", fmt.Errorf("docx_escape: %w", err)
			}
			builder.WriteString(`</w:t>`)
			if idx < len(segments)-1 {
				builder.WriteString(`<w:br/>`)
			}
		}
		builder.WriteString(`</w:r></w:p>`)
	}
	builder.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>`)
	builder.WriteString(`</w:body></w:document>`)
	return builder.String(), nil
}

func htmlToParagraphs(input string) []string {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return nil
	}
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = brPattern.ReplaceAllString(normalized, "\n")
	normalized = closingBlockPattern.ReplaceAllString(normalized, "\n\n")
	normalized = openingBlockPattern.ReplaceAllString(normalized, "")
	normalized = genericTagPattern.ReplaceAllString(normalized, "")
	normalized = html.UnescapeString(normalized)

	lines := strings.Split(normalized, "\n")
	paragraphs := make([]string, 0, len(lines))
	var builder strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if builder.Len() > 0 {
				paragraphs = append(paragraphs, strings.TrimSpace(builder.String()))
				builder.Reset()
			}
			continue
		}
		if builder.Len() > 0 {
			builder.WriteRune(' ')
		}
		builder.WriteString(trimmed)
	}
	if builder.Len() > 0 {
		paragraphs = append(paragraphs, strings.TrimSpace(builder.String()))
	}
	return paragraphs
}

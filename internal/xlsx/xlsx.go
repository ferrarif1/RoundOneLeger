package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
)

type Workbook struct {
	Sheets []Sheet
}

type Sheet struct {
	Name string
	Rows [][]string
}

func Parse(data []byte) (*Workbook, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	workbook := &Workbook{}

	sharedStrings, err := parseSharedStrings(reader)
	if err != nil {
		return nil, err
	}

	sheetsInfo, err := parseWorkbookRelationships(reader)
	if err != nil {
		return nil, err
	}

	for _, info := range sheetsInfo {
		file, err := reader.Open(info.Path)
		if err != nil {
			return nil, fmt.Errorf("open sheet %s: %w", info.Name, err)
		}
		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("read sheet %s: %w", info.Name, err)
		}

		rows, err := parseSheet(content, sharedStrings)
		if err != nil {
			return nil, fmt.Errorf("parse sheet %s: %w", info.Name, err)
		}

		workbook.Sheets = append(workbook.Sheets, Sheet{Name: info.Name, Rows: rows})
	}

	return workbook, nil
}

func NewWorkbook() *Workbook {
	return &Workbook{}
}

func (w *Workbook) AddSheet(name string, rows [][]string) {
	clone := make([][]string, len(rows))
	for i, row := range rows {
		rowClone := make([]string, len(row))
		copy(rowClone, row)
		clone[i] = rowClone
	}
	w.Sheets = append(w.Sheets, Sheet{Name: name, Rows: clone})
}

func (w *Workbook) Rows(name string) ([][]string, bool) {
	for _, sheet := range w.Sheets {
		if strings.EqualFold(sheet.Name, name) {
			clone := make([][]string, len(sheet.Rows))
			for i, row := range sheet.Rows {
				rowClone := make([]string, len(row))
				copy(rowClone, row)
				clone[i] = rowClone
			}
			return clone, true
		}
	}
	return nil, false
}

func (w *Workbook) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	if err := writeFile(zw, "[Content_Types].xml", buildContentTypes(len(w.Sheets))); err != nil {
		return nil, err
	}
	if err := writeFile(zw, "_rels/.rels", []byte(rootRels)); err != nil {
		return nil, err
	}
	if err := writeFile(zw, "xl/workbook.xml", buildWorkbookXML(w.Sheets)); err != nil {
		return nil, err
	}
	if err := writeFile(zw, "xl/_rels/workbook.xml.rels", buildWorkbookRels(len(w.Sheets))); err != nil {
		return nil, err
	}

	for i, sheet := range w.Sheets {
		sheetPath := fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
		if err := writeFile(zw, sheetPath, buildSheetXML(sheet.Rows)); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return buf.Bytes(), nil
}

func writeFile(zw *zip.Writer, name string, data []byte) error {
	writer, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

type sheetInfo struct {
	Name string
	Path string
}

func parseSharedStrings(reader *zip.Reader) ([]string, error) {
	file, err := reader.Open("xl/sharedStrings.xml")
	if err != nil {
		return nil, nil
	}
	defer file.Close()

	type text struct {
		T string `xml:"t"`
	}
	type si struct {
		Text text `xml:"t"`
		IS   struct {
			Text text `xml:"t"`
		} `xml:"is"`
	}
	type sharedStrings struct {
		Items []si `xml:"si"`
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read shared strings: %w", err)
	}
	var doc sharedStrings
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse shared strings: %w", err)
	}

	strings := make([]string, len(doc.Items))
	for i, item := range doc.Items {
		if item.Text.T != "" {
			strings[i] = item.Text.T
		} else {
			strings[i] = item.IS.Text.T
		}
	}
	return strings, nil
}

func parseWorkbookRelationships(reader *zip.Reader) ([]sheetInfo, error) {
	workbookFile, err := reader.Open("xl/workbook.xml")
	if err != nil {
		return nil, fmt.Errorf("workbook.xml missing: %w", err)
	}
	workbookData, err := io.ReadAll(workbookFile)
	workbookFile.Close()
	if err != nil {
		return nil, fmt.Errorf("read workbook.xml: %w", err)
	}

	type sheet struct {
		Name  string
		ID    string
		Order int
	}
	sheets := []sheet{}
	decoder := xml.NewDecoder(bytes.NewReader(workbookData))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse workbook.xml: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "sheet" {
			continue
		}
		item := sheet{}
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "name":
				item.Name = attr.Value
			case "id":
				item.ID = attr.Value
			case "sheetId":
				if v, err := strconv.Atoi(attr.Value); err == nil {
					item.Order = v
				}
			}
		}
		if item.Name != "" && item.ID != "" {
			sheets = append(sheets, item)
		}
	}
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets defined")
	}

	relFile, err := reader.Open("xl/_rels/workbook.xml.rels")
	if err != nil {
		return nil, fmt.Errorf("workbook relations missing: %w", err)
	}
	relData, err := io.ReadAll(relFile)
	relFile.Close()
	if err != nil {
		return nil, fmt.Errorf("read workbook relations: %w", err)
	}

	type relationship struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	}
	type relationships struct {
		Relations []relationship `xml:"Relationship"`
	}
	var rels relationships
	if err := xml.Unmarshal(relData, &rels); err != nil {
		return nil, fmt.Errorf("parse workbook relations: %w", err)
	}

	idToTarget := map[string]string{}
	for _, rel := range rels.Relations {
		idToTarget[rel.ID] = rel.Target
	}

	sort.Slice(sheets, func(i, j int) bool {
		return sheets[i].Order < sheets[j].Order
	})

	infos := make([]sheetInfo, 0, len(sheets))
	for _, sheet := range sheets {
		target, ok := idToTarget[sheet.ID]
		if !ok {
			continue
		}
		sheetPath := path.Clean(path.Join("xl", target))
		infos = append(infos, sheetInfo{Name: sheet.Name, Path: sheetPath})
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("no sheet targets resolved")
	}
	return infos, nil
}

func parseSheet(data []byte, sharedStrings []string) ([][]string, error) {
	type cell struct {
		Ref    string `xml:"r,attr"`
		Type   string `xml:"t,attr"`
		Value  string `xml:"v"`
		Inline struct {
			Text string `xml:"t"`
		} `xml:"is"`
	}
	type row struct {
		R     int    `xml:"r,attr"`
		Cells []cell `xml:"c"`
	}
	type sheet struct {
		Rows []row `xml:"sheetData>row"`
	}

	var sh sheet
	if err := xml.Unmarshal(data, &sh); err != nil {
		return nil, fmt.Errorf("unmarshal sheet: %w", err)
	}

	resolve := func(c cell) string {
		switch c.Type {
		case "s":
			idx, err := strconv.Atoi(strings.TrimSpace(c.Value))
			if err == nil && idx >= 0 && idx < len(sharedStrings) {
				return sharedStrings[idx]
			}
			return ""
		case "inlineStr":
			return c.Inline.Text
		default:
			return c.Value
		}
	}

	rows := make([][]string, len(sh.Rows))
	for i, row := range sh.Rows {
		maxCol := 0
		cols := make(map[int]string)
		for _, cell := range row.Cells {
			col := columnIndex(cell.Ref)
			if col > maxCol {
				maxCol = col
			}
			cols[col] = resolve(cell)
		}
		line := make([]string, maxCol)
		for c := 1; c <= maxCol; c++ {
			line[c-1] = cols[c]
		}
		rows[i] = line
	}
	return rows, nil
}

func columnIndex(ref string) int {
	ref = strings.ToUpper(ref)
	letters := 0
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			letters = letters*26 + int(r-'A'+1)
		} else {
			break
		}
	}
	if letters == 0 {
		return 1
	}
	return letters
}

func buildContentTypes(sheetCount int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	b.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	for i := 1; i <= sheetCount; i++ {
		fmt.Fprintf(&b, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i)
	}
	b.WriteString(`</Types>`)
	return []byte(b.String())
}

const rootRels = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`

func buildWorkbookXML(sheets []Sheet) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	b.WriteString(`<sheets>`)
	for i, sheet := range sheets {
		fmt.Fprintf(&b, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(sheet.Name), i+1, i+1)
	}
	b.WriteString(`</sheets>`)
	b.WriteString(`</workbook>`)
	return []byte(b.String())
}

func buildWorkbookRels(sheetCount int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= sheetCount; i++ {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i)
	}
	b.WriteString(`</Relationships>`)
	return []byte(b.String())
}

func buildSheetXML(rows [][]string) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	b.WriteString(`<sheetData>`)
	for i, row := range rows {
		rIdx := i + 1
		b.WriteString(fmt.Sprintf(`<row r="%d">`, rIdx))
		for j, value := range row {
			if value == "" {
				continue
			}
			colRef := columnLetters(j + 1)
			b.WriteString(`<c t="inlineStr" r="` + colRef + fmt.Sprint(rIdx) + `"><is><t>`) // xml escape
			b.WriteString(xmlEscape(value))
			b.WriteString(`</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData>`)
	b.WriteString(`</worksheet>`)
	return []byte(b.String())
}

func columnLetters(idx int) string {
	result := ""
	for idx > 0 {
		idx--
		result = string(rune('A'+idx%26)) + result
		idx /= 26
	}
	return result
}

func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

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

// Workbook represents a simplified XLSX workbook with inline string cells.
type Workbook struct {
	Sheets []Sheet
}

// Sheet represents a sheet with ordered rows and columns.
type Sheet struct {
	Name string
	Rows [][]string
}

// Encode produces an XLSX binary containing the workbook data.
func Encode(wb Workbook) ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	if err := writeFile(zw, "[Content_Types].xml", contentTypesXML(len(wb.Sheets))); err != nil {
		return nil, err
	}
	if err := writeFile(zw, "_rels/.rels", rootRelsXML); err != nil {
		return nil, err
	}
	if err := writeFile(zw, "xl/workbook.xml", workbookXML(wb)); err != nil {
		return nil, err
	}
	if err := writeFile(zw, "xl/_rels/workbook.xml.rels", workbookRelsXML(len(wb.Sheets))); err != nil {
		return nil, err
	}
	for i, sheet := range wb.Sheets {
		name := fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
		if err := writeFile(zw, name, sheetXML(sheet)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func contentTypesXML(sheetCount int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	b.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	for i := 1; i <= sheetCount; i++ {
		b.WriteString(fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i))
	}
	b.WriteString(`</Types>`)
	return []byte(b.String())
}

var rootRelsXML = []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
	`</Relationships>`)

func workbookXML(wb Workbook) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" `)
	b.WriteString(`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	b.WriteString(`<sheets>`)
	for i, sheet := range wb.Sheets {
		b.WriteString(fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, escapeXML(sheet.Name), i+1, i+1))
	}
	b.WriteString(`</sheets></workbook>`)
	return []byte(b.String())
}

func workbookRelsXML(sheetCount int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= sheetCount; i++ {
		b.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i))
	}
	b.WriteString(`</Relationships>`)
	return []byte(b.String())
}

func sheetXML(sheet Sheet) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	b.WriteString(`<sheetData>`)
	for i, row := range sheet.Rows {
		b.WriteString(fmt.Sprintf(`<row r="%d">`, i+1))
		for j, cell := range row {
			if cell == "" {
				continue
			}
			ref := cellRef(i, j)
			b.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, escapeXML(cell)))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return []byte(b.String())
}

func escapeXML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
}

func cellRef(row, col int) string {
	return columnName(col) + strconv.Itoa(row+1)
}

func columnName(index int) string {
	name := ""
	for index >= 0 {
		rem := index % 26
		name = string(rune('A'+rem)) + name
		index = index/26 - 1
	}
	return name
}

// Decode parses a simplified XLSX workbook into memory.
func Decode(data []byte) (Workbook, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Workbook{}, err
	}
	files := make(map[string]*zip.File)
	for _, f := range reader.File {
		files[path.Clean(f.Name)] = f
	}

	workbookFile, ok := files["xl/workbook.xml"]
	if !ok {
		return Workbook{}, fmt.Errorf("workbook.xml missing")
	}
	workbookData, err := readZipFile(workbookFile)
	if err != nil {
		return Workbook{}, err
	}
	rels := map[string]string{}
	if relFile, ok := files["xl/_rels/workbook.xml.rels"]; ok {
		relData, err := readZipFile(relFile)
		if err != nil {
			return Workbook{}, err
		}
		rels = parseRelationships(relData)
	}
	sharedStrings := []string{}
	if ssFile, ok := files["xl/sharedStrings.xml"]; ok {
		ssData, err := readZipFile(ssFile)
		if err != nil {
			return Workbook{}, err
		}
		sharedStrings = parseSharedStrings(ssData)
	}

	type sheetInfo struct {
		Name string `xml:"name,attr"`
		ID   string `xml:"sheetId,attr"`
		RID  string `xml:"id,attr"`
	}
	type workbookDef struct {
		Sheets []sheetInfo `xml:"sheets>sheet"`
	}
	var wbDef workbookDef
	if err := xml.Unmarshal(workbookData, &wbDef); err != nil {
		return Workbook{}, err
	}
	workbook := Workbook{}
	for _, info := range wbDef.Sheets {
		target := rels[info.RID]
		if target == "" {
			target = fmt.Sprintf("worksheets/sheet%s.xml", info.ID)
		}
		sheetFile, ok := files[path.Clean("xl/"+target)]
		if !ok {
			continue
		}
		sheetData, err := readZipFile(sheetFile)
		if err != nil {
			return Workbook{}, err
		}
		sheet := parseSheet(sheetData, sharedStrings)
		sheet.Name = info.Name
		workbook.Sheets = append(workbook.Sheets, sheet)
	}
	return workbook, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func parseRelationships(data []byte) map[string]string {
	type rel struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	}
	type doc struct {
		Relationships []rel `xml:"Relationship"`
	}
	var d doc
	_ = xml.Unmarshal(data, &d)
	result := make(map[string]string, len(d.Relationships))
	for _, r := range d.Relationships {
		result[r.ID] = r.Target
	}
	return result
}

func parseSharedStrings(data []byte) []string {
	type si struct {
		Text string `xml:"t"`
	}
	type sst struct {
		Items []si `xml:"si"`
	}
	var doc sst
	_ = xml.Unmarshal(data, &doc)
	out := make([]string, len(doc.Items))
	for i, item := range doc.Items {
		out[i] = item.Text
	}
	return out
}

func parseSheet(data []byte, sharedStrings []string) Sheet {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rows := [][]string{}
	currentRow := -1
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "row" {
				currentRow++
				rows = append(rows, []string{})
			}
			if elem.Name.Local == "c" {
				var cell struct {
					R string `xml:"r,attr"`
					T string `xml:"t,attr"`
				}
				cell.R = attr(elem.Attr, "r")
				cell.T = attr(elem.Attr, "t")
				col := columnIndex(cell.R)
				for len(rows[currentRow]) <= col {
					rows[currentRow] = append(rows[currentRow], "")
				}
				value := readCellValue(decoder)
				if cell.T == "s" {
					idx, _ := strconv.Atoi(value)
					if idx >= 0 && idx < len(sharedStrings) {
						value = sharedStrings[idx]
					}
				}
				rows[currentRow][col] = value
			}
		}
	}
	for i := range rows {
		trim := len(rows[i])
		for trim > 0 && rows[i][trim-1] == "" {
			trim--
		}
		rows[i] = rows[i][:trim]
	}
	return Sheet{Rows: rows}
}

func attr(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func readCellValue(decoder *xml.Decoder) string {
	for {
		token, err := decoder.Token()
		if err != nil {
			return ""
		}
		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "v" || elem.Name.Local == "t" {
				var value string
				_ = decoder.DecodeElement(&value, &elem)
				return value
			}
			if elem.Name.Local == "is" {
				var inline struct {
					Text string `xml:"t"`
				}
				_ = decoder.DecodeElement(&inline, &elem)
				return inline.Text
			}
		case xml.EndElement:
			if elem.Name.Local == "c" {
				return ""
			}
		}
	}
}

func columnIndex(ref string) int {
	ref = strings.ToUpper(ref)
	letters := ""
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			letters += string(r)
		} else {
			break
		}
	}
	if letters == "" {
		return 0
	}
	index := 0
	for _, r := range letters {
		index = index*26 + int(r-'A'+1)
	}
	return index - 1
}

// SheetByName retrieves a sheet by name.
func (wb Workbook) SheetByName(name string) (Sheet, bool) {
	for _, sheet := range wb.Sheets {
		if strings.EqualFold(sheet.Name, name) {
			return sheet, true
		}
	}
	return Sheet{}, false
}

// SortSheets sorts sheets by provided order, keeping unspecified ones after.
func (wb *Workbook) SortSheets(order []string) {
	lookup := make(map[string]int, len(order))
	for i, name := range order {
		lookup[strings.ToLower(name)] = i
	}
	sort.SliceStable(wb.Sheets, func(i, j int) bool {
		a := strings.ToLower(wb.Sheets[i].Name)
		b := strings.ToLower(wb.Sheets[j].Name)
		ia, oka := lookup[a]
		ib, okb := lookup[b]
		switch {
		case oka && okb:
			return ia < ib
		case oka:
			return true
		case okb:
			return false
		default:
			return a < b
		}
	})
}

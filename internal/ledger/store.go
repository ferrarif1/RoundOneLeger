package ledger

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"

	"ledger/internal/xlsx"
)

type IPEntry struct {
	ID          int64    `json:"id"`
	Address     string   `json:"address"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

type DeviceEntry struct {
	ID         int64    `json:"id"`
	Identifier string   `json:"identifier"`
	Type       string   `json:"type"`
	Owner      string   `json:"owner"`
	Tags       []string `json:"tags,omitempty"`
}

type PersonnelEntry struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Role    string   `json:"role"`
	Contact string   `json:"contact"`
	Tags    []string `json:"tags,omitempty"`
}

type SystemEntry struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Environment string   `json:"environment"`
	Owner       string   `json:"owner"`
	Tags        []string `json:"tags,omitempty"`
}

type CombinedEntry struct {
	Personnel PersonnelEntry `json:"personnel"`
	Device    DeviceEntry    `json:"device"`
	System    SystemEntry    `json:"system"`
	IP        IPEntry        `json:"ip"`
}

type snapshot struct {
	ips          []IPEntry
	devices      []DeviceEntry
	personnel    []PersonnelEntry
	systems      []SystemEntry
	nextIPID     int64
	nextDeviceID int64
	nextPersonID int64
	nextSystemID int64
}

type HistoryStatus struct {
	UndoSteps int `json:"undoSteps"`
	RedoSteps int `json:"redoSteps"`
}

type LedgerState struct {
	IPs       []IPEntry        `json:"ips"`
	Devices   []DeviceEntry    `json:"devices"`
	Personnel []PersonnelEntry `json:"personnel"`
	Systems   []SystemEntry    `json:"systems"`
}

type Store struct {
	mu           sync.RWMutex
	ips          []IPEntry
	devices      []DeviceEntry
	personnel    []PersonnelEntry
	systems      []SystemEntry
	nextIPID     int64
	nextDeviceID int64
	nextPersonID int64
	nextSystemID int64
	history      []*snapshot
	future       []*snapshot
	historyLimit int
}

func NewStore() *Store {
	return &Store{
		nextIPID:     1,
		nextDeviceID: 1,
		nextPersonID: 1,
		nextSystemID: 1,
		historyLimit: 10,
	}
}

func (s *Store) captureSnapshotLocked() *snapshot {
	return &snapshot{
		ips:          cloneIPEntries(s.ips),
		devices:      cloneDeviceEntries(s.devices),
		personnel:    clonePersonnelEntries(s.personnel),
		systems:      cloneSystemEntries(s.systems),
		nextIPID:     s.nextIPID,
		nextDeviceID: s.nextDeviceID,
		nextPersonID: s.nextPersonID,
		nextSystemID: s.nextSystemID,
	}
}

func (s *Store) applySnapshotLocked(snap *snapshot) {
	s.ips = cloneIPEntries(snap.ips)
	s.devices = cloneDeviceEntries(snap.devices)
	s.personnel = clonePersonnelEntries(snap.personnel)
	s.systems = cloneSystemEntries(snap.systems)
	s.nextIPID = snap.nextIPID
	s.nextDeviceID = snap.nextDeviceID
	s.nextPersonID = snap.nextPersonID
	s.nextSystemID = snap.nextSystemID
}

func (s *Store) pushHistoryLocked() {
	snap := s.captureSnapshotLocked()
	s.history = append(s.history, snap)
	if len(s.history) > s.historyLimit {
		s.history = append([]*snapshot(nil), s.history[len(s.history)-s.historyLimit:]...)
	}
	s.future = nil
}

func (s *Store) currentStateLocked() LedgerState {
	return LedgerState{
		IPs:       cloneIPEntries(s.ips),
		Devices:   cloneDeviceEntries(s.devices),
		Personnel: clonePersonnelEntries(s.personnel),
		Systems:   cloneSystemEntries(s.systems),
	}
}

func (s *Store) statusLocked() HistoryStatus {
	return HistoryStatus{UndoSteps: len(s.history), RedoSteps: len(s.future)}
}

func (s *Store) CurrentState() LedgerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentStateLocked()
}

func (s *Store) HistoryStatus() HistoryStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

func (s *Store) Undo() (LedgerState, HistoryStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.history) == 0 {
		return LedgerState{}, s.statusLocked(), errors.New("no_undo_available")
	}

	current := s.captureSnapshotLocked()
	s.future = append(s.future, current)
	if len(s.future) > s.historyLimit {
		s.future = append([]*snapshot(nil), s.future[len(s.future)-s.historyLimit:]...)
	}

	lastIdx := len(s.history) - 1
	snap := s.history[lastIdx]
	s.history = s.history[:lastIdx]
	s.applySnapshotLocked(snap)

	return s.currentStateLocked(), s.statusLocked(), nil
}

func (s *Store) Redo() (LedgerState, HistoryStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.future) == 0 {
		return LedgerState{}, s.statusLocked(), errors.New("no_redo_available")
	}

	current := s.captureSnapshotLocked()
	s.history = append(s.history, current)
	if len(s.history) > s.historyLimit {
		s.history = append([]*snapshot(nil), s.history[len(s.history)-s.historyLimit:]...)
	}

	lastIdx := len(s.future) - 1
	snap := s.future[lastIdx]
	s.future = s.future[:lastIdx]
	s.applySnapshotLocked(snap)

	return s.currentStateLocked(), s.statusLocked(), nil
}

type IPInput struct {
	Address     string
	Description string
	Tags        []string
}

type DeviceInput struct {
	Identifier string
	Type       string
	Owner      string
	Tags       []string
}

type PersonnelInput struct {
	Name    string
	Role    string
	Contact string
	Tags    []string
}

type SystemInput struct {
	Name        string
	Environment string
	Owner       string
	Tags        []string
}

func (s *Store) GetIPs() []IPEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]IPEntry, len(s.ips))
	for i, entry := range s.ips {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func (s *Store) GetDevices() []DeviceEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DeviceEntry, len(s.devices))
	for i, entry := range s.devices {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func (s *Store) GetPersonnel() []PersonnelEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PersonnelEntry, len(s.personnel))
	for i, entry := range s.personnel {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func (s *Store) GetSystems() []SystemEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SystemEntry, len(s.systems))
	for i, entry := range s.systems {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func (s *Store) AddIP(input IPInput) (IPEntry, error) {
	if strings.TrimSpace(input.Address) == "" {
		return IPEntry{}, errors.New("address_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pushHistoryLocked()

	entry := IPEntry{
		ID:          s.nextIPID,
		Address:     strings.TrimSpace(input.Address),
		Description: strings.TrimSpace(input.Description),
		Tags:        normalizeTags(input.Tags),
	}
	s.nextIPID++
	s.ips = append(s.ips, entry)
	return entry, nil
}

func (s *Store) UpdateIP(id int64, input IPInput) (IPEntry, error) {
	if strings.TrimSpace(input.Address) == "" {
		return IPEntry{}, errors.New("address_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findIPIndex(s.ips, id)
	if idx == -1 {
		return IPEntry{}, errors.New("ip_not_found")
	}

	s.pushHistoryLocked()

	entry := s.ips[idx]
	entry.Address = strings.TrimSpace(input.Address)
	entry.Description = strings.TrimSpace(input.Description)
	entry.Tags = normalizeTags(input.Tags)
	s.ips[idx] = entry
	return entry, nil
}

func (s *Store) DeleteIP(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findIPIndex(s.ips, id)
	if idx == -1 {
		return errors.New("ip_not_found")
	}

	s.pushHistoryLocked()

	s.ips = append(s.ips[:idx], s.ips[idx+1:]...)
	return nil
}

func (s *Store) ReorderIPs(order []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(order) != len(s.ips) {
		return errors.New("order_length_mismatch")
	}

	byID := make(map[int64]IPEntry, len(s.ips))
	for _, entry := range s.ips {
		byID[entry.ID] = entry
	}

	reordered := make([]IPEntry, len(order))
	for i, id := range order {
		entry, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown_ip_id:%d", id)
		}
		reordered[i] = entry
	}

	s.pushHistoryLocked()
	s.ips = reordered
	return nil
}

func (s *Store) AddDevice(input DeviceInput) (DeviceEntry, error) {
	if strings.TrimSpace(input.Identifier) == "" {
		return DeviceEntry{}, errors.New("identifier_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pushHistoryLocked()

	entry := DeviceEntry{
		ID:         s.nextDeviceID,
		Identifier: strings.TrimSpace(input.Identifier),
		Type:       strings.TrimSpace(input.Type),
		Owner:      strings.TrimSpace(input.Owner),
		Tags:       normalizeTags(input.Tags),
	}
	s.nextDeviceID++
	s.devices = append(s.devices, entry)
	return entry, nil
}

func (s *Store) UpdateDevice(id int64, input DeviceInput) (DeviceEntry, error) {
	if strings.TrimSpace(input.Identifier) == "" {
		return DeviceEntry{}, errors.New("identifier_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findDeviceIndex(s.devices, id)
	if idx == -1 {
		return DeviceEntry{}, errors.New("device_not_found")
	}

	s.pushHistoryLocked()

	entry := s.devices[idx]
	entry.Identifier = strings.TrimSpace(input.Identifier)
	entry.Type = strings.TrimSpace(input.Type)
	entry.Owner = strings.TrimSpace(input.Owner)
	entry.Tags = normalizeTags(input.Tags)
	s.devices[idx] = entry
	return entry, nil
}

func (s *Store) DeleteDevice(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findDeviceIndex(s.devices, id)
	if idx == -1 {
		return errors.New("device_not_found")
	}

	s.pushHistoryLocked()

	s.devices = append(s.devices[:idx], s.devices[idx+1:]...)
	return nil
}

func (s *Store) ReorderDevices(order []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(order) != len(s.devices) {
		return errors.New("order_length_mismatch")
	}

	byID := make(map[int64]DeviceEntry, len(s.devices))
	for _, entry := range s.devices {
		byID[entry.ID] = entry
	}

	reordered := make([]DeviceEntry, len(order))
	for i, id := range order {
		entry, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown_device_id:%d", id)
		}
		reordered[i] = entry
	}

	s.pushHistoryLocked()
	s.devices = reordered
	return nil
}

func (s *Store) AddPersonnel(input PersonnelInput) (PersonnelEntry, error) {
	if strings.TrimSpace(input.Name) == "" {
		return PersonnelEntry{}, errors.New("name_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pushHistoryLocked()

	entry := PersonnelEntry{
		ID:      s.nextPersonID,
		Name:    strings.TrimSpace(input.Name),
		Role:    strings.TrimSpace(input.Role),
		Contact: strings.TrimSpace(input.Contact),
		Tags:    normalizeTags(input.Tags),
	}
	s.nextPersonID++
	s.personnel = append(s.personnel, entry)
	return entry, nil
}

func (s *Store) UpdatePersonnel(id int64, input PersonnelInput) (PersonnelEntry, error) {
	if strings.TrimSpace(input.Name) == "" {
		return PersonnelEntry{}, errors.New("name_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findPersonnelIndex(s.personnel, id)
	if idx == -1 {
		return PersonnelEntry{}, errors.New("personnel_not_found")
	}

	s.pushHistoryLocked()

	entry := s.personnel[idx]
	entry.Name = strings.TrimSpace(input.Name)
	entry.Role = strings.TrimSpace(input.Role)
	entry.Contact = strings.TrimSpace(input.Contact)
	entry.Tags = normalizeTags(input.Tags)
	s.personnel[idx] = entry
	return entry, nil
}

func (s *Store) DeletePersonnel(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findPersonnelIndex(s.personnel, id)
	if idx == -1 {
		return errors.New("personnel_not_found")
	}

	s.pushHistoryLocked()

	s.personnel = append(s.personnel[:idx], s.personnel[idx+1:]...)
	return nil
}

func (s *Store) ReorderPersonnel(order []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(order) != len(s.personnel) {
		return errors.New("order_length_mismatch")
	}

	byID := make(map[int64]PersonnelEntry, len(s.personnel))
	for _, entry := range s.personnel {
		byID[entry.ID] = entry
	}

	reordered := make([]PersonnelEntry, len(order))
	for i, id := range order {
		entry, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown_personnel_id:%d", id)
		}
		reordered[i] = entry
	}

	s.pushHistoryLocked()
	s.personnel = reordered
	return nil
}

func (s *Store) AddSystem(input SystemInput) (SystemEntry, error) {
	if strings.TrimSpace(input.Name) == "" {
		return SystemEntry{}, errors.New("name_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pushHistoryLocked()

	entry := SystemEntry{
		ID:          s.nextSystemID,
		Name:        strings.TrimSpace(input.Name),
		Environment: strings.TrimSpace(input.Environment),
		Owner:       strings.TrimSpace(input.Owner),
		Tags:        normalizeTags(input.Tags),
	}
	s.nextSystemID++
	s.systems = append(s.systems, entry)
	return entry, nil
}

func (s *Store) UpdateSystem(id int64, input SystemInput) (SystemEntry, error) {
	if strings.TrimSpace(input.Name) == "" {
		return SystemEntry{}, errors.New("name_required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findSystemIndex(s.systems, id)
	if idx == -1 {
		return SystemEntry{}, errors.New("system_not_found")
	}

	s.pushHistoryLocked()

	entry := s.systems[idx]
	entry.Name = strings.TrimSpace(input.Name)
	entry.Environment = strings.TrimSpace(input.Environment)
	entry.Owner = strings.TrimSpace(input.Owner)
	entry.Tags = normalizeTags(input.Tags)
	s.systems[idx] = entry
	return entry, nil
}

func (s *Store) DeleteSystem(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := findSystemIndex(s.systems, id)
	if idx == -1 {
		return errors.New("system_not_found")
	}

	s.pushHistoryLocked()

	s.systems = append(s.systems[:idx], s.systems[idx+1:]...)
	return nil
}

func (s *Store) ReorderSystems(order []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(order) != len(s.systems) {
		return errors.New("order_length_mismatch")
	}

	byID := make(map[int64]SystemEntry, len(s.systems))
	for _, entry := range s.systems {
		byID[entry.ID] = entry
	}

	reordered := make([]SystemEntry, len(order))
	for i, id := range order {
		entry, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown_system_id:%d", id)
		}
		reordered[i] = entry
	}

	s.pushHistoryLocked()
	s.systems = reordered
	return nil
}

func (s *Store) Combined() []CombinedEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.personnel) == 0 || len(s.devices) == 0 || len(s.systems) == 0 || len(s.ips) == 0 {
		return []CombinedEntry{}
	}

	combined := make([]CombinedEntry, 0, len(s.personnel)*len(s.devices)*len(s.systems)*len(s.ips))
	for _, p := range s.personnel {
		for _, d := range s.devices {
			for _, sys := range s.systems {
				for _, ip := range s.ips {
					combined = append(combined, CombinedEntry{
						Personnel: p,
						Device:    d,
						System:    sys,
						IP:        ip,
					})
				}
			}
		}
	}
	return combined
}

func (s *Store) ImportExcel(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read excel: %w", err)
	}
	wb, err := xlsx.Parse(data)
	if err != nil {
		return err
	}

	ips, err := extractIPs(wb)
	if err != nil {
		return err
	}
	devices, err := extractDevices(wb)
	if err != nil {
		return err
	}
	personnel, err := extractPersonnel(wb)
	if err != nil {
		return err
	}
	systems, err := extractSystems(wb)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pushHistoryLocked()

	s.ips = assignIPIDs(ips, &s.nextIPID)
	s.devices = assignDeviceIDs(devices, &s.nextDeviceID)
	s.personnel = assignPersonnelIDs(personnel, &s.nextPersonID)
	s.systems = assignSystemIDs(systems, &s.nextSystemID)
	return nil
}

func (s *Store) ExportExcel() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wb := xlsx.NewWorkbook()
	wb.AddSheet(sheetIPs, buildIPRows(s.ips))
	wb.AddSheet(sheetDevices, buildDeviceRows(s.devices))
	wb.AddSheet(sheetPersonnel, buildPersonnelRows(s.personnel))
	wb.AddSheet(sheetSystems, buildSystemRows(s.systems))
	wb.AddSheet(sheetCombined, buildCombinedRows(s.personnel, s.devices, s.systems, s.ips))

	data, err := wb.Bytes()
	if err != nil {
		return nil, fmt.Errorf("write workbook: %w", err)
	}
	return data, nil
}

const (
	sheetIPs       = "IPs"
	sheetDevices   = "Devices"
	sheetPersonnel = "Personnel"
	sheetSystems   = "Systems"
	sheetCombined  = "CombinedMatrix"
)

var ipAddressPattern = regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1?[0-9]{1,2})(?:\.(?:25[0-5]|2[0-4][0-9]|1?[0-9]{1,2})){3}|(?:[0-9a-fA-F]{1,4}:){2,7}[0-9a-fA-F]{1,4})$`)

func normalizeHeaderValue(v string) string {
	cleaned := strings.TrimSpace(strings.ToLower(v))
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", "\u00a0", "")
	return replacer.Replace(cleaned)
}

func headerMatches(header string, keywords []string) bool {
	norm := normalizeHeaderValue(header)
	for _, keyword := range keywords {
		if strings.Contains(norm, keyword) {
			return true
		}
	}
	return false
}

func findColumn(headers []string, keywords []string) int {
	for idx, header := range headers {
		if headerMatches(header, keywords) {
			return idx
		}
	}
	return -1
}

func columnValues(rows [][]string, column int) []string {
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) > column {
			values = append(values, row[column])
		} else {
			values = append(values, "")
		}
	}
	return values
}

func findColumnWithDetector(rows [][]string, keywords []string, detector func([]string) bool) int {
	if len(rows) == 0 {
		return -1
	}
	headers := rows[0]
	if idx := findColumn(headers, keywords); idx != -1 {
		return idx
	}
	if detector == nil {
		return -1
	}
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	for column := 0; column < maxCols; column++ {
		if detector(columnValues(rows[1:], column)) {
			return column
		}
	}
	return -1
}

func valueFor(row []string, preferred int, fallback int) string {
	if preferred >= 0 && len(row) > preferred {
		return row[preferred]
	}
	if fallback >= 0 && len(row) > fallback {
		return row[fallback]
	}
	return ""
}

func extractIPs(wb *xlsx.Workbook) ([]IPEntry, error) {
	rows, ok := wb.Rows(sheetIPs)
	if !ok {
		return nil, fmt.Errorf("sheet %s not found", sheetIPs)
	}
	if len(rows) <= 1 {
		return nil, nil
	}
	addressIdx := findColumnWithDetector(rows, []string{"ip", "ipaddress", "address", "地址"}, func(values []string) bool {
		for _, v := range values {
			candidate := strings.TrimSpace(v)
			if candidate == "" {
				continue
			}
			if ipAddressPattern.MatchString(candidate) {
				return true
			}
		}
		return false
	})
	descriptionIdx := findColumn(rows[0], []string{"description", "desc", "备注", "note", "说明"})
	tagsIdx := findColumn(rows[0], []string{"tags", "tag", "标签"})

	entries := make([]IPEntry, 0, len(rows)-1)
	for _, row := range rows[1:] {
		entry := IPEntry{}
		entry.Address = strings.TrimSpace(valueFor(row, addressIdx, 0))
		entry.Description = strings.TrimSpace(valueFor(row, descriptionIdx, 1))
		if tags := valueFor(row, tagsIdx, 2); tags != "" {
			entry.Tags = splitTags(tags)
		}
		if entry.Address == "" {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func extractDevices(wb *xlsx.Workbook) ([]DeviceEntry, error) {
	rows, ok := wb.Rows(sheetDevices)
	if !ok {
		return nil, fmt.Errorf("sheet %s not found", sheetDevices)
	}
	if len(rows) <= 1 {
		return nil, nil
	}
	identifierIdx := findColumn(rows[0], []string{"identifier", "device", "设备", "资产", "serial", "sn"})
	typeIdx := findColumn(rows[0], []string{"type", "类别", "型号", "category"})
	ownerIdx := findColumn(rows[0], []string{"owner", "负责人", "责任人", "user"})
	tagsIdx := findColumn(rows[0], []string{"tags", "tag", "标签"})

	entries := make([]DeviceEntry, 0, len(rows)-1)
	for _, row := range rows[1:] {
		entry := DeviceEntry{}
		entry.Identifier = strings.TrimSpace(valueFor(row, identifierIdx, 0))
		entry.Type = strings.TrimSpace(valueFor(row, typeIdx, 1))
		entry.Owner = strings.TrimSpace(valueFor(row, ownerIdx, 2))
		if tags := valueFor(row, tagsIdx, 3); tags != "" {
			entry.Tags = splitTags(tags)
		}
		if entry.Identifier == "" {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func extractPersonnel(wb *xlsx.Workbook) ([]PersonnelEntry, error) {
	rows, ok := wb.Rows(sheetPersonnel)
	if !ok {
		return nil, fmt.Errorf("sheet %s not found", sheetPersonnel)
	}
	if len(rows) <= 1 {
		return nil, nil
	}
	nameIdx := findColumn(rows[0], []string{"name", "personnel", "人员", "姓名"})
	roleIdx := findColumn(rows[0], []string{"role", "职位", "职务", "岗位", "title"})
	contactIdx := findColumn(rows[0], []string{"contact", "联系方式", "电话", "邮箱", "email", "phone"})
	tagsIdx := findColumn(rows[0], []string{"tags", "tag", "标签"})

	entries := make([]PersonnelEntry, 0, len(rows)-1)
	for _, row := range rows[1:] {
		entry := PersonnelEntry{}
		entry.Name = strings.TrimSpace(valueFor(row, nameIdx, 0))
		entry.Role = strings.TrimSpace(valueFor(row, roleIdx, 1))
		entry.Contact = strings.TrimSpace(valueFor(row, contactIdx, 2))
		if tags := valueFor(row, tagsIdx, 3); tags != "" {
			entry.Tags = splitTags(tags)
		}
		if entry.Name == "" {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func extractSystems(wb *xlsx.Workbook) ([]SystemEntry, error) {
	rows, ok := wb.Rows(sheetSystems)
	if !ok {
		return nil, fmt.Errorf("sheet %s not found", sheetSystems)
	}
	if len(rows) <= 1 {
		return nil, nil
	}
	nameIdx := findColumn(rows[0], []string{"name", "system", "系统", "应用"})
	envIdx := findColumn(rows[0], []string{"environment", "env", "环境"})
	ownerIdx := findColumn(rows[0], []string{"owner", "负责人", "责任人", "systemowner"})
	tagsIdx := findColumn(rows[0], []string{"tags", "tag", "标签"})

	entries := make([]SystemEntry, 0, len(rows)-1)
	for _, row := range rows[1:] {
		entry := SystemEntry{}
		entry.Name = strings.TrimSpace(valueFor(row, nameIdx, 0))
		entry.Environment = strings.TrimSpace(valueFor(row, envIdx, 1))
		entry.Owner = strings.TrimSpace(valueFor(row, ownerIdx, 2))
		if tags := valueFor(row, tagsIdx, 3); tags != "" {
			entry.Tags = splitTags(tags)
		}
		if entry.Name == "" {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func buildIPRows(entries []IPEntry) [][]string {
	rows := [][]string{{"Address", "Description", "Tags"}}
	for _, entry := range entries {
		rows = append(rows, []string{entry.Address, entry.Description, strings.Join(entry.Tags, ", ")})
	}
	return rows
}

func buildDeviceRows(entries []DeviceEntry) [][]string {
	rows := [][]string{{"Identifier", "Type", "Owner", "Tags"}}
	for _, entry := range entries {
		rows = append(rows, []string{entry.Identifier, entry.Type, entry.Owner, strings.Join(entry.Tags, ", ")})
	}
	return rows
}

func buildPersonnelRows(entries []PersonnelEntry) [][]string {
	rows := [][]string{{"Name", "Role", "Contact", "Tags"}}
	for _, entry := range entries {
		rows = append(rows, []string{entry.Name, entry.Role, entry.Contact, strings.Join(entry.Tags, ", ")})
	}
	return rows
}

func buildSystemRows(entries []SystemEntry) [][]string {
	rows := [][]string{{"Name", "Environment", "Owner", "Tags"}}
	for _, entry := range entries {
		rows = append(rows, []string{entry.Name, entry.Environment, entry.Owner, strings.Join(entry.Tags, ", ")})
	}
	return rows
}

func buildCombinedRows(personnel []PersonnelEntry, devices []DeviceEntry, systems []SystemEntry, ips []IPEntry) [][]string {
	header := []string{"Personnel", "Role", "Contact", "PersonnelTags", "Device", "DeviceType", "DeviceOwner", "DeviceTags", "System", "Environment", "SystemOwner", "SystemTags", "IP", "IPDescription", "IPTags"}
	rows := [][]string{header}
	for _, p := range personnel {
		for _, d := range devices {
			for _, sys := range systems {
				for _, ip := range ips {
					rows = append(rows, []string{
						p.Name,
						p.Role,
						p.Contact,
						strings.Join(p.Tags, ", "),
						d.Identifier,
						d.Type,
						d.Owner,
						strings.Join(d.Tags, ", "),
						sys.Name,
						sys.Environment,
						sys.Owner,
						strings.Join(sys.Tags, ", "),
						ip.Address,
						ip.Description,
						strings.Join(ip.Tags, ", "),
					})
				}
			}
		}
	}
	return rows
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneIPEntries(in []IPEntry) []IPEntry {
	out := make([]IPEntry, len(in))
	for i, entry := range in {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func cloneDeviceEntries(in []DeviceEntry) []DeviceEntry {
	out := make([]DeviceEntry, len(in))
	for i, entry := range in {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func clonePersonnelEntries(in []PersonnelEntry) []PersonnelEntry {
	out := make([]PersonnelEntry, len(in))
	for i, entry := range in {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func cloneSystemEntries(in []SystemEntry) []SystemEntry {
	out := make([]SystemEntry, len(in))
	for i, entry := range in {
		out[i] = entry
		out[i].Tags = cloneStrings(entry.Tags)
	}
	return out
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		cleaned := strings.TrimSpace(tag)
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}
	sort.Strings(normalized)
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	return normalizeTags(parts)
}

func findIPIndex(entries []IPEntry, id int64) int {
	for i, entry := range entries {
		if entry.ID == id {
			return i
		}
	}
	return -1
}

func findDeviceIndex(entries []DeviceEntry, id int64) int {
	for i, entry := range entries {
		if entry.ID == id {
			return i
		}
	}
	return -1
}

func findPersonnelIndex(entries []PersonnelEntry, id int64) int {
	for i, entry := range entries {
		if entry.ID == id {
			return i
		}
	}
	return -1
}

func findSystemIndex(entries []SystemEntry, id int64) int {
	for i, entry := range entries {
		if entry.ID == id {
			return i
		}
	}
	return -1
}

func assignIPIDs(entries []IPEntry, next *int64) []IPEntry {
	out := make([]IPEntry, len(entries))
	for i, entry := range entries {
		id := *next
		*next = *next + 1
		entry.ID = id
		entry.Tags = normalizeTags(entry.Tags)
		out[i] = entry
	}
	return out
}

func assignDeviceIDs(entries []DeviceEntry, next *int64) []DeviceEntry {
	out := make([]DeviceEntry, len(entries))
	for i, entry := range entries {
		id := *next
		*next = *next + 1
		entry.ID = id
		entry.Tags = normalizeTags(entry.Tags)
		out[i] = entry
	}
	return out
}

func assignPersonnelIDs(entries []PersonnelEntry, next *int64) []PersonnelEntry {
	out := make([]PersonnelEntry, len(entries))
	for i, entry := range entries {
		id := *next
		*next = *next + 1
		entry.ID = id
		entry.Tags = normalizeTags(entry.Tags)
		out[i] = entry
	}
	return out
}

func assignSystemIDs(entries []SystemEntry, next *int64) []SystemEntry {
	out := make([]SystemEntry, len(entries))
	for i, entry := range entries {
		id := *next
		*next = *next + 1
		entry.ID = id
		entry.Tags = normalizeTags(entry.Tags)
		out[i] = entry
	}
	return out
}

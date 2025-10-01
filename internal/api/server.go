package api

import (
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ledger/internal/auth"
	"ledger/internal/db"
	"ledger/internal/middleware"
	"ledger/internal/models"
	"ledger/internal/xlsx"
)

// Server wires handlers to the in-memory store and session manager.
type Server struct {
	Database *db.Database
	Store    *models.LedgerStore
	Sessions *auth.Manager
}

// RegisterRoutes attaches handlers to the gin engine.
func (s *Server) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", s.handleHealth)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/enroll-request", s.handleEnrollRequest)
		authGroup.POST("/enroll-complete", s.handleEnrollComplete)
		authGroup.POST("/request-nonce", s.handleRequestNonce)
		authGroup.POST("/login-password", s.handleLoginPassword)
		authGroup.POST("/login", s.handleLogin)
	}

	secured := router.Group("/api/v1")
	secured.Use(middleware.RequireSession(s.Sessions))
	{
		secured.GET("/ledgers/:type", s.handleListLedger)
		secured.POST("/ledgers/:type", s.handleCreateLedger)
		secured.PUT("/ledgers/:type/:id", s.handleUpdateLedger)
		secured.DELETE("/ledgers/:type/:id", s.handleDeleteLedger)
		secured.POST("/ledgers/:type/reorder", s.handleReorderLedger)
		secured.POST("/ledgers/:type/import", s.handleImportLedger)
		secured.GET("/ledgers/export", s.handleExportLedger)
		secured.POST("/ledgers/import", s.handleImportWorkbook)
		secured.GET("/ledger-cartesian", s.handleLedgerMatrix)

		secured.GET("/ip-allowlist", s.handleListAllowlist)
		secured.POST("/ip-allowlist", s.handleCreateAllowlist)
		secured.PUT("/ip-allowlist/:id", s.handleUpdateAllowlist)
		secured.DELETE("/ip-allowlist/:id", s.handleDeleteAllowlist)

		secured.POST("/history/undo", s.handleUndo)
		secured.POST("/history/redo", s.handleRedo)
		secured.GET("/history", s.handleHistoryStatus)

		secured.GET("/audit", s.handleAuditList)
		secured.GET("/audit/verify", s.handleAuditVerify)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	payload := gin.H{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)}
	if s.Database != nil {
		if err := s.Database.PingContext(c.Request.Context()); err != nil {
			payload["database"] = gin.H{"status": "unavailable", "error": err.Error()}
		} else {
			payload["database"] = gin.H{"status": "ok"}
		}
	}
	c.JSON(http.StatusOK, payload)
}

type enrollRequest struct {
	Username   string `json:"username"`
	DeviceName string `json:"device_name"`
	PublicKey  string `json:"public_key"`
}

type enrollResponse struct {
	DeviceID string `json:"device_id"`
	Nonce    string `json:"nonce"`
}

func (s *Server) handleEnrollRequest(c *gin.Context) {
	var req enrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	key, err := base64.RawStdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_public_key"})
		return
	}
	enrollment, err := s.Store.StartEnrollment(req.Username, req.DeviceName, key)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, enrollResponse{DeviceID: enrollment.DeviceID, Nonce: enrollment.Nonce})
}

type enrollCompleteRequest struct {
	Username    string `json:"username"`
	DeviceID    string `json:"device_id"`
	Nonce       string `json:"nonce"`
	Signature   string `json:"signature"`
	Fingerprint string `json:"fingerprint"`
}

func (s *Server) handleEnrollComplete(c *gin.Context) {
	var req enrollCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	sig, err := base64.RawStdEncoding.DecodeString(req.Signature)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_signature"})
		return
	}
	device, err := s.Store.CompleteEnrollment(req.Username, req.DeviceID, req.Nonce, sig, req.Fingerprint)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_id": device.ID, "fingerprint_sum": device.FingerprintSum})
}

type nonceRequest struct {
	Username string `json:"username"`
	DeviceID string `json:"device_id"`
	SDID     string `json:"sdid"`
}

type nonceResponse struct {
	Nonce string `json:"nonce"`
}

func (s *Server) handleRequestNonce(c *gin.Context) {
	var req nonceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.SDID
	}
	if deviceID == "" || req.Username == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	challenge, err := s.Store.RequestLoginNonce(req.Username, deviceID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nonceResponse{Nonce: challenge.Nonce})
}

type loginRequest struct {
	Username    string `json:"username"`
	DeviceID    string `json:"device_id"`
	SDID        string `json:"sdid"`
	Nonce       string `json:"nonce"`
	Signature   string `json:"signature"`
	Fingerprint string `json:"fingerprint"`
}

type loginPasswordRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = req.SDID
	}
	if deviceID == "" || req.Username == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	sig, err := base64.RawStdEncoding.DecodeString(req.Signature)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_signature"})
		return
	}
	ip := clientIP(c.Request)
	if _, err := s.Store.ValidateLogin(req.Username, deviceID, req.Nonce, sig, req.Fingerprint, ip); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	session, err := s.Sessions.Issue(strings.ToLower(req.Username), deviceID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (s *Server) handleLoginPassword(c *gin.Context) {
	var req loginPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	if req.Username == "" || req.Password == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	if _, err := s.Store.AuthenticatePassword(req.Username, req.Password); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	session, err := s.Sessions.Issue(strings.ToLower(req.Username), "password")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	c.JSON(http.StatusOK, session)
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		parts := strings.Split(v, ",")
		return strings.TrimSpace(parts[0])
	}
	host := r.RemoteAddr
	if strings.Contains(host, ":") {
		host, _, _ = strings.Cut(host, ":")
	}
	return host
}

func (s *Server) handleListLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	entries := s.Store.ListEntries(typ)
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

type ledgerRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Attributes  map[string]string   `json:"attributes"`
	Tags        []string            `json:"tags"`
	Links       map[string][]string `json:"links"`
}

func (s *Server) handleCreateLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req ledgerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	entry := models.LedgerEntry{
		Name:        req.Name,
		Description: req.Description,
		Attributes:  req.Attributes,
		Tags:        req.Tags,
		Links:       convertLinks(req.Links),
	}
	session := currentSession(c, s.Sessions)
	created, err := s.Store.CreateEntry(typ, entry, session)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, created)
}

func convertLinks(input map[string][]string) map[models.LedgerType][]string {
	if input == nil {
		return nil
	}
	out := make(map[models.LedgerType][]string, len(input))
	for key, values := range input {
		if typ, ok := parseLedgerType(key); ok {
			out[typ] = append([]string{}, values...)
		}
	}
	return out
}

func (s *Server) handleUpdateLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req ledgerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	session := currentSession(c, s.Sessions)
	updated, err := s.Store.UpdateEntry(typ, c.Param("id"), models.LedgerEntry{
		Name:        req.Name,
		Description: req.Description,
		Attributes:  req.Attributes,
		Tags:        req.Tags,
		Links:       convertLinks(req.Links),
	}, session)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) handleDeleteLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	session := currentSession(c, s.Sessions)
	if err := s.Store.DeleteEntry(typ, c.Param("id"), session); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type reorderRequest struct {
	IDs []string `json:"ids"`
}

func (s *Server) handleReorderLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	session := currentSession(c, s.Sessions)
	entries, err := s.Store.ReorderEntries(typ, req.IDs, session)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

type importRequest struct {
	Data string `json:"data"`
}

func (s *Server) handleImportLedger(c *gin.Context) {
	typ, ok := parseLedgerType(c.Param("type"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "unknown_ledger"})
		return
	}
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	raw, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_base64"})
		return
	}
	workbook, err := xlsx.Decode(raw)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_workbook"})
		return
	}
	sheetName := sheetNameForType(typ)
	sheet, ok := workbook.SheetByName(sheetName)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "sheet_missing"})
		return
	}
	entries := parseLedgerSheet(typ, sheet)
	session := currentSession(c, s.Sessions)
	s.Store.ReplaceEntries(typ, entries, session)
	c.JSON(http.StatusOK, gin.H{"items": s.Store.ListEntries(typ)})
}

var ipRegex = regexp.MustCompile(`(?i)^(?:\d{1,3}\.){3}\d{1,3}$|^(?:[a-f0-9]{0,4}:){2,7}[a-f0-9]{0,4}$`)

func parseLedgerSheet(typ models.LedgerType, sheet xlsx.Sheet) []models.LedgerEntry {
	if len(sheet.Rows) == 0 {
		return nil
	}
	headers := normaliseHeaders(sheet.Rows[0])
	entries := make([]models.LedgerEntry, 0, len(sheet.Rows)-1)
	for _, row := range sheet.Rows[1:] {
		if len(row) == 0 {
			continue
		}
		entry := models.LedgerEntry{Attributes: map[string]string{}, Links: map[models.LedgerType][]string{}}
		for idx, header := range headers {
			if idx >= len(row) {
				continue
			}
			value := strings.TrimSpace(row[idx])
			switch header {
			case "id":
				entry.ID = value
			case "name":
				entry.Name = value
			case "description":
				entry.Description = value
			case "tags":
				entry.Tags = strings.Split(value, ";")
			default:
				if header == "" {
					if typ == models.LedgerTypeIP && ipRegex.MatchString(value) {
						entry.Attributes["address"] = value
						if entry.Name == "" {
							entry.Name = value
						}
					}
					continue
				}
				if strings.HasPrefix(header, "link_") {
					if typ, ok := parseLedgerType(strings.TrimPrefix(header, "link_")); ok {
						entry.Links[typ] = strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == ',' })
					}
					continue
				}
				if entry.Attributes == nil {
					entry.Attributes = make(map[string]string)
				}
				entry.Attributes[header] = value
			}
		}
		if typ == models.LedgerTypeIP && entry.Name == "" {
			if addr := entry.Attributes["address"]; addr != "" {
				entry.Name = addr
			}
		}
		if entry.ID == "" {
			entry.ID = models.GenerateID(string(typ))
		}
		entry.CreatedAt = time.Now().UTC()
		entries = append(entries, entry)
	}
	return entries
}

func normaliseHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, header := range headers {
		h := strings.TrimSpace(strings.ToLower(header))
		out[i] = h
	}
	return out
}

func (s *Server) handleExportLedger(c *gin.Context) {
	workbook := s.buildWorkbook()
	data, err := xlsx.Encode(workbook)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "export_failed"})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=ledger.xlsx")
	_, _ = c.Writer.Write(data)
}

func (s *Server) handleImportWorkbook(c *gin.Context) {
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	raw, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_base64"})
		return
	}
	workbook, err := xlsx.Decode(raw)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_workbook"})
		return
	}
	session := currentSession(c, s.Sessions)
	for _, typ := range models.AllLedgerTypes {
		sheetName := sheetNameForType(typ)
		if sheet, ok := workbook.SheetByName(sheetName); ok {
			entries := parseLedgerSheet(typ, sheet)
			s.Store.ReplaceEntries(typ, entries, session)
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "imported"})
}

func (s *Server) handleLedgerMatrix(c *gin.Context) {
	matrix := s.buildCartesianRows()
	c.JSON(http.StatusOK, gin.H{"rows": matrix})
}

func (s *Server) handleListAllowlist(c *gin.Context) {
	entries := s.Store.ListAllowlist()
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

type allowRequest struct {
	Label       string `json:"label"`
	CIDR        string `json:"cidr"`
	Description string `json:"description"`
}

func (s *Server) handleCreateAllowlist(c *gin.Context) {
	var req allowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	entry, err := s.Store.AppendAllowlist(&models.IPAllowlistEntry{Label: req.Label, CIDR: req.CIDR, Description: req.Description}, currentSession(c, s.Sessions))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (s *Server) handleUpdateAllowlist(c *gin.Context) {
	var req allowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	entry := &models.IPAllowlistEntry{ID: c.Param("id"), Label: req.Label, CIDR: req.CIDR, Description: req.Description}
	updated, err := s.Store.AppendAllowlist(entry, currentSession(c, s.Sessions))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) handleDeleteAllowlist(c *gin.Context) {
	if !s.Store.RemoveAllowlist(c.Param("id"), currentSession(c, s.Sessions)) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleUndo(c *gin.Context) {
	if err := s.Store.Undo(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleRedo(c *gin.Context) {
	if err := s.Store.Redo(); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleHistoryStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"undo": s.Store.CanUndo(), "redo": s.Store.CanRedo()})
}

func (s *Server) handleAuditList(c *gin.Context) {
	audits := s.Store.ListAudits()
	c.JSON(http.StatusOK, gin.H{"items": audits})
}

func (s *Server) handleAuditVerify(c *gin.Context) {
	ok := s.Store.VerifyAuditChain()
	c.JSON(http.StatusOK, gin.H{"verified": ok})
}

func parseLedgerType(value string) (models.LedgerType, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "ips", "ip":
		return models.LedgerTypeIP, true
	case "devices", "device":
		return models.LedgerTypeDevice, true
	case "personnel", "people", "person":
		return models.LedgerTypePersonnel, true
	case "systems", "system":
		return models.LedgerTypeSystem, true
	default:
		return "", false
	}
}

func currentSession(c *gin.Context, manager *auth.Manager) string {
	if c == nil {
		return "system"
	}
	if sessionAny, ok := c.Get(middleware.ContextSessionKey); ok {
		if session, ok := sessionAny.(*auth.Session); ok {
			return session.Username
		}
	}
	return "system"
}

func (s *Server) buildWorkbook() xlsx.Workbook {
	sheets := make([]xlsx.Sheet, 0, len(models.AllLedgerTypes)+1)
	for _, typ := range models.AllLedgerTypes {
		entries := s.Store.ListEntries(typ)
		sheet := xlsx.Sheet{Name: sheetNameForType(typ)}
		header := []string{"ID", "Name", "Description", "Tags"}
		keys := attributeKeys(entries)
		header = append(header, keys...)
		for _, other := range models.AllLedgerTypes {
			if other == typ {
				continue
			}
			header = append(header, "link_"+string(other))
		}
		sheet.Rows = append(sheet.Rows, header)
		for _, entry := range entries {
			row := []string{entry.ID, entry.Name, entry.Description, strings.Join(entry.Tags, ";")}
			for _, key := range keys {
				row = append(row, entry.Attributes[key])
			}
			for _, other := range models.AllLedgerTypes {
				if other == typ {
					continue
				}
				row = append(row, strings.Join(entry.Links[other], ";"))
			}
			sheet.Rows = append(sheet.Rows, row)
		}
		sheets = append(sheets, sheet)
	}
	combined := s.buildCartesianRows()
	matrixSheet := xlsx.Sheet{Name: "Matrix"}
	matrixHeader := []string{"IP", "Device", "Personnel", "System"}
	matrixSheet.Rows = append(matrixSheet.Rows, matrixHeader)
	matrixSheet.Rows = append(matrixSheet.Rows, combined...)
	sheets = append(sheets, matrixSheet)
	workbook := xlsx.Workbook{Sheets: sheets}
	workbook.SortSheets([]string{"IP", "Device", "Personnel", "System", "Matrix"})
	return workbook
}

func attributeKeys(entries []models.LedgerEntry) []string {
	keysMap := make(map[string]struct{})
	for _, entry := range entries {
		for key := range entry.Attributes {
			keysMap[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keysMap))
	for key := range keysMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *Server) buildCartesianRows() [][]string {
	ips := s.Store.ListEntries(models.LedgerTypeIP)
	devices := s.Store.ListEntries(models.LedgerTypeDevice)
	personnel := s.Store.ListEntries(models.LedgerTypePersonnel)
	systems := s.Store.ListEntries(models.LedgerTypeSystem)

	rows := [][]string{}
	for _, device := range devices {
		linkedIPs := uniqueOrAll(device.Links[models.LedgerTypeIP], ips)
		linkedPersonnel := uniqueOrAll(device.Links[models.LedgerTypePersonnel], personnel)
		linkedSystems := uniqueOrAll(device.Links[models.LedgerTypeSystem], systems)
		for _, ipID := range linkedIPs {
			ipName := lookupName(ipID, ips)
			for _, personID := range linkedPersonnel {
				personName := lookupName(personID, personnel)
				for _, systemID := range linkedSystems {
					systemName := lookupName(systemID, systems)
					rows = append(rows, []string{ipName, device.Name, personName, systemName})
				}
			}
		}
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"", "", "", ""})
	}
	return rows
}

func uniqueOrAll(ids []string, entries []models.LedgerEntry) []string {
	if len(ids) == 0 {
		out := make([]string, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entry.ID)
		}
		return out
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		for _, entry := range entries {
			out = append(out, entry.ID)
		}
	}
	return out
}

func lookupName(id string, entries []models.LedgerEntry) string {
	for _, entry := range entries {
		if entry.ID == id {
			return entry.Name
		}
	}
	return id
}

func sheetNameForType(typ models.LedgerType) string {
	switch typ {
	case models.LedgerTypeIP:
		return "IP"
	case models.LedgerTypeDevice:
		return "Device"
	case models.LedgerTypePersonnel:
		return "Personnel"
	case models.LedgerTypeSystem:
		return "System"
	default:
		return strings.Title(string(typ))
	}
}

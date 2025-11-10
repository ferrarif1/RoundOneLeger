package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
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
	Database          *db.Database
	Store             *models.LedgerStore
	Sessions          *auth.Manager
	DataDir           string
	SnapshotRetention int
}

// RegisterRoutes attaches handlers to the gin engine.
func (s *Server) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", s.handleHealth)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/password-login", s.handlePasswordLogin)
		authGroup.POST("/logout", s.handleLogout)
		authGroup.POST("/change-password", s.handleChangePassword)
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

		secured.GET("/workspaces", s.handleListWorkspaces)
		secured.POST("/workspaces", s.handleCreateWorkspace)
		secured.GET("/workspaces/:id", s.handleGetWorkspace)
		secured.PUT("/workspaces/:id", s.handleUpdateWorkspace)
		secured.DELETE("/workspaces/:id", s.handleDeleteWorkspace)
		secured.POST("/workspaces/:id/import/excel", s.handleImportWorkspaceExcel)
		secured.POST("/workspaces/:id/import/text", s.handleImportWorkspaceText)
		secured.GET("/workspaces/:id/export", s.handleExportWorkspace)
		secured.POST("/workspaces/:id/export/selected", s.handleExportWorkspaceSelected)

		secured.GET("/users", s.handleListUsers)
		secured.POST("/users", s.handleCreateUser)
		secured.DELETE("/users/:id", s.handleDeleteUser)

		secured.GET("/ip-allowlist", s.handleListAllowlist)
		secured.POST("/ip-allowlist", s.handleCreateAllowlist)
		secured.PUT("/ip-allowlist/:id", s.handleUpdateAllowlist)
		secured.DELETE("/ip-allowlist/:id", s.handleDeleteAllowlist)

		secured.POST("/history/undo", s.handleUndo)
		secured.POST("/history/redo", s.handleRedo)
		secured.GET("/history", s.handleHistoryStatus)

		secured.GET("/audit", s.handleAuditList)
		secured.GET("/audit/verify", s.handleAuditVerify)
		secured.GET("/export/all", s.handleExportAll)
		secured.POST("/import/all", s.handleImportAll)
		secured.POST("/admin/save-snapshot", s.handleManualSave)
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

// SDID nonce and base64 helpers removed as we now use username/password only

// SDID-related types removed

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// SDID login handler removed

func (s *Server) handlePasswordLogin(c *gin.Context) {
	var req passwordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	user, err := s.Store.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, models.ErrUsernameInvalid) || errors.Is(err, models.ErrPasswordTooShort) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	session, err := s.Sessions.Issue(user.Username, user.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session_issue_failed"})
		return
	}
	s.Store.RecordLogin(user.Username)
	c.JSON(http.StatusOK, gin.H{
		"token":     session.Token,
		"username":  user.Username,
		"admin":     user.Admin,
		"issuedAt":  session.IssuedAt,
		"expiresAt": session.ExpiresAt,
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		s.Sessions.Revoke(token)
	}
	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

func (s *Server) handleChangePassword(c *gin.Context) {
	session := currentSession(c, s.Sessions)
	if strings.TrimSpace(session) == "" || session == "system" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	if err := s.Store.ChangePassword(session, strings.TrimSpace(req.OldPassword), strings.TrimSpace(req.NewPassword), session); err != nil {
		status := http.StatusUnauthorized
		switch {
		case errors.Is(err, models.ErrPasswordTooShort), errors.Is(err, models.ErrPasswordTooWeak), errors.Is(err, models.ErrUsernameInvalid):
			status = http.StatusBadRequest
		case errors.Is(err, models.ErrUserNotFound):
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "password_changed"})
}

// Challenge resolution removed

// Approval submission removed

// Approval endpoints removed

func canonicalizeJSONValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		b, _ := json.Marshal(v)
		return string(b)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = canonicalizeJSONValue(item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, strconv.Quote(key)+":"+canonicalizeJSONValue(v[key]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func canonicalizeJSON(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	return canonicalizeJSONValue(value), nil
}

// JWK decode removed

// SDID claims helpers removed

// SDID verification removed

// SDID identity approval evaluation removed

// Approval helpers removed

// Challenge satisfaction removed

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		parts := strings.Split(v, ",")
		candidate := strings.TrimSpace(parts[0])
		if ip := net.ParseIP(candidate); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if ip := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); ip != nil {
			return ip.String()
		}
		return r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
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

type workspaceColumnPayload struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Width int    `json:"width,omitempty"`
}

type workspaceRowPayload struct {
	ID          string            `json:"id"`
	Cells       map[string]string `json:"cells"`
	Styles      map[string]string `json:"styles,omitempty"`
	Highlighted bool              `json:"highlighted,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty"`
}

type workspaceResponse struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Kind      string                   `json:"kind"`
	ParentID  string                   `json:"parentId,omitempty"`
	Columns   []workspaceColumnPayload `json:"columns"`
	Rows      []workspaceRowPayload    `json:"rows"`
	Document  string                   `json:"document,omitempty"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
}

type workspaceTreeItem struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Kind      string              `json:"kind"`
	ParentID  string              `json:"parentId,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Children  []workspaceTreeItem `json:"children,omitempty"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Admin     bool      `json:"admin"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type userCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

type workspaceRequest struct {
	Name     string                   `json:"name"`
	Document string                   `json:"document"`
	Columns  []workspaceColumnPayload `json:"columns"`
	Rows     []workspaceRowPayload    `json:"rows"`
	Kind     string                   `json:"kind"`
	ParentID string                   `json:"parentId"`
}

type workspaceUpdateRequest struct {
	Name     *string                   `json:"name,omitempty"`
	Document *string                   `json:"document,omitempty"`
	Columns  *[]workspaceColumnPayload `json:"columns,omitempty"`
	Rows     *[]workspaceRowPayload    `json:"rows,omitempty"`
	ParentID *string                   `json:"parentId,omitempty"`
}

type workspaceTextImportRequest struct {
	Text      string `json:"text"`
	Delimiter string `json:"delimiter"`
	HasHeader *bool  `json:"hasHeader"`
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

func (s *Server) handleListWorkspaces(c *gin.Context) {
	items := s.Store.ListWorkspaces()
	buckets := make(map[string][]workspaceTreeItem)
	for _, item := range items {
		parent := strings.TrimSpace(item.ParentID)
		buckets[parent] = append(buckets[parent], workspaceTreeItemFromModel(item))
	}
	var build func(parent string) []workspaceTreeItem
	build = func(parent string) []workspaceTreeItem {
		nodes := buckets[parent]
		result := make([]workspaceTreeItem, len(nodes))
		for i, node := range nodes {
			node.Children = build(node.ID)
			result[i] = node
		}
		return result
	}
	c.JSON(http.StatusOK, gin.H{"items": build("")})
}

func (s *Server) handleCreateWorkspace(c *gin.Context) {
	var req workspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	actor := currentSession(c, s.Sessions)
	kind := models.ParseWorkspaceKind(req.Kind)
	workspace, err := s.Store.CreateWorkspace(
		req.Name,
		kind,
		req.ParentID,
		payloadColumnsToModel(req.Columns),
		payloadRowsToModel(req.Rows),
		req.Document,
		actor,
	)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceParentInvalid) || errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleGetWorkspace(c *gin.Context) {
	workspace, err := s.Store.GetWorkspace(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleUpdateWorkspace(c *gin.Context) {
	var req workspaceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	update := models.WorkspaceUpdate{}
	if req.Name != nil {
		update.SetName = true
		update.Name = *req.Name
	}
	if req.Document != nil {
		update.SetDocument = true
		update.Document = *req.Document
	}
	if req.Columns != nil {
		update.SetColumns = true
		update.Columns = payloadColumnsToModel(*req.Columns)
	}
	if req.Rows != nil {
		update.SetRows = true
		update.Rows = payloadRowsToModel(*req.Rows)
	}
	if req.ParentID != nil {
		update.SetParent = true
		update.ParentID = strings.TrimSpace(*req.ParentID)
	}
	actor := currentSession(c, s.Sessions)
	workspace, err := s.Store.UpdateWorkspace(c.Param("id"), update, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, models.ErrWorkspaceParentInvalid) || errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleDeleteWorkspace(c *gin.Context) {
	if err := s.Store.DeleteWorkspace(c.Param("id"), currentSession(c, s.Sessions)); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleImportWorkspaceExcel(c *gin.Context) {
	uploaded, _, err := c.Request.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing_file"})
		return
	}
	defer uploaded.Close()

	data, err := io.ReadAll(uploaded)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "file_read_failed"})
		return
	}

	workbook, err := xlsx.Decode(data)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_workbook"})
		return
	}
	if len(workbook.Sheets) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "workbook_empty"})
		return
	}
	sheet := workbook.Sheets[0]
	headers := []string{}
	rows := [][]string{}
	if len(sheet.Rows) > 0 {
		headers = append(headers, sheet.Rows[0]...)
		for _, row := range sheet.Rows[1:] {
			copied := append([]string{}, row...)
			rows = append(rows, copied)
		}
	}
	actor := currentSession(c, s.Sessions)
	workspace, err := s.Store.ReplaceWorkspaceData(c.Param("id"), headers, rows, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleImportWorkspaceText(c *gin.Context) {
	var req workspaceTextImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "empty_text"})
		return
	}
	delimiter := normaliseDelimiter(req.Delimiter)
	hasHeader := true
	if req.HasHeader != nil {
		hasHeader = *req.HasHeader
	}
	headers, records := parseDelimitedText(text, delimiter, hasHeader)
	actor := currentSession(c, s.Sessions)
	workspace, err := s.Store.ReplaceWorkspaceData(c.Param("id"), headers, records, actor)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, models.ErrWorkspaceKindUnsupported) {
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workspace": workspaceToResponse(workspace)})
}

func (s *Server) handleExportWorkspace(c *gin.Context) {
	workspace, err := s.Store.GetWorkspace(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}

	if !models.WorkspaceKindSupportsTable(workspace.Kind) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": models.ErrWorkspaceKindUnsupported.Error()})
		return
	}

	rows := make([][]string, 0, len(workspace.Rows)+1)
	if len(workspace.Columns) > 0 {
		header := make([]string, len(workspace.Columns))
		for i, column := range workspace.Columns {
			header[i] = column.Title
		}
		rows = append(rows, header)
	}
	for _, row := range workspace.Rows {
		record := make([]string, len(workspace.Columns))
		for i, column := range workspace.Columns {
			record[i] = row.Cells[column.ID]
		}
		rows = append(rows, record)
	}
	sheetName := workspace.Name
	if strings.TrimSpace(sheetName) == "" {
		sheetName = "workspace"
	}
	workbook := xlsx.Workbook{Sheets: []xlsx.Sheet{{Name: sheetName, Rows: rows}}}
	encoded, err := xlsx.Encode(workbook)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "encode_failed"})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.xlsx\"", "workspace"))
	c.Writer.Write(encoded)
}

type exportSelectedRequest struct {
	RowIDs []string `json:"rowIds"`
}

func (s *Server) handleExportWorkspaceSelected(c *gin.Context) {
	workspace, err := s.Store.GetWorkspace(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, models.ErrWorkspaceNotFound) {
			status = http.StatusNotFound
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	if !models.WorkspaceKindSupportsTable(workspace.Kind) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": models.ErrWorkspaceKindUnsupported.Error()})
		return
	}
	var req exportSelectedRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.RowIDs) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	allow := make(map[string]struct{}, len(req.RowIDs))
	for _, id := range req.RowIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			allow[id] = struct{}{}
		}
	}
	rows := make([][]string, 0, len(workspace.Rows)+1)
	if len(workspace.Columns) > 0 {
		header := make([]string, len(workspace.Columns))
		for i, column := range workspace.Columns {
			header[i] = column.Title
		}
		rows = append(rows, header)
	}
	for _, row := range workspace.Rows {
		if _, ok := allow[row.ID]; !ok {
			continue
		}
		record := make([]string, len(workspace.Columns))
		for i, column := range workspace.Columns {
			record[i] = row.Cells[column.ID]
		}
		rows = append(rows, record)
	}
	sheetName := workspace.Name
	if strings.TrimSpace(sheetName) == "" {
		sheetName = "workspace"
	}
	workbook := xlsx.Workbook{Sheets: []xlsx.Sheet{{Name: sheetName, Rows: rows}}}
	encoded, err := xlsx.Encode(workbook)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "encode_failed"})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_selected.xlsx\"", "workspace"))
	c.Writer.Write(encoded)
}

func (s *Server) handleListUsers(c *gin.Context) {
	session := currentSession(c, s.Sessions)
	if !s.Store.IsUserAdmin(session) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}
	users := s.Store.ListUsers()
	items := make([]userResponse, 0, len(users))
	for _, user := range users {
		items = append(items, userToResponse(user))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) handleCreateUser(c *gin.Context) {
	session := currentSession(c, s.Sessions)
	if !s.Store.IsUserAdmin(session) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}
	var req userCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	user, err := s.Store.CreateUser(req.Username, req.Password, req.Admin, session)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, models.ErrUsernameInvalid), errors.Is(err, models.ErrPasswordTooShort):
			status = http.StatusBadRequest
		case errors.Is(err, models.ErrUserExists):
			status = http.StatusConflict
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": userToResponse(user)})
}

func (s *Server) handleDeleteUser(c *gin.Context) {
	session := currentSession(c, s.Sessions)
	if !s.Store.IsUserAdmin(session) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}
	if err := s.Store.DeleteUser(c.Param("id"), session); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, models.ErrUserNotFound):
			status = http.StatusNotFound
		case errors.Is(err, models.ErrUserDeleteLastAdmin):
			status = http.StatusBadRequest
		}
		c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func workspaceToResponse(workspace *models.Workspace) workspaceResponse {
	if workspace == nil {
		return workspaceResponse{}
	}
	columns := make([]workspaceColumnPayload, len(workspace.Columns))
	for i, column := range workspace.Columns {
		columns[i] = workspaceColumnPayload{ID: column.ID, Title: column.Title, Width: column.Width}
	}
	rows := make([]workspaceRowPayload, len(workspace.Rows))
	for i, row := range workspace.Rows {
		cells := make(map[string]string, len(row.Cells))
		for key, value := range row.Cells {
			cells[key] = value
		}
		styles := map[string]string{}
		if row.Styles != nil {
			for k, v := range row.Styles {
				styles[k] = v
			}
		}
		rows[i] = workspaceRowPayload{ID: row.ID, Cells: cells, Styles: styles, Highlighted: row.Highlighted, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	}
	return workspaceResponse{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Kind:      string(models.NormalizeWorkspaceKind(workspace.Kind)),
		ParentID:  strings.TrimSpace(workspace.ParentID),
		Columns:   columns,
		Rows:      rows,
		Document:  workspace.Document,
		CreatedAt: workspace.CreatedAt,
		UpdatedAt: workspace.UpdatedAt,
	}
}

func workspaceTreeItemFromModel(workspace *models.Workspace) workspaceTreeItem {
	if workspace == nil {
		return workspaceTreeItem{}
	}
	return workspaceTreeItem{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Kind:      string(models.NormalizeWorkspaceKind(workspace.Kind)),
		ParentID:  strings.TrimSpace(workspace.ParentID),
		CreatedAt: workspace.CreatedAt,
		UpdatedAt: workspace.UpdatedAt,
	}
}

func userToResponse(user *models.User) userResponse {
	if user == nil {
		return userResponse{}
	}
	return userResponse{
		ID:        user.ID,
		Username:  user.Username,
		Admin:     user.Admin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func payloadColumnsToModel(columns []workspaceColumnPayload) []models.WorkspaceColumn {
	if len(columns) == 0 {
		return nil
	}
	out := make([]models.WorkspaceColumn, 0, len(columns))
	for _, column := range columns {
		out = append(out, models.WorkspaceColumn{ID: strings.TrimSpace(column.ID), Title: column.Title, Width: column.Width})
	}
	return out
}

func payloadRowsToModel(rows []workspaceRowPayload) []models.WorkspaceRow {
	if len(rows) == 0 {
		return nil
	}
	out := make([]models.WorkspaceRow, 0, len(rows))
	for _, row := range rows {
		cells := make(map[string]string, len(row.Cells))
		for key, value := range row.Cells {
			cells[key] = value
		}
		styles := map[string]string{}
		if row.Styles != nil {
			for k, v := range row.Styles {
				styles[k] = v
			}
		}
		out = append(out, models.WorkspaceRow{ID: strings.TrimSpace(row.ID), Cells: cells, Styles: styles, Highlighted: row.Highlighted, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out
}

func normaliseDelimiter(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "", "tab", "\\t":
		return "\t"
	case "comma", ",":
		return ","
	case "semicolon", ";":
		return ";"
	case "space":
		return " "
	default:
		return trimmed
	}
}

func parseDelimitedText(text string, delimiter string, hasHeader bool) ([]string, [][]string) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	if len(cleaned) == 0 {
		return []string{}, [][]string{}
	}
	header := []string{}
	data := [][]string{}
	for idx, line := range cleaned {
		fields := splitWithDelimiter(line, delimiter)
		if idx == 0 && hasHeader {
			header = append(header, fields...)
			continue
		}
		data = append(data, fields)
	}
	return header, data
}

func splitWithDelimiter(line, delimiter string) []string {
	if delimiter == "" {
		return strings.Fields(line)
	}
	parts := strings.Split(line, delimiter)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
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

func (s *Server) handleExportAll(c *gin.Context) {
	snapshot := s.Store.ExportSnapshot()
	c.JSON(http.StatusOK, snapshot)
}

func (s *Server) handleImportAll(c *gin.Context) {
	var snapshot models.Snapshot
	if err := c.ShouldBindJSON(&snapshot); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_payload"})
		return
	}
	if err := s.Store.ImportSnapshot(&snapshot); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "imported"})
}

func (s *Server) handleManualSave(c *gin.Context) {
	if strings.TrimSpace(s.DataDir) == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "data_dir_not_configured"})
		return
	}
	if err := s.Store.SaveToWithRetention(s.DataDir, s.SnapshotRetention); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

func parseLedgerType(value string) (models.LedgerType, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "ips", "ip":
		return models.LedgerTypeIP, true
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
	matrixHeader := []string{"IP", "Personnel", "System"}
	matrixSheet.Rows = append(matrixSheet.Rows, matrixHeader)
	matrixSheet.Rows = append(matrixSheet.Rows, combined...)
	sheets = append(sheets, matrixSheet)
	workbook := xlsx.Workbook{Sheets: sheets}
	workbook.SortSheets([]string{"IP", "Personnel", "System", "Matrix"})
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
	personnel := s.Store.ListEntries(models.LedgerTypePersonnel)
	systems := s.Store.ListEntries(models.LedgerTypeSystem)

	rows := [][]string{}
	for _, system := range systems {
		linkedIPs := uniqueOrAll(system.Links[models.LedgerTypeIP], ips)
		linkedPersonnel := uniqueOrAll(system.Links[models.LedgerTypePersonnel], personnel)
		for _, ipID := range linkedIPs {
			ipName := lookupName(ipID, ips)
			for _, personID := range linkedPersonnel {
				personName := lookupName(personID, personnel)
				rows = append(rows, []string{ipName, personName, system.Name})
			}
		}
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"", "", ""})
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
	case models.LedgerTypePersonnel:
		return "Personnel"
	case models.LedgerTypeSystem:
		return "System"
	default:
		return strings.Title(string(typ))
	}
}

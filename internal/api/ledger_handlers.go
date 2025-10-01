package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"ledger/internal/models"
	"ledger/internal/xlsx"
)

const legacyActor = "legacy-api"

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

type LedgerState struct {
	IPs       []IPEntry        `json:"ips"`
	Devices   []DeviceEntry    `json:"devices"`
	Personnel []PersonnelEntry `json:"personnel"`
	Systems   []SystemEntry    `json:"systems"`
}

type HistoryStatus struct {
	UndoSteps int `json:"undoSteps"`
	RedoSteps int `json:"redoSteps"`
}

type LedgerHandler struct {
	Store *models.LedgerStore

	mu      sync.RWMutex
	legacy  map[models.LedgerType]map[int64]string
	reverse map[models.LedgerType]map[string]int64
	next    map[models.LedgerType]int64
}

func (h *LedgerHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/ledger/ips", h.getIPs)
	router.POST("/ledger/ips", h.createIP)
	router.PUT("/ledger/ips/:id", h.updateIP)
	router.DELETE("/ledger/ips/:id", h.deleteIP)
	router.PUT("/ledger/ips/order", h.reorderIPs)

	router.GET("/ledger/state", h.getState)
	router.GET("/ledger/history", h.getHistoryStatus)
	router.POST("/ledger/history/undo", h.undoLedger)
	router.POST("/ledger/history/redo", h.redoLedger)

	router.GET("/ledger/devices", h.getDevices)
	router.POST("/ledger/devices", h.createDevice)
	router.PUT("/ledger/devices/:id", h.updateDevice)
	router.DELETE("/ledger/devices/:id", h.deleteDevice)
	router.PUT("/ledger/devices/order", h.reorderDevices)

	router.GET("/ledger/personnel", h.getPersonnel)
	router.POST("/ledger/personnel", h.createPersonnel)
	router.PUT("/ledger/personnel/:id", h.updatePersonnel)
	router.DELETE("/ledger/personnel/:id", h.deletePersonnel)
	router.PUT("/ledger/personnel/order", h.reorderPersonnel)

	router.GET("/ledger/systems", h.getSystems)
	router.POST("/ledger/systems", h.createSystem)
	router.PUT("/ledger/systems/:id", h.updateSystem)
	router.DELETE("/ledger/systems/:id", h.deleteSystem)
	router.PUT("/ledger/systems/order", h.reorderSystems)

	router.GET("/ledger/combined", h.getCombined)
	router.POST("/ledger/import", h.importExcel)
	router.GET("/ledger/export", h.exportExcel)
}

func (h *LedgerHandler) getIPs(c *gin.Context) {
	h.syncLegacyIDs()
	entries := h.Store.ListEntries(models.LedgerTypeIP)
	c.JSON(http.StatusOK, gin.H{"ips": h.convertIPEntries(entries)})
}

func (h *LedgerHandler) getDevices(c *gin.Context) {
	h.syncLegacyIDs()
	entries := h.Store.ListEntries(models.LedgerTypeDevice)
	c.JSON(http.StatusOK, gin.H{"devices": h.convertDeviceEntries(entries)})
}

func (h *LedgerHandler) getPersonnel(c *gin.Context) {
	h.syncLegacyIDs()
	entries := h.Store.ListEntries(models.LedgerTypePersonnel)
	c.JSON(http.StatusOK, gin.H{"personnel": h.convertPersonnelEntries(entries)})
}

func (h *LedgerHandler) getSystems(c *gin.Context) {
	h.syncLegacyIDs()
	entries := h.Store.ListEntries(models.LedgerTypeSystem)
	c.JSON(http.StatusOK, gin.H{"systems": h.convertSystemEntries(entries)})
}

func (h *LedgerHandler) getCombined(c *gin.Context) {
	h.syncLegacyIDs()
	ips := h.convertIPEntries(h.Store.ListEntries(models.LedgerTypeIP))
	devices := h.convertDeviceEntries(h.Store.ListEntries(models.LedgerTypeDevice))
	personnel := h.convertPersonnelEntries(h.Store.ListEntries(models.LedgerTypePersonnel))
	systems := h.convertSystemEntries(h.Store.ListEntries(models.LedgerTypeSystem))
	c.JSON(http.StatusOK, gin.H{"combined": buildCombinedEntries(personnel, devices, systems, ips)})
}

func (h *LedgerHandler) getState(c *gin.Context) {
	h.syncLegacyIDs()
	state := LedgerState{
		IPs:       h.convertIPEntries(h.Store.ListEntries(models.LedgerTypeIP)),
		Devices:   h.convertDeviceEntries(h.Store.ListEntries(models.LedgerTypeDevice)),
		Personnel: h.convertPersonnelEntries(h.Store.ListEntries(models.LedgerTypePersonnel)),
		Systems:   h.convertSystemEntries(h.Store.ListEntries(models.LedgerTypeSystem)),
	}
	c.JSON(http.StatusOK, gin.H{"state": state, "history": h.currentHistoryStatus()})
}

func (h *LedgerHandler) getHistoryStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"history": h.currentHistoryStatus()})
}

type ledgerEntryRequest struct {
	Address     string   `json:"address"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Identifier  string   `json:"identifier"`
	Type        string   `json:"type"`
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Contact     string   `json:"contact"`
	Environment string   `json:"environment"`
}

type ledgerOrderRequest struct {
	Order []int64 `json:"order"`
}

func (h *LedgerHandler) createIP(c *gin.Context) {
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	address := strings.TrimSpace(req.Address)
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address_required"})
		return
	}
	entry := models.LedgerEntry{
		Name:        address,
		Description: strings.TrimSpace(req.Description),
		Attributes: map[string]string{
			"address": address,
		},
		Tags: req.Tags,
	}
	created, err := h.Store.CreateEntry(models.LedgerTypeIP, entry, legacyActor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusCreated, h.convertIPEntry(created))
}

func (h *LedgerHandler) updateIP(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	address := strings.TrimSpace(req.Address)
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address_required"})
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypeIP, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "ip_not_found"})
		return
	}
	update := models.LedgerEntry{
		Name:        address,
		Description: strings.TrimSpace(req.Description),
		Attributes: map[string]string{
			"address": address,
		},
		Tags: req.Tags,
	}
	updated, err := h.Store.UpdateEntry(models.LedgerTypeIP, storeID, update, legacyActor)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, h.convertIPEntry(updated))
}

func (h *LedgerHandler) deleteIP(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypeIP, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "ip_not_found"})
		return
	}
	if err := h.Store.DeleteEntry(models.LedgerTypeIP, storeID, legacyActor); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderIPs(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	h.syncLegacyIDs()
	ordered, err := h.translateOrder(models.LedgerTypeIP, req.Order)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.Store.ReorderEntries(models.LedgerTypeIP, ordered, legacyActor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) createDevice(c *gin.Context) {
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identifier_required"})
		return
	}
	entry := models.LedgerEntry{
		Name: identifier,
		Attributes: map[string]string{
			"identifier": identifier,
			"type":       strings.TrimSpace(req.Type),
			"owner":      strings.TrimSpace(req.Owner),
		},
		Tags: req.Tags,
	}
	created, err := h.Store.CreateEntry(models.LedgerTypeDevice, entry, legacyActor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusCreated, h.convertDeviceEntry(created))
}

func (h *LedgerHandler) updateDevice(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identifier_required"})
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypeDevice, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "device_not_found"})
		return
	}
	update := models.LedgerEntry{
		Name: identifier,
		Attributes: map[string]string{
			"identifier": identifier,
			"type":       strings.TrimSpace(req.Type),
			"owner":      strings.TrimSpace(req.Owner),
		},
		Tags: req.Tags,
	}
	updated, err := h.Store.UpdateEntry(models.LedgerTypeDevice, storeID, update, legacyActor)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, h.convertDeviceEntry(updated))
}

func (h *LedgerHandler) deleteDevice(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypeDevice, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "device_not_found"})
		return
	}
	if err := h.Store.DeleteEntry(models.LedgerTypeDevice, storeID, legacyActor); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderDevices(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	h.syncLegacyIDs()
	ordered, err := h.translateOrder(models.LedgerTypeDevice, req.Order)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.Store.ReorderEntries(models.LedgerTypeDevice, ordered, legacyActor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) createPersonnel(c *gin.Context) {
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name_required"})
		return
	}
	entry := models.LedgerEntry{
		Name: name,
		Attributes: map[string]string{
			"name":    name,
			"role":    strings.TrimSpace(req.Role),
			"contact": strings.TrimSpace(req.Contact),
		},
		Tags: req.Tags,
	}
	created, err := h.Store.CreateEntry(models.LedgerTypePersonnel, entry, legacyActor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusCreated, h.convertPersonnelEntry(created))
}

func (h *LedgerHandler) updatePersonnel(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name_required"})
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypePersonnel, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "personnel_not_found"})
		return
	}
	update := models.LedgerEntry{
		Name: name,
		Attributes: map[string]string{
			"name":    name,
			"role":    strings.TrimSpace(req.Role),
			"contact": strings.TrimSpace(req.Contact),
		},
		Tags: req.Tags,
	}
	updated, err := h.Store.UpdateEntry(models.LedgerTypePersonnel, storeID, update, legacyActor)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, h.convertPersonnelEntry(updated))
}

func (h *LedgerHandler) deletePersonnel(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypePersonnel, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "personnel_not_found"})
		return
	}
	if err := h.Store.DeleteEntry(models.LedgerTypePersonnel, storeID, legacyActor); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderPersonnel(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	h.syncLegacyIDs()
	ordered, err := h.translateOrder(models.LedgerTypePersonnel, req.Order)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.Store.ReorderEntries(models.LedgerTypePersonnel, ordered, legacyActor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) createSystem(c *gin.Context) {
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name_required"})
		return
	}
	entry := models.LedgerEntry{
		Name: name,
		Attributes: map[string]string{
			"name":        name,
			"environment": strings.TrimSpace(req.Environment),
			"owner":       strings.TrimSpace(req.Owner),
		},
		Tags: req.Tags,
	}
	created, err := h.Store.CreateEntry(models.LedgerTypeSystem, entry, legacyActor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusCreated, h.convertSystemEntry(created))
}

func (h *LedgerHandler) updateSystem(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name_required"})
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypeSystem, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "system_not_found"})
		return
	}
	update := models.LedgerEntry{
		Name: name,
		Attributes: map[string]string{
			"name":        name,
			"environment": strings.TrimSpace(req.Environment),
			"owner":       strings.TrimSpace(req.Owner),
		},
		Tags: req.Tags,
	}
	updated, err := h.Store.UpdateEntry(models.LedgerTypeSystem, storeID, update, legacyActor)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, h.convertSystemEntry(updated))
}

func (h *LedgerHandler) deleteSystem(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	h.syncLegacyIDs()
	storeID, found := h.storeIDFromLegacy(models.LedgerTypeSystem, id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "system_not_found"})
		return
	}
	if err := h.Store.DeleteEntry(models.LedgerTypeSystem, storeID, legacyActor); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrEntryNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderSystems(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	h.syncLegacyIDs()
	ordered, err := h.translateOrder(models.LedgerTypeSystem, req.Order)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.Store.ReorderEntries(models.LedgerTypeSystem, ordered, legacyActor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) undoLedger(c *gin.Context) {
	if err := h.Store.Undo(); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "history": h.currentHistoryStatus()})
		return
	}
	h.syncLegacyIDs()
	state := LedgerState{
		IPs:       h.convertIPEntries(h.Store.ListEntries(models.LedgerTypeIP)),
		Devices:   h.convertDeviceEntries(h.Store.ListEntries(models.LedgerTypeDevice)),
		Personnel: h.convertPersonnelEntries(h.Store.ListEntries(models.LedgerTypePersonnel)),
		Systems:   h.convertSystemEntries(h.Store.ListEntries(models.LedgerTypeSystem)),
	}
	c.JSON(http.StatusOK, gin.H{"state": state, "history": h.currentHistoryStatus()})
}

func (h *LedgerHandler) redoLedger(c *gin.Context) {
	if err := h.Store.Redo(); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "history": h.currentHistoryStatus()})
		return
	}
	h.syncLegacyIDs()
	state := LedgerState{
		IPs:       h.convertIPEntries(h.Store.ListEntries(models.LedgerTypeIP)),
		Devices:   h.convertDeviceEntries(h.Store.ListEntries(models.LedgerTypeDevice)),
		Personnel: h.convertPersonnelEntries(h.Store.ListEntries(models.LedgerTypePersonnel)),
		Systems:   h.convertSystemEntries(h.Store.ListEntries(models.LedgerTypeSystem)),
	}
	c.JSON(http.StatusOK, gin.H{"state": state, "history": h.currentHistoryStatus()})
}

func (h *LedgerHandler) importExcel(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_parse_failed"})
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_read_failed"})
		return
	}
	workbook, err := xlsx.Decode(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_workbook"})
		return
	}
	for _, typ := range models.AllLedgerTypes {
		sheetName := sheetNameForType(typ)
		if sheet, ok := workbook.SheetByName(sheetName); ok {
			entries := parseLedgerSheet(typ, sheet)
			h.Store.ReplaceEntries(typ, entries, legacyActor)
		} else {
			h.Store.ReplaceEntries(typ, nil, legacyActor)
		}
	}
	h.syncLegacyIDs()
	c.JSON(http.StatusOK, gin.H{"status": "imported"})
}

func (h *LedgerHandler) exportExcel(c *gin.Context) {
	workbook := h.buildWorkbook()
	data, err := xlsx.Encode(workbook)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=ledger_export.xlsx")
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	http.ServeContent(c.Writer, c.Request, "ledger_export.xlsx", time.Now(), bytes.NewReader(data))
}

func (h *LedgerHandler) buildWorkbook() xlsx.Workbook {
	sheets := make([]xlsx.Sheet, 0, len(models.AllLedgerTypes)+1)
	for _, typ := range models.AllLedgerTypes {
		entries := h.Store.ListEntries(typ)
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
	matrix := h.buildCartesianRows()
	matrixSheet := xlsx.Sheet{Name: "Matrix"}
	matrixSheet.Rows = append(matrixSheet.Rows, []string{"IP", "Device", "Personnel", "System"})
	matrixSheet.Rows = append(matrixSheet.Rows, matrix...)
	sheets = append(sheets, matrixSheet)
	workbook := xlsx.Workbook{Sheets: sheets}
	workbook.SortSheets([]string{"IP", "Device", "Personnel", "System", "Matrix"})
	return workbook
}

func (h *LedgerHandler) buildCartesianRows() [][]string {
	ips := h.Store.ListEntries(models.LedgerTypeIP)
	devices := h.Store.ListEntries(models.LedgerTypeDevice)
	personnel := h.Store.ListEntries(models.LedgerTypePersonnel)
	systems := h.Store.ListEntries(models.LedgerTypeSystem)

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

func (h *LedgerHandler) translateOrder(typ models.LedgerType, order []int64) ([]string, error) {
	entries := h.Store.ListEntries(typ)
	if len(order) != len(entries) {
		return nil, errors.New("order_length_mismatch")
	}
	ordered := make([]string, len(order))
	for i, legacyID := range order {
		storeID, found := h.storeIDFromLegacy(typ, legacyID)
		if !found {
			return nil, errors.New("unknown_id")
		}
		ordered[i] = storeID
	}
	return ordered, nil
}

func (h *LedgerHandler) currentHistoryStatus() HistoryStatus {
	undo, redo := h.Store.HistoryDepth()
	return HistoryStatus{UndoSteps: undo, RedoSteps: redo}
}

func (h *LedgerHandler) syncLegacyIDs() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.legacy == nil {
		h.legacy = make(map[models.LedgerType]map[int64]string)
	}
	if h.reverse == nil {
		h.reverse = make(map[models.LedgerType]map[string]int64)
	}
	if h.next == nil {
		h.next = make(map[models.LedgerType]int64)
	}
	for _, typ := range models.AllLedgerTypes {
		entries := h.Store.ListEntries(typ)
		prevReverse := h.reverse[typ]
		newLegacy := make(map[int64]string, len(entries))
		newReverse := make(map[string]int64, len(entries))
		next := h.next[typ]
		if next < 1 {
			next = 1
		}
		for _, entry := range entries {
			if prevReverse != nil {
				if legacyID, ok := prevReverse[entry.ID]; ok {
					newLegacy[legacyID] = entry.ID
					newReverse[entry.ID] = legacyID
					if legacyID >= next {
						next = legacyID + 1
					}
					continue
				}
			}
			legacyID := next
			next++
			newLegacy[legacyID] = entry.ID
			newReverse[entry.ID] = legacyID
		}
		h.legacy[typ] = newLegacy
		h.reverse[typ] = newReverse
		h.next[typ] = next
	}
}

func (h *LedgerHandler) storeIDFromLegacy(typ models.LedgerType, legacyID int64) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.legacy == nil {
		return "", false
	}
	ids := h.legacy[typ]
	if ids == nil {
		return "", false
	}
	storeID, ok := ids[legacyID]
	return storeID, ok
}

func (h *LedgerHandler) legacyIDFromStore(typ models.LedgerType, storeID string) int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.reverse == nil {
		return 0
	}
	ids := h.reverse[typ]
	if ids == nil {
		return 0
	}
	return ids[storeID]
}

func (h *LedgerHandler) convertIPEntries(entries []models.LedgerEntry) []IPEntry {
	out := make([]IPEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, h.convertIPEntry(entry))
	}
	return out
}

func (h *LedgerHandler) convertIPEntry(entry models.LedgerEntry) IPEntry {
	id := h.legacyIDFromStore(models.LedgerTypeIP, entry.ID)
	address := entry.Attributes["address"]
	if address == "" {
		address = entry.Name
	}
	return IPEntry{
		ID:          id,
		Address:     address,
		Description: entry.Description,
		Tags:        copyStrings(entry.Tags),
	}
}

func (h *LedgerHandler) convertDeviceEntries(entries []models.LedgerEntry) []DeviceEntry {
	out := make([]DeviceEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, h.convertDeviceEntry(entry))
	}
	return out
}

func (h *LedgerHandler) convertDeviceEntry(entry models.LedgerEntry) DeviceEntry {
	id := h.legacyIDFromStore(models.LedgerTypeDevice, entry.ID)
	attrs := entry.Attributes
	identifier := attrs["identifier"]
	if identifier == "" {
		identifier = entry.Name
	}
	return DeviceEntry{
		ID:         id,
		Identifier: identifier,
		Type:       attrs["type"],
		Owner:      attrs["owner"],
		Tags:       copyStrings(entry.Tags),
	}
}

func (h *LedgerHandler) convertPersonnelEntries(entries []models.LedgerEntry) []PersonnelEntry {
	out := make([]PersonnelEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, h.convertPersonnelEntry(entry))
	}
	return out
}

func (h *LedgerHandler) convertPersonnelEntry(entry models.LedgerEntry) PersonnelEntry {
	id := h.legacyIDFromStore(models.LedgerTypePersonnel, entry.ID)
	attrs := entry.Attributes
	name := attrs["name"]
	if name == "" {
		name = entry.Name
	}
	return PersonnelEntry{
		ID:      id,
		Name:    name,
		Role:    attrs["role"],
		Contact: attrs["contact"],
		Tags:    copyStrings(entry.Tags),
	}
}

func (h *LedgerHandler) convertSystemEntries(entries []models.LedgerEntry) []SystemEntry {
	out := make([]SystemEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, h.convertSystemEntry(entry))
	}
	return out
}

func (h *LedgerHandler) convertSystemEntry(entry models.LedgerEntry) SystemEntry {
	id := h.legacyIDFromStore(models.LedgerTypeSystem, entry.ID)
	attrs := entry.Attributes
	name := attrs["name"]
	if name == "" {
		name = entry.Name
	}
	return SystemEntry{
		ID:          id,
		Name:        name,
		Environment: attrs["environment"],
		Owner:       attrs["owner"],
		Tags:        copyStrings(entry.Tags),
	}
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func buildCombinedEntries(personnel []PersonnelEntry, devices []DeviceEntry, systems []SystemEntry, ips []IPEntry) []CombinedEntry {
	if len(personnel) == 0 || len(devices) == 0 || len(systems) == 0 || len(ips) == 0 {
		return []CombinedEntry{}
	}
	combined := make([]CombinedEntry, 0, len(personnel)*len(devices)*len(systems)*len(ips))
	for _, p := range personnel {
		for _, d := range devices {
			for _, s := range systems {
				for _, ip := range ips {
					combined = append(combined, CombinedEntry{Personnel: p, Device: d, System: s, IP: ip})
				}
			}
		}
	}
	return combined
}

func parseIDParam(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
		return 0, false
	}
	return id, true
}

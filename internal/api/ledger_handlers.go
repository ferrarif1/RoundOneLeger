package api

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ledger/internal/ledger"
)

type LedgerHandler struct {
	Store *ledger.Store
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
	c.JSON(http.StatusOK, gin.H{"ips": h.Store.GetIPs()})
}

func (h *LedgerHandler) getDevices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"devices": h.Store.GetDevices()})
}

func (h *LedgerHandler) getPersonnel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"personnel": h.Store.GetPersonnel()})
}

func (h *LedgerHandler) getSystems(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"systems": h.Store.GetSystems()})
}

func (h *LedgerHandler) getCombined(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"combined": h.Store.Combined()})
}

func (h *LedgerHandler) getState(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"state": h.Store.CurrentState(), "history": h.Store.HistoryStatus()})
}

func (h *LedgerHandler) getHistoryStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"history": h.Store.HistoryStatus()})
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
	entry, err := h.Store.AddIP(ledger.IPInput{Address: req.Address, Description: req.Description, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, entry)
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

	entry, err := h.Store.UpdateIP(id, ledger.IPInput{Address: req.Address, Description: req.Description, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *LedgerHandler) deleteIP(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteIP(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderIPs(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if err := h.Store.ReorderIPs(req.Order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) createDevice(c *gin.Context) {
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	entry, err := h.Store.AddDevice(ledger.DeviceInput{Identifier: req.Identifier, Type: req.Type, Owner: req.Owner, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, entry)
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
	entry, err := h.Store.UpdateDevice(id, ledger.DeviceInput{Identifier: req.Identifier, Type: req.Type, Owner: req.Owner, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *LedgerHandler) deleteDevice(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteDevice(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderDevices(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if err := h.Store.ReorderDevices(req.Order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) createPersonnel(c *gin.Context) {
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	entry, err := h.Store.AddPersonnel(ledger.PersonnelInput{Name: req.Name, Role: req.Role, Contact: req.Contact, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, entry)
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
	entry, err := h.Store.UpdatePersonnel(id, ledger.PersonnelInput{Name: req.Name, Role: req.Role, Contact: req.Contact, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *LedgerHandler) deletePersonnel(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.Store.DeletePersonnel(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderPersonnel(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if err := h.Store.ReorderPersonnel(req.Order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) createSystem(c *gin.Context) {
	var req ledgerEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	entry, err := h.Store.AddSystem(ledger.SystemInput{Name: req.Name, Environment: req.Environment, Owner: req.Owner, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, entry)
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
	entry, err := h.Store.UpdateSystem(id, ledger.SystemInput{Name: req.Name, Environment: req.Environment, Owner: req.Owner, Tags: req.Tags})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *LedgerHandler) deleteSystem(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteSystem(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *LedgerHandler) reorderSystems(c *gin.Context) {
	var req ledgerOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if err := h.Store.ReorderSystems(req.Order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reordered"})
}

func (h *LedgerHandler) undoLedger(c *gin.Context) {
	state, history, err := h.Store.Undo()
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "history": h.Store.HistoryStatus()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": state, "history": history})
}

func (h *LedgerHandler) redoLedger(c *gin.Context) {
	state, history, err := h.Store.Redo()
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "history": h.Store.HistoryStatus()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": state, "history": history})
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

	if err := h.Store.ImportExcel(file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "imported"})
}

func (h *LedgerHandler) exportExcel(c *gin.Context) {
	data, err := h.Store.ExportExcel()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Disposition", "attachment; filename=ledger_export.xlsx")
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	http.ServeContent(c.Writer, c.Request, "ledger_export.xlsx", time.Now(), bytes.NewReader(data))
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

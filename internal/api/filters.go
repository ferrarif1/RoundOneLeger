package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"ledger/internal/models"
)

type filterParam struct {
	Property string      `json:"property"`
	Op       string      `json:"op"`
	Value    interface{} `json:"value"`
}

type sortParam struct {
	Property  string `json:"property"`
	Direction string `json:"direction"`
}

func parseFilters(c *gin.Context) ([]models.FilterClause, []models.SortClause, bool) {
	var filters []filterParam
	var sorts []sortParam
	validProp := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if raw := c.Query("filters"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &filters); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_filters"})
			return nil, nil, false
		}
	}
	if raw := c.Query("sort"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &sorts); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_sort"})
			return nil, nil, false
		}
	}
	var outFilters []models.FilterClause
	for _, f := range filters {
		prop := strings.TrimSpace(f.Property)
		if prop == "" || !validProp.MatchString(prop) {
			continue
		}
		op := strings.ToLower(strings.TrimSpace(f.Op))
		if op != "eq" && op != "contains" {
			continue
		}
		outFilters = append(outFilters, models.FilterClause{
			Property: prop,
			Op:       op,
			Value:    f.Value,
		})
	}
	var outSorts []models.SortClause
	for _, s := range sorts {
		if !validProp.MatchString(s.Property) {
			continue
		}
		dir := "asc"
		if strings.ToLower(s.Direction) == "desc" {
			dir = "desc"
		}
		outSorts = append(outSorts, models.SortClause{
			Property:  s.Property,
			Direction: dir,
		})
	}
	return outFilters, outSorts, true
}

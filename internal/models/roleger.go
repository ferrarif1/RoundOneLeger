package models

import "time"

// Roleger-style database models (draft) for multi-view, property-driven data.

type PropertyType string

const (
	PropertyText        PropertyType = "text"
	PropertyNumber      PropertyType = "number"
	PropertyDate        PropertyType = "date"
	PropertySelect      PropertyType = "select"
	PropertyMultiSelect PropertyType = "multi_select"
	PropertyRelation    PropertyType = "relation"
	PropertyCheckbox    PropertyType = "checkbox"
	PropertyURL         PropertyType = "url"
	PropertyEmail       PropertyType = "email"
	PropertyPhone       PropertyType = "phone"
	PropertyUser        PropertyType = "user"
	PropertyFormula     PropertyType = "formula"
	PropertyRollup      PropertyType = "rollup"
)

type Table struct {
	ID          string     `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	CreatedBy   string     `json:"created_by" db:"created_by"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	Version     int        `json:"version" db:"version"`
	Properties  []Property `json:"properties" db:"-"`
	Views       []View     `json:"views" db:"-"`
}

type Property struct {
	ID        string         `json:"id" db:"id"`
	TableID   string         `json:"table_id" db:"table_id"`
	Name      string         `json:"name" db:"name"`
	Type      PropertyType   `json:"type" db:"type"`
	Options   []SelectOption `json:"options,omitempty" db:"-"`
	Relation  *RelationMeta  `json:"relation,omitempty" db:"-"`
	Formula   string         `json:"formula,omitempty" db:"formula"`
	Rollup    *RollupMeta    `json:"rollup,omitempty" db:"-"`
	SortOrder int            `json:"sort_order" db:"sort_order"`
}

type SelectOption struct {
	ID    string `json:"id" db:"id"`
	Label string `json:"label" db:"label"`
	Color string `json:"color,omitempty" db:"color"`
}

type RelationMeta struct {
	TargetTableID string `json:"target_table_id" db:"target_table_id"`
	RelationType  string `json:"relation_type" db:"relation_type"` // m2m | o2m | o2o
}

type RollupMeta struct {
	SourcePropertyID   string `json:"source_property_id" db:"source_property_id"`
	RelationPropertyID string `json:"relation_property_id" db:"relation_property_id"`
	Aggregation        string `json:"aggregation" db:"aggregation"`
}

type ViewLayout string

const (
	ViewTable   ViewLayout = "table"
	ViewList    ViewLayout = "list"
	ViewGallery ViewLayout = "gallery"
	ViewKanban  ViewLayout = "kanban"
)

type View struct {
	ID      string       `json:"id" db:"id"`
	TableID string       `json:"table_id" db:"table_id"`
	Name    string       `json:"name" db:"name"`
	Layout  ViewLayout   `json:"layout" db:"layout"`
	Filters []ViewFilter `json:"filters" db:"-"`
	Sorts   []ViewSort   `json:"sorts" db:"-"`
	Group   *ViewGroup   `json:"group,omitempty" db:"-"`
	Columns []ViewColumn `json:"columns" db:"-"`
}

type ViewFilter struct {
	PropertyID string      `json:"property_id" db:"property_id"`
	Op         string      `json:"op" db:"op"`
	Value      interface{} `json:"value" db:"-"`
}

type ViewSort struct {
	PropertyID string `json:"property_id" db:"property_id"`
	Direction  string `json:"direction" db:"direction"` // asc | desc
}

type ViewGroup struct {
	PropertyID string `json:"property_id" db:"property_id"`
	Direction  string `json:"direction" db:"direction"`
}

type ViewColumn struct {
	PropertyID string `json:"property_id" db:"property_id"`
	Visible    bool   `json:"visible" db:"visible"`
	Width      int    `json:"width,omitempty" db:"width"`
	SortOrder  int    `json:"sort_order" db:"sort_order"`
}

type RecordItem struct {
	ID         string                 `json:"id" db:"id"`
	TableID    string                 `json:"table_id" db:"table_id"`
	Properties map[string]interface{} `json:"properties" db:"properties"` // JSONB map
	CreatedBy  string                 `json:"created_by" db:"created_by"`
	UpdatedBy  string                 `json:"updated_by" db:"updated_by"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
	Version    int                    `json:"version" db:"version"`
	TrashedAt  *time.Time             `json:"trashed_at,omitempty" db:"trashed_at"`
}

type Block struct {
	ID       string                 `json:"id" db:"id"`
	PageID   string                 `json:"page_id" db:"page_id"`
	Type     string                 `json:"type" db:"type"`
	Props    map[string]interface{} `json:"props" db:"props"` // JSONB
	Order    int                    `json:"order" db:"order"`
	ParentID string                 `json:"parent_id" db:"parent_id"`
}

// FilterClause represents a simple property filter.
type FilterClause struct {
	Property string      `json:"property"`
	Op       string      `json:"op"`
	Value    interface{} `json:"value"`
}

// SortClause represents a property sort.
type SortClause struct {
	Property  string `json:"property"`
	Direction string `json:"direction"` // asc|desc
}

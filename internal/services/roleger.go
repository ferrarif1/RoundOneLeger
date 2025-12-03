package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ledger/internal/models"
)

// RolegerService provides CRUD for tables/views/records/blocks.
// This is a draft skeleton; wire SQL queries and validations before use.
type RolegerService struct {
	DB *sql.DB
}

func NewRolegerService(db *sql.DB) *RolegerService {
	return &RolegerService{DB: db}
}

func (s *RolegerService) ListTables(ctx context.Context) ([]models.Table, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, description, created_by, updated_at, version FROM tables ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Table
	for rows.Next() {
		var t models.Table
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.UpdatedAt, &t.Version); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *RolegerService) ListViews(ctx context.Context, tableID string) ([]models.View, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, table_id, name, layout, filters, sorts, "group", columns FROM views WHERE table_id=$1 ORDER BY name`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.View
	for rows.Next() {
		var v models.View
		var filters, sorts, groupRaw, cols []byte
		if err := rows.Scan(&v.ID, &v.TableID, &v.Name, &v.Layout, &filters, &sorts, &groupRaw, &cols); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(filters, &v.Filters)
		_ = json.Unmarshal(sorts, &v.Sorts)
		if len(groupRaw) > 0 {
			var g models.ViewGroup
			if err := json.Unmarshal(groupRaw, &g); err == nil {
				v.Group = &g
			}
		}
		_ = json.Unmarshal(cols, &v.Columns)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *RolegerService) ListRecords(ctx context.Context, tableID string, limit, offset int, filters []models.FilterClause, sorts []models.SortClause) ([]models.RecordItem, int, error) {
	where := "table_id=$1 AND trashed_at IS NULL"
	args := []interface{}{tableID}
	argPos := 2
	// Build WHERE from filters (already sanitized upstream).
	for _, f := range filters {
		switch strings.ToLower(f.Op) {
		case "eq":
			where += fmt.Sprintf(" AND properties ->> '%s' = $%d", f.Property, argPos)
			args = append(args, f.Value)
			argPos++
		case "contains":
			where += fmt.Sprintf(" AND properties ->> '%s' ILIKE '%%' || $%d || '%%'", f.Property, argPos)
			args = append(args, f.Value)
			argPos++
		}
	}
	order := " ORDER BY updated_at DESC"
	if len(sorts) > 0 {
		var parts []string
		for _, srt := range sorts {
			dir := "ASC"
			if strings.ToLower(srt.Direction) == "desc" {
				dir = "DESC"
			}
			parts = append(parts, fmt.Sprintf("properties ->> '%s' %s", srt.Property, dir))
		}
		order = " ORDER BY " + strings.Join(parts, ", ")
	}
	query := fmt.Sprintf(`SELECT id, table_id, properties, created_by, updated_by, updated_at, version, trashed_at FROM records WHERE %s%s LIMIT $%d OFFSET $%d`, where, order, argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.RecordItem
	for rows.Next() {
		var r models.RecordItem
		var props []byte
		var trashed sql.NullTime
		if err := rows.Scan(&r.ID, &r.TableID, &props, &r.CreatedBy, &r.UpdatedBy, &r.UpdatedAt, &r.Version, &trashed); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(props, &r.Properties)
		if trashed.Valid {
			r.TrashedAt = &trashed.Time
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// Count with the same filters to keep pagination accurate.
	countQuery := fmt.Sprintf(`SELECT count(1) FROM records WHERE %s`, where)
	var total int
	if err := s.DB.QueryRowContext(ctx, countQuery, args[:argPos-1]...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *RolegerService) ListProperties(ctx context.Context, tableID string) ([]models.Property, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, table_id, name, type, options, relation, formula, rollup, sort_order FROM properties WHERE table_id=$1 ORDER BY sort_order`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Property
	for rows.Next() {
		var p models.Property
		var options, relation, rollup []byte
		if err := rows.Scan(&p.ID, &p.TableID, &p.Name, &p.Type, &options, &relation, &p.Formula, &rollup, &p.SortOrder); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(options, &p.Options)
		if len(relation) > 0 {
			var rel models.RelationMeta
			if err := json.Unmarshal(relation, &rel); err == nil {
				p.Relation = &rel
			}
		}
		if len(rollup) > 0 {
			var r models.RollupMeta
			if err := json.Unmarshal(rollup, &r); err == nil {
				p.Rollup = &r
			}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *RolegerService) CreateTable(ctx context.Context, table models.Table) (*models.Table, error) {
	if table.ID == "" {
		table.ID = generateID()
	}
	if table.UpdatedAt.IsZero() {
		table.UpdatedAt = time.Now()
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO tables (id, name, description, created_by, updated_at, version) VALUES ($1,$2,$3,$4,$5,$6)`,
		table.ID, table.Name, table.Description, table.CreatedBy, table.UpdatedAt, table.Version)
	if err != nil {
		return nil, err
	}
	return &table, nil
}

// generateID returns a simple time-based ID; replace with ULID/UUID as needed.
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}

func (s *RolegerService) UpdateView(ctx context.Context, view models.View) (*models.View, error) {
	filters, _ := json.Marshal(view.Filters)
	sorts, _ := json.Marshal(view.Sorts)
	groupRaw, _ := json.Marshal(view.Group)
	cols, _ := json.Marshal(view.Columns)
	_, err := s.DB.ExecContext(ctx, `
		UPDATE views SET name=$1, layout=$2, filters=$3, sorts=$4, "group"=$5, columns=$6
		WHERE id=$7 AND table_id=$8
	`, view.Name, view.Layout, filters, sorts, groupRaw, cols, view.ID, view.TableID)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *RolegerService) UpdateRecordProperties(ctx context.Context, tableID, recordID string, props map[string]interface{}) (*models.RecordItem, error) {
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	var updated models.RecordItem
	err = s.DB.QueryRowContext(ctx, `
		UPDATE records
		SET properties = $1, updated_at = $2, version = version + 1
		WHERE id = $3 AND table_id = $4
		RETURNING id, table_id, properties, created_by, updated_by, updated_at, version, trashed_at
	`, propsJSON, time.Now(), recordID, tableID).
		Scan(&updated.ID, &updated.TableID, &propsJSON, &updated.CreatedBy, &updated.UpdatedBy, &updated.UpdatedAt, &updated.Version, &updated.TrashedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(propsJSON, &updated.Properties)
	return &updated, nil
}

// UpdateRecordsBulk updates multiple records using a single SQL with unnest.
func (s *RolegerService) UpdateRecordsBulk(ctx context.Context, tableID string, updates map[string]map[string]interface{}) ([]models.RecordItem, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	const chunkSize = 200
	var results []models.RecordItem
	batch := func(ids []string, propsArr [][]byte) error {
		now := time.Now()
		rows, err := s.DB.QueryContext(ctx, `
			WITH payload AS (
				SELECT unnest($1::text[]) AS id, unnest($2::jsonb[]) AS props
			), updated AS (
				UPDATE records r
				SET properties = p.props,
					updated_at = $3,
					version = r.version + 1
				FROM payload p
				WHERE r.id = p.id AND r.table_id = $4
				RETURNING r.id, r.table_id, r.properties, r.created_by, r.updated_by, r.updated_at, r.version, r.trashed_at
			)
			SELECT * FROM updated
		`, ids, propsArr, now, tableID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r models.RecordItem
			var propsJSON []byte
			if err := rows.Scan(&r.ID, &r.TableID, &propsJSON, &r.CreatedBy, &r.UpdatedBy, &r.UpdatedAt, &r.Version, &r.TrashedAt); err != nil {
				return err
			}
			_ = json.Unmarshal(propsJSON, &r.Properties)
			results = append(results, r)
		}
		return rows.Err()
	}

	ids := make([]string, 0, len(updates))
	propsArr := make([][]byte, 0, len(updates))
	for id, props := range updates {
		ids = append(ids, id)
		j, err := json.Marshal(props)
		if err != nil {
			return nil, err
		}
		propsArr = append(propsArr, j)
		if len(ids) >= chunkSize {
			if err := batch(ids, propsArr); err != nil {
				return nil, err
			}
			ids = ids[:0]
			propsArr = propsArr[:0]
		}
	}
	if len(ids) > 0 {
		if err := batch(ids, propsArr); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (s *RolegerService) CreateRecord(ctx context.Context, tableID string, props map[string]interface{}, createdBy string) (*models.RecordItem, error) {
	if tableID == "" {
		return nil, sql.ErrNoRows
	}
	if props == nil {
		props = map[string]interface{}{}
	}
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	id := generateID()
	now := time.Now()
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO records (id, table_id, properties, created_by, updated_by, updated_at, version)
		VALUES ($1,$2,$3,$4,$4,$5,1)
	`, id, tableID, propsJSON, createdBy, now)
	if err != nil {
		return nil, err
	}
	return &models.RecordItem{
		ID:         id,
		TableID:    tableID,
		Properties: props,
		CreatedBy:  createdBy,
		UpdatedBy:  createdBy,
		UpdatedAt:  now,
		Version:    1,
	}, nil
}

func (s *RolegerService) DeleteRecord(ctx context.Context, tableID, recordID, actor string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE records
		SET trashed_at = $1, updated_by = $2, version = version + 1
		WHERE id = $3 AND table_id = $4
	`, time.Now(), actor, recordID, tableID)
	return err
}

func (s *RolegerService) CreateProperty(ctx context.Context, prop models.Property) (*models.Property, error) {
	if prop.ID == "" {
		prop.ID = generateID()
	}
	options, _ := json.Marshal(prop.Options)
	relation, _ := json.Marshal(prop.Relation)
	rollup, _ := json.Marshal(prop.Rollup)
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO properties (id, table_id, name, type, options, relation, formula, rollup, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, prop.ID, prop.TableID, prop.Name, prop.Type, options, relation, prop.Formula, rollup, prop.SortOrder)
	if err != nil {
		return nil, err
	}
	return &prop, nil
}

func (s *RolegerService) CreateView(ctx context.Context, view models.View) (*models.View, error) {
	if view.ID == "" {
		view.ID = generateID()
	}
	filters, _ := json.Marshal(view.Filters)
	sorts, _ := json.Marshal(view.Sorts)
	groupRaw, _ := json.Marshal(view.Group)
	cols, _ := json.Marshal(view.Columns)
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO views (id, table_id, name, layout, filters, sorts, "group", columns)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, view.ID, view.TableID, view.Name, view.Layout, filters, sorts, groupRaw, cols)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

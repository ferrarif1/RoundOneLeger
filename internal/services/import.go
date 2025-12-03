package services

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"ledger/internal/models"
)

type ImportService struct {
	DB        *sql.DB
	BatchSize int
}

func NewImportService(db *sql.DB) *ImportService {
	return &ImportService{DB: db, BatchSize: resolveBatchSize()}
}

func (s *ImportService) CreateTask(ctx context.Context, tableID string, payloadPath string) (*models.ImportTask, error) {
	id := generateID()
	task := &models.ImportTask{
		ID:          id,
		TableID:     tableID,
		Status:      models.ImportPending,
		Progress:    0,
		PayloadPath: payloadPath,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO import_tasks (id, table_id, status, progress, error, payload_path, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, task.ID, task.TableID, task.Status, task.Progress, task.Error, task.PayloadPath, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *ImportService) UpdateTaskStatus(ctx context.Context, id string, status models.ImportTaskStatus, progress int, errMsg string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE import_tasks SET status=$1, progress=$2, error=$3, updated_at=$4 WHERE id=$5
	`, status, progress, errMsg, time.Now(), id)
	return err
}

func (s *ImportService) GetTask(ctx context.Context, id string) (*models.ImportTask, error) {
	var t models.ImportTask
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, table_id, status, progress, error, payload_path, created_at, updated_at
		FROM import_tasks WHERE id=$1
	`, id).Scan(&t.ID, &t.TableID, &t.Status, &t.Progress, &t.Error, &t.PayloadPath, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Cleanup removes payload file after completion.
func (s *ImportService) Cleanup(task *models.ImportTask) {
	if task.PayloadPath != "" {
		_ = os.Remove(task.PayloadPath)
	}
}

// ProcessCSV reads CSV file and inserts records in batches.
func (s *ImportService) ProcessCSV(ctx context.Context, task *models.ImportTask) error {
	// First pass: count rows for progress percentage
	totalRows, err := countCSVRows(task.PayloadPath)
	if err != nil {
		return err
	}
	f, err := os.Open(task.PayloadPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	batchSize := s.BatchSize
	var batch []map[string]interface{}
	processed := 0
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		props := make(map[string]interface{})
		for i, h := range headers {
			if i < len(record) {
				props[h] = record[i]
			}
		}
		batch = append(batch, props)
		if len(batch) >= batchSize {
			if err := s.insertBatch(ctx, task.TableID, batch); err != nil {
				return err
			}
			processed += len(batch)
			progress := processed
			if totalRows > 0 {
				progress = int(float64(processed) / float64(totalRows) * 100)
				if progress > 100 {
					progress = 100
				}
			}
			_ = s.UpdateTaskStatus(ctx, task.ID, models.ImportRunning, progress, "")
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := s.insertBatch(ctx, task.TableID, batch); err != nil {
			return err
		}
		processed += len(batch)
		progress := processed
		if totalRows > 0 {
			progress = int(float64(processed) / float64(totalRows) * 100)
			if progress > 100 {
				progress = 100
			}
		}
		_ = s.UpdateTaskStatus(ctx, task.ID, models.ImportRunning, progress, "")
	}
	return s.UpdateTaskStatus(ctx, task.ID, models.ImportSuccess, 100, "")
}

func (s *ImportService) insertBatch(ctx context.Context, tableID string, propsList []map[string]interface{}) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO records (id, table_id, properties, created_by, updated_by, updated_at, version)
		VALUES ($1,$2,$3,$4,$4,$5,1)
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, props := range propsList {
		id := generateID()
		propsJSON, _ := json.Marshal(props)
		if _, err := stmt.ExecContext(ctx, id, tableID, propsJSON, "system", now); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func countCSVRows(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	// consume header
	if _, err := reader.Read(); err != nil {
		return 0, err
	}
	count := 0
	for {
		_, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		count++
	}
	return count, nil
}

func resolveBatchSize() int {
	if v := os.Getenv("IMPORT_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return 200
}

// ProcessXLSX handles Excel import by streaming rows from the first sheet.
func (s *ImportService) ProcessXLSX(ctx context.Context, task *models.ImportTask) error {
	totalRows, headers, err := countXLSXRows(task.PayloadPath)
	if err != nil {
		return err
	}
	f, err := excelize.OpenFile(task.PayloadPath)
	if err != nil {
		return err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets in workbook")
	}
	sheet := sheets[0]
	rows, err := f.Rows(sheet)
	if err != nil {
		return err
	}
	defer rows.Close()
	batchSize := s.BatchSize
	var batch []map[string]interface{}
	processed := 0
	rowIndex := 0
	for rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			return err
		}
		props := make(map[string]interface{})
		for i, h := range headers {
			if i < len(row) {
				props[h] = row[i]
			}
		}
		batch = append(batch, props)
		if len(batch) >= batchSize {
			if err := s.insertBatch(ctx, task.TableID, batch); err != nil {
				return err
			}
			processed += len(batch)
			progress := processed
			if totalRows > 0 {
				progress = int(float64(processed) / float64(totalRows) * 100)
				if progress > 100 {
					progress = 100
				}
			}
			_ = s.UpdateTaskStatus(ctx, task.ID, models.ImportRunning, progress, "")
			batch = batch[:0]
		}
		rowIndex++
	}
	if len(batch) > 0 {
		if err := s.insertBatch(ctx, task.TableID, batch); err != nil {
			return err
		}
		processed += len(batch)
		progress := processed
		if totalRows > 0 {
			progress = int(float64(processed) / float64(totalRows) * 100)
			if progress > 100 {
				progress = 100
			}
		}
		_ = s.UpdateTaskStatus(ctx, task.ID, models.ImportRunning, progress, "")
	}
	return s.UpdateTaskStatus(ctx, task.ID, models.ImportSuccess, 100, "")
}

func countXLSXRows(path string) (int, []string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, nil, fmt.Errorf("no sheets in workbook")
	}
	sheet := sheets[0]
	rows, err := f.Rows(sheet)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var headers []string
	count := 0
	rowIndex := 0
	for rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			return 0, nil, err
		}
		if rowIndex == 0 {
			headers = row
		} else {
			count++
		}
		rowIndex++
	}
	if count == 0 {
		return 0, headers, fmt.Errorf("no data rows")
	}
	return count, headers, nil
}

package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type DatasetRepo struct{ pool *pgxpool.Pool }

func NewDatasetRepo(pool *pgxpool.Pool) *DatasetRepo { return &DatasetRepo{pool: pool} }

const datasetCols = `d.id, d.owner_id, u.display_name, d.original_filename, coalesce(d.label, ''),
	d.size_bytes, d.sha256, d.status, d.row_counts, d.year_min, d.year_max,
	coalesce(d.error_message, ''), d.uploaded_at, d.finished_at, d.deleted_at`

func scanDataset(r pgx.Row) (domain.Dataset, error) {
	var d domain.Dataset
	err := r.Scan(&d.ID, &d.OwnerID, &d.OwnerName, &d.OriginalFilename, &d.Label,
		&d.SizeBytes, &d.SHA256, &d.Status, &d.RowCounts, &d.YearMin, &d.YearMax,
		&d.ErrorMessage, &d.UploadedAt, &d.FinishedAt, &d.DeletedAt)
	return d, err
}

func (r *DatasetRepo) Create(ctx context.Context, d domain.Dataset, fileKey string) (domain.Dataset, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO datasets (owner_id, original_filename, file_key, sha256, size_bytes, status, label)
		VALUES ($1, $2, $3, $4, $5, $6, nullif($7, ''))
		RETURNING id`,
		d.OwnerID, d.OriginalFilename, fileKey, d.SHA256, d.SizeBytes,
		string(domain.StatusProcessing), d.Label).Scan(&id)
	if err != nil {
		return domain.Dataset{}, err
	}
	return r.ByID(ctx, id)
}

// ByID อ่านได้ทั้งชุดที่ยังอยู่และชุดที่ถูกลบ เพราะหน้าถังขยะและการกู้คืนต้องใช้
// ส่วน query ของ dashboard ทั้งหมดไปผ่าน v_active_datasets แทน
func (r *DatasetRepo) ByID(ctx context.Context, id uuid.UUID) (domain.Dataset, error) {
	d, err := scanDataset(r.pool.QueryRow(ctx, `
		SELECT `+datasetCols+`
		FROM datasets d JOIN users u ON u.id = d.owner_id
		WHERE d.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Dataset{}, domain.ErrDatasetNotFound
	}
	return d, err
}

func (r *DatasetRepo) FileKey(ctx context.Context, id uuid.UUID) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `SELECT file_key FROM datasets WHERE id = $1`, id).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrDatasetNotFound
	}
	return key, err
}

func (r *DatasetRepo) List(ctx context.Context, f ports.DatasetFilter) ([]domain.Dataset, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// ทุกคนในองค์กรเห็นไฟล์ของกันและกัน จึงไม่กรองตามเจ้าของเว้นแต่ผู้เรียกขอมาเอง
	where := `d.deleted_at IS NULL`
	if f.Deleted {
		where = `d.deleted_at IS NOT NULL`
	}
	args := []any{limit, f.Offset}
	if f.OwnerID != nil {
		where += fmt.Sprintf(` AND d.owner_id = $%d`, len(args)+1)
		args = append(args, *f.OwnerID)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+datasetCols+`
		FROM datasets d JOIN users u ON u.id = d.owner_id
		WHERE `+where+`
		ORDER BY d.uploaded_at DESC
		LIMIT $1 OFFSET $2`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Dataset{}
	for rows.Next() {
		d, err := scanDataset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// FindBySHA ใช้เตือนว่าไฟล์นี้เคยอัปโหลดแล้ว แต่ไม่บล็อกการอัปโหลดซ้ำ
// เพราะผู้ใช้อาจตั้งใจสร้างชุดใหม่จากไฟล์เดิมจริง ๆ
func (r *DatasetRepo) FindBySHA(ctx context.Context, sha []byte) (domain.Dataset, error) {
	d, err := scanDataset(r.pool.QueryRow(ctx, `
		SELECT `+datasetCols+`
		FROM datasets d JOIN users u ON u.id = d.owner_id
		WHERE d.sha256 = $1 AND d.deleted_at IS NULL
		ORDER BY d.uploaded_at DESC LIMIT 1`, sha))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Dataset{}, domain.ErrDatasetNotFound
	}
	return d, err
}

func (r *DatasetRepo) MarkReady(ctx context.Context, id uuid.UUID, counts map[string]int, yearMin, yearMax *int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE datasets
		SET status = 'ready', row_counts = $2, year_min = $3, year_max = $4,
		    error_message = NULL, finished_at = now()
		WHERE id = $1`, id, counts, yearMin, yearMax)
	return err
}

func (r *DatasetRepo) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE datasets SET status = 'failed', error_message = $2, finished_at = now()
		WHERE id = $1`, id, reason)
	return err
}

// SoftDelete ตั้งเวลาที่ลบเท่านั้น ไม่แตะข้อมูล fact เลย ทำให้กู้คืนได้ทันทีและประวัติไม่ขาดตอน
func (r *DatasetRepo) SoftDelete(ctx context.Context, id, by uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE datasets SET deleted_at = now(), deleted_by = $2
		WHERE id = $1 AND deleted_at IS NULL`, id, by)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDatasetNotFound
	}
	return nil
}

func (r *DatasetRepo) Restore(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE datasets SET deleted_at = NULL, deleted_by = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDatasetNotFound
	}
	return nil
}

func (r *DatasetRepo) SaveIssues(ctx context.Context, id uuid.UUID, issues []domain.ImportIssue) error {
	if len(issues) == 0 {
		return nil
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"import_issues"},
		[]string{"dataset_id", "sheet", "row_no", "severity", "field", "message"},
		pgx.CopyFromSlice(len(issues), func(i int) ([]any, error) {
			s := issues[i]
			var rowNo *int
			if s.RowNo > 0 {
				rowNo = &s.RowNo
			}
			return []any{id, s.Sheet, rowNo, string(s.Severity), s.Field, s.Message}, nil
		}))
	return err
}

func (r *DatasetRepo) Issues(ctx context.Context, id uuid.UUID, limit, offset int) ([]domain.ImportIssue, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM import_issues WHERE dataset_id = $1`, id).Scan(&total); err != nil {
		return nil, 0, err
	}

	// เรียง blocking ก่อน warning ก่อน info เพื่อให้สิ่งที่ต้องแก้จริงอยู่หน้าแรกเสมอ
	rows, err := r.pool.Query(ctx, `
		SELECT sheet, coalesce(row_no, 0), severity, coalesce(field, ''), message
		FROM import_issues
		WHERE dataset_id = $1
		ORDER BY CASE severity WHEN 'blocking' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END,
		         sheet, row_no NULLS FIRST, id
		LIMIT $2 OFFSET $3`, id, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []domain.ImportIssue{}
	for rows.Next() {
		var i domain.ImportIssue
		if err := rows.Scan(&i.Sheet, &i.RowNo, &i.Severity, &i.Field, &i.Message); err != nil {
			return nil, 0, err
		}
		out = append(out, i)
	}
	return out, total, rows.Err()
}

func (r *DatasetRepo) PurgeExpired(ctx context.Context, olderThan time.Time) (int, error) {
	// ล้างเฉพาะข้อมูล fact ส่วน record ของ dataset และไฟล์ต้นฉบับยังอยู่ ให้ re-import ได้
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM datasets
		WHERE deleted_at IS NOT NULL AND deleted_at < $1 AND purged_at IS NULL`, olderThan)
	if err != nil {
		return 0, err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, id := range ids {
		if err := r.deleteFacts(ctx, r.pool, id); err != nil {
			return 0, err
		}
		if _, err := r.pool.Exec(ctx, `UPDATE datasets SET purged_at = now() WHERE id = $1`, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

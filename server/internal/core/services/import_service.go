package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

var (
	ErrFileTooLarge = errors.New("ไฟล์ใหญ่เกินกำหนด")
	ErrNotExcelFile = errors.New("ไฟล์นี้ไม่ใช่ .xlsx — ตรวจสอบว่าบันทึกจาก Excel ในรูปแบบ Workbook (.xlsx) แล้วหรือยัง")
)

type ImportService struct {
	datasets ports.DatasetRepo
	files    ports.FileStore
	reader   ports.WorkbookReader
	audit    ports.AuditRepo
	regions  ports.RegionRepo
	log      *slog.Logger
	maxBytes int64
}

func NewImportService(
	datasets ports.DatasetRepo, files ports.FileStore, reader ports.WorkbookReader,
	audit ports.AuditRepo, regions ports.RegionRepo, log *slog.Logger, maxMB int64,
) *ImportService {
	return &ImportService{
		datasets: datasets, files: files, reader: reader, audit: audit,
		regions: regions, log: log, maxBytes: maxMB * 1024 * 1024,
	}
}

// UploadResult คือสิ่งที่ตอบกลับทันทีหลังรับไฟล์ ก่อนการประมวลผลจะเสร็จ
type UploadResult struct {
	Dataset domain.Dataset `json:"dataset"`
	// DuplicateOf ชี้ไปยังชุดข้อมูลเดิมที่มีเนื้อไฟล์เหมือนกันทุกไบต์
	// เป็นเพียงคำเตือน ไม่ได้ขัดขวางการอัปโหลด เพราะผู้ใช้อาจตั้งใจสร้างชุดใหม่จากไฟล์เดิม
	DuplicateOf *domain.Dataset `json:"duplicate_of,omitempty"`
}

// Upload รับไฟล์ เก็บลง storage สร้าง record แล้วคืนทันที
// การอ่านและเขียนลงฐานข้อมูลเกิดขึ้นเบื้องหลัง หน้าเว็บติดตามสถานะด้วยการ poll
func (s *ImportService) Upload(ctx context.Context, owner domain.User, filename, label string, src io.Reader, ip string) (UploadResult, error) {
	// อ่านทั้งไฟล์ลงหน่วยความจำครั้งเดียว เพราะต้องใช้ทั้งคำนวณ hash ตรวจชนิดไฟล์ และเขียนลงดิสก์
	// ที่ขนาดจำกัด 25 MB ถือว่าคุ้มกว่าการอ่านไฟล์ซ้ำหลายรอบ
	limited := io.LimitReader(src, s.maxBytes+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return UploadResult{}, err
	}
	if int64(len(buf)) > s.maxBytes {
		return UploadResult{}, fmt.Errorf("%w (จำกัด %d MB)", ErrFileTooLarge, s.maxBytes/1024/1024)
	}
	if !looksLikeXLSX(buf) {
		return UploadResult{}, ErrNotExcelFile
	}

	sum := sha256.Sum256(buf)
	sha := sum[:]

	var duplicate *domain.Dataset
	if existing, err := s.datasets.FindBySHA(ctx, sha); err == nil {
		duplicate = &existing
	}

	key, err := randomFileKey()
	if err != nil {
		return UploadResult{}, err
	}
	size, err := s.files.Put(ctx, key, bytes.NewReader(buf))
	if err != nil {
		return UploadResult{}, fmt.Errorf("บันทึกไฟล์ไม่สำเร็จ: %w", err)
	}

	ds, err := s.datasets.Create(ctx, domain.Dataset{
		OwnerID:          owner.ID,
		OriginalFilename: sanitizeFilename(filename),
		Label:            strings.TrimSpace(label),
		SizeBytes:        size,
		SHA256:           sha,
	}, key)
	if err != nil {
		// เก็บกวาดไฟล์ที่เขียนไปแล้ว ไม่ให้เหลือไฟล์กำพร้าที่ไม่มี record ชี้ถึง
		_ = s.files.Delete(ctx, key)
		return UploadResult{}, err
	}

	_ = s.audit.Record(ctx, &owner.ID, "dataset.upload", ds.ID.String(),
		map[string]any{"filename": ds.OriginalFilename, "size": size}, ip)

	return UploadResult{Dataset: ds, DuplicateOf: duplicate}, nil
}

// Process อ่านไฟล์ที่เก็บไว้แล้วเขียนลงฐานข้อมูล
//
// ตั้งใจให้เรียกได้ซ้ำอย่างปลอดภัย เพราะ ReplaceFacts ล้างของเดิมก่อนเขียนเสมอ
func (s *ImportService) Process(ctx context.Context, datasetID uuid.UUID) error {
	key, err := s.datasets.FileKey(ctx, datasetID)
	if err != nil {
		return err
	}
	f, err := s.files.Open(ctx, key)
	if err != nil {
		return s.fail(ctx, datasetID, "เปิดไฟล์ที่เก็บไว้ไม่ได้: "+err.Error())
	}
	defer f.Close()

	wb, err := s.reader.Read(f)
	if err != nil {
		return s.fail(ctx, datasetID, err.Error())
	}

	// บันทึกปัญหาที่พบก่อนเสมอ แม้จะล้มเหลว เพื่อให้ผู้ใช้เห็นว่าไฟล์ผิดตรงไหน
	if err := s.datasets.SaveIssues(ctx, datasetID, wb.Issues); err != nil {
		s.log.Error("บันทึกรายการปัญหาไม่สำเร็จ", "dataset", datasetID, "err", err)
	}

	if wb.HasBlockingIssue() {
		return s.fail(ctx, datasetID, firstBlockingMessage(wb.Issues))
	}

	s.applyRegions(ctx, wb)

	if err := s.datasets.ReplaceFacts(ctx, datasetID, wb); err != nil {
		return s.fail(ctx, datasetID, err.Error())
	}

	var yearMin, yearMax *int
	if min, max, ok := wb.Years(); ok {
		yearMin, yearMax = &min, &max
	}
	if err := s.datasets.MarkReady(ctx, datasetID, wb.RowCounts(), yearMin, yearMax); err != nil {
		return err
	}

	s.log.Info("นำเข้าชุดข้อมูลสำเร็จ",
		"dataset", datasetID, "sheets", len(wb.SheetsRead), "issues", len(wb.Issues))
	return nil
}

// ProcessAsync รันการประมวลผลเบื้องหลัง โดยใช้ context แยกจาก request
// เพื่อไม่ให้งานถูกยกเลิกกลางคันเมื่อผู้ใช้ปิดหน้าเว็บ
func (s *ImportService) ProcessAsync(datasetID uuid.UUID) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := s.Process(ctx, datasetID); err != nil {
			s.log.Error("นำเข้าชุดข้อมูลไม่สำเร็จ", "dataset", datasetID, "err", err)
		}
	}()
}

func (s *ImportService) Summary(ctx context.Context, datasetID uuid.UUID) (domain.ImportSummary, error) {
	ds, err := s.datasets.ByID(ctx, datasetID)
	if err != nil {
		return domain.ImportSummary{}, err
	}
	issues, _, err := s.datasets.Issues(ctx, datasetID, 500, 0)
	if err != nil {
		return domain.ImportSummary{}, err
	}

	sum := domain.ImportSummary{
		DatasetID: ds.ID,
		Status:    ds.Status,
		RowCounts: ds.RowCounts,
		Issues:    issues,
	}
	for name, n := range ds.RowCounts {
		if n > 0 {
			sum.SheetsRead = append(sum.SheetsRead, name)
		}
	}
	sum.Tally(issues)
	return sum, nil
}

// Delete เป็น soft delete ตามที่ตกลงไว้ ข้อมูล fact ยังอยู่ครบและกู้คืนได้ทันที
func (s *ImportService) Delete(ctx context.Context, actor domain.User, datasetID uuid.UUID, ip string) error {
	ds, err := s.datasets.ByID(ctx, datasetID)
	if err != nil {
		return err
	}
	// ตรวจสิทธิ์ที่ชั้นนี้ ไม่ใช่แค่ซ่อนปุ่มในหน้าเว็บ
	if !ds.CanBeDeletedBy(actor) {
		return domain.ErrNotOwner
	}
	if err := s.datasets.SoftDelete(ctx, datasetID, actor.ID); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, &actor.ID, "dataset.delete", datasetID.String(),
		map[string]any{"filename": ds.OriginalFilename}, ip)
	return nil
}

func (s *ImportService) Restore(ctx context.Context, actor domain.User, datasetID uuid.UUID, ip string) error {
	ds, err := s.datasets.ByID(ctx, datasetID)
	if err != nil {
		return err
	}
	if !ds.CanBeDeletedBy(actor) {
		return domain.ErrNotOwner
	}
	if err := s.datasets.Restore(ctx, datasetID); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, &actor.ID, "dataset.restore", datasetID.String(), nil, ip)
	return nil
}

func (s *ImportService) List(ctx context.Context, actor domain.User, f ports.DatasetFilter) ([]domain.Dataset, error) {
	out, err := s.datasets.List(ctx, f)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].CanDelete = out[i].CanBeDeletedBy(actor)
	}
	return out, nil
}

// applyRegions ทับภาคที่ตัวอ่านเดาไว้ด้วยค่าที่ admin ตั้งไว้ในฐานข้อมูล
//
// ไฟล์ Excel ไม่มีคอลัมน์ภาค ตัวอ่านจึงเติมค่าตั้งต้นให้ก่อน
// แต่ถ้า admin เคยแก้ไว้ ค่านั้นต้องชนะเสมอ ไม่งั้นการแก้จะหายทุกครั้งที่อัปโหลดไฟล์ใหม่
func (s *ImportService) applyRegions(ctx context.Context, wb *domain.Workbook) {
	saved, err := s.regions.All(ctx)
	if err != nil {
		s.log.Warn("อ่านภาคของศูนย์จากฐานข้อมูลไม่สำเร็จ ใช้ค่าตั้งต้นแทน", "err", err)
		return
	}
	for i, c := range wb.Customers {
		if region, ok := saved[c.Code]; ok {
			wb.Customers[i].Region = region
		}
	}
}

func (s *ImportService) fail(ctx context.Context, id uuid.UUID, reason string) error {
	if err := s.datasets.MarkFailed(ctx, id, reason); err != nil {
		return err
	}
	return fmt.Errorf("นำเข้าไม่สำเร็จ: %s", reason)
}

func firstBlockingMessage(issues []domain.ImportIssue) string {
	for _, i := range issues {
		if i.Severity == domain.SeverityBlocking {
			if i.Sheet != "" {
				return i.Sheet + ": " + i.Message
			}
			return i.Message
		}
	}
	return "พบปัญหาที่ทำให้นำเข้าต่อไม่ได้"
}

// looksLikeXLSX ตรวจจากเนื้อไฟล์จริง ไม่เชื่อนามสกุลที่ผู้ใช้ส่งมา
// ไฟล์ .xlsx คือ ZIP archive จึงต้องขึ้นต้นด้วยลายเซ็นของ ZIP
func looksLikeXLSX(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	return b[0] == 'P' && b[1] == 'K' && (b[2] == 3 || b[2] == 5 || b[2] == 7)
}

// sanitizeFilename เก็บไว้แค่แสดงผล ไม่เคยใช้เขียนไฟล์จริง
// ตัดพาธออกให้หมดเผื่อ browser ส่งพาธเต็มมา และจำกัดความยาวไม่ให้ล้นหน้าจอ
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, name)
	if name == "" {
		name = "uploaded.xlsx"
	}
	if r := []rune(name); len(r) > 180 {
		name = string(r[:180])
	}
	return name
}

func randomFileKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ".xlsx", nil
}

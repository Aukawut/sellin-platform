package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type DatasetStatus string

const (
	StatusProcessing DatasetStatus = "processing"
	StatusReady      DatasetStatus = "ready"
	StatusFailed     DatasetStatus = "failed"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityBlocking Severity = "blocking"
)

var (
	ErrDatasetNotFound = errors.New("ไม่พบชุดข้อมูลที่ระบุ")
	ErrDatasetNotReady = errors.New("ชุดข้อมูลนี้ยังนำเข้าไม่เสร็จ")
	ErrNotOwner        = errors.New("ลบหรือกู้คืนได้เฉพาะไฟล์ของตัวเอง")
)

// Dataset คือไฟล์ Excel หนึ่งไฟล์ที่ถูกอัปโหลด พร้อมข้อมูลทั้งหมดที่แตกออกมาจากไฟล์นั้น
// การอัปโหลดไฟล์ใหม่จะสร้าง Dataset ใหม่เสมอ ไม่เคยทับของเดิม
type Dataset struct {
	ID               uuid.UUID      `json:"id"`
	OwnerID          uuid.UUID      `json:"owner_id"`
	OwnerName        string         `json:"owner_name"`
	OriginalFilename string         `json:"original_filename"`
	Label            string         `json:"label,omitempty"`
	SizeBytes        int64          `json:"size_bytes"`
	SHA256           []byte         `json:"-"`
	Status           DatasetStatus  `json:"status"`
	RowCounts        map[string]int `json:"row_counts"`
	YearMin          *int           `json:"year_min,omitempty"`
	YearMax          *int           `json:"year_max,omitempty"`
	ErrorMessage     string         `json:"error_message,omitempty"`
	UploadedAt       time.Time      `json:"uploaded_at"`
	FinishedAt       *time.Time     `json:"finished_at,omitempty"`
	DeletedAt        *time.Time     `json:"deleted_at,omitempty"`

	// CanDelete คำนวณต่อผู้ใช้ที่เรียก ไม่ได้เก็บใน DB
	CanDelete bool `json:"can_delete"`
}

func (d Dataset) IsDeleted() bool { return d.DeletedAt != nil }

// CanBeDeletedBy สะท้อนกติกาที่ตกลงไว้: ทุกคนในองค์กรเห็นไฟล์ของกันและกัน
// แต่ลบได้เฉพาะไฟล์ของตัวเอง ส่วน admin ลบได้ทุกไฟล์
func (d Dataset) CanBeDeletedBy(u User) bool {
	return u.Role == RoleAdmin || d.OwnerID == u.ID
}

type ImportIssue struct {
	Sheet    string   `json:"sheet"`
	RowNo    int      `json:"row_no,omitempty"`
	Severity Severity `json:"severity"`
	Field    string   `json:"field,omitempty"`
	Message  string   `json:"message"`
}

// ImportSummary คือสิ่งที่หน้าเว็บแสดงหลังนำเข้าเสร็จ
type ImportSummary struct {
	DatasetID  uuid.UUID      `json:"dataset_id"`
	Status     DatasetStatus  `json:"status"`
	SheetsRead []string       `json:"sheets_read"`
	RowCounts  map[string]int `json:"row_counts"`
	Counts     struct {
		Info     int `json:"info"`
		Warning  int `json:"warning"`
		Blocking int `json:"blocking"`
	} `json:"issue_counts"`
	Issues []ImportIssue `json:"issues,omitempty"`
}

func (s *ImportSummary) Tally(issues []ImportIssue) {
	for _, i := range issues {
		switch i.Severity {
		case SeverityInfo:
			s.Counts.Info++
		case SeverityWarning:
			s.Counts.Warning++
		case SeverityBlocking:
			s.Counts.Blocking++
		}
	}
}

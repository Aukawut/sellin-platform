// Package filestore เก็บไฟล์ Excel ต้นฉบับลง volume ในเครื่อง
//
// ชื่อไฟล์บนดิสก์เป็นค่าสุ่มที่ระบบสร้างเอง ไม่เคยใช้ชื่อที่ผู้ใช้ส่งมา
// เพื่อไม่ให้ชื่อไฟล์ที่ตั้งใจร้ายพาไปเขียนนอกโฟลเดอร์ที่กำหนด
package filestore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("สร้างโฟลเดอร์เก็บไฟล์ไม่สำเร็จ: %w", err)
	}
	return &Local{root: abs}, nil
}

// path ตรวจว่า key ไม่พาออกนอก root ก่อนคืนพาธจริงเสมอ
func (l *Local) path(key string) (string, error) {
	if key == "" || strings.ContainsAny(key, `/\`) || key == "." || key == ".." {
		return "", fmt.Errorf("ชื่อไฟล์ไม่ถูกต้อง")
	}
	full := filepath.Join(l.root, key)
	if !strings.HasPrefix(full, l.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("ชื่อไฟล์ไม่ถูกต้อง")
	}
	return full, nil
}

func (l *Local) Put(ctx context.Context, key string, r io.Reader) (int64, error) {
	full, err := l.path(key)
	if err != nil {
		return 0, err
	}

	// เขียนลงไฟล์ชั่วคราวก่อนแล้วค่อย rename เพื่อไม่ให้มีไฟล์ครึ่ง ๆ กลาง ๆ ถ้าเขียนล้มเหลว
	tmp, err := os.CreateTemp(l.root, ".upload-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	n, err := io.Copy(tmp, r)
	if err != nil {
		return 0, err
	}
	if err := tmp.Sync(); err != nil {
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		return 0, err
	}
	if err := os.Rename(tmpName, full); err != nil {
		return 0, err
	}
	return n, nil
}

func (l *Local) Open(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	full, err := l.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("เปิดไฟล์ที่เก็บไว้ไม่ได้: %w", err)
	}
	return f, nil
}

func (l *Local) Delete(ctx context.Context, key string) error {
	full, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

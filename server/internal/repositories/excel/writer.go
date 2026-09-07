package excel

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// Workbook ห่อ excelize ไว้ให้ service เขียนข้อมูลออกไฟล์โดยไม่ต้องรู้จักไลบรารีนี้
//
// เป็น adapter ฝั่งขาออกคู่กับ Reader ที่เป็นขาเข้า
// ตรรกะว่าจะส่งออกอะไรบ้างอยู่ใน service ส่วนที่นี่รู้แค่วิธีเขียนลงไฟล์ Excel
type Workbook struct {
	f   *excelize.File
	err error
}

func NewWorkbook() *Workbook {
	f := excelize.NewFile()
	// excelize สร้าง Sheet1 มาให้เสมอ ลบทิ้งหลังเพิ่ม sheet จริงแล้ว
	return &Workbook{f: f}
}

// AddSheet เพิ่มแผ่นใหม่พร้อมหัวตารางและข้อมูล
//
// เขียนทีละแถวด้วย SetSheetRow ซึ่งเร็วกว่าการตั้งค่าทีละเซลล์อย่างมาก
// ที่ระดับหลักหมื่นแถวความต่างนี้คือหลักวินาที
func (w *Workbook) AddSheet(name string, header []string, rows [][]any) {
	if w.err != nil {
		return
	}
	if _, err := w.f.NewSheet(name); err != nil {
		w.err = fmt.Errorf("สร้าง sheet %q ไม่สำเร็จ: %w", name, err)
		return
	}

	headerRow := make([]any, len(header))
	for i, h := range header {
		headerRow[i] = h
	}
	if err := w.f.SetSheetRow(name, "A1", &headerRow); err != nil {
		w.err = err
		return
	}

	for i, row := range rows {
		cell := fmt.Sprintf("A%d", i+2)
		r := row
		if err := w.f.SetSheetRow(name, cell, &r); err != nil {
			w.err = err
			return
		}
	}

	// ตรึงแถวหัวตารางไว้ เพื่อให้เลื่อนดูข้อมูลหลักหมื่นแถวแล้วยังรู้ว่าคอลัมน์ไหนคืออะไร
	_ = w.f.SetPanes(name, &excelize.Panes{
		Freeze: true, Split: false, XSplit: 0, YSplit: 1,
		TopLeftCell: "A2", ActivePane: "bottomLeft",
	})

	w.setDimension(name, len(header), len(rows)+1)
}

// setDimension เขียนแท็ก <dimension> ให้ชัดเจน
//
// แท็กนี้ไม่บังคับตามสเปกและ Excel เองก็อ่านไฟล์ได้โดยไม่มีมัน
// แต่เครื่องมืออื่นหลายตัว เช่น openpyxl ในโหมดอ่านอย่างเดียว เชื่อค่านี้แล้วหยุดอ่านแค่ช่วงที่ระบุ
// ถ้าไม่เขียนไว้ ไฟล์ที่ส่งออกจะดูเหมือนมีข้อมูลแค่เซลล์เดียวเมื่อเปิดด้วยเครื่องมือเหล่านั้น
func (w *Workbook) setDimension(sheet string, cols, rows int) {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	last, err := excelize.ColumnNumberToName(cols)
	if err != nil {
		return
	}
	_ = w.f.SetSheetDimension(sheet, fmt.Sprintf("A1:%s%d", last, rows))
}

// SetColumnWidths ตั้งความกว้างคอลัมน์ให้ข้อความไทยยาว ๆ ไม่ถูกบีบจนอ่านไม่ออก
func (w *Workbook) SetColumnWidths(sheet string, widths map[string]float64) {
	if w.err != nil {
		return
	}
	for col, width := range widths {
		_ = w.f.SetColWidth(sheet, col, col, width)
	}
}

// WriteTo เขียนไฟล์ออกไปยังปลายทาง แล้วปิดทรัพยากรของ excelize
func (w *Workbook) WriteTo(dst io.Writer, firstSheet string) error {
	defer w.f.Close()

	if w.err != nil {
		return w.err
	}
	if idx, err := w.f.GetSheetIndex(firstSheet); err == nil && idx >= 0 {
		w.f.SetActiveSheet(idx)
	}
	// ลบแผ่นเปล่าที่ excelize สร้างมาให้ตอนเริ่ม ไม่งั้นไฟล์จะมี Sheet1 ว่าง ๆ ติดมาด้วย
	if idx, err := w.f.GetSheetIndex("Sheet1"); err == nil && idx >= 0 && firstSheet != "Sheet1" {
		_ = w.f.DeleteSheet("Sheet1")
	}

	_, err := w.f.WriteTo(dst)
	return err
}

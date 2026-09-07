import { useRef, useState } from 'react'
import type { DragEvent } from 'react'
import { FileSpreadsheet, TriangleAlert, Upload } from 'lucide-react'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Field'
import { Icon } from '../../components/ui/Icon'
import { useUploadDataset } from './api'
import { ApiError } from '../../lib/api'
import { bytes } from '../../lib/format'

export function UploadPanel({ onUploaded }: { onUploaded: (id: string) => void }) {
  const upload = useUploadDataset()
  const inputRef = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [label, setLabel] = useState('')
  const [dragging, setDragging] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [duplicateOf, setDuplicateOf] = useState<string | null>(null)

  function pick(f: File | undefined) {
    if (!f) return
    setError(null)
    setDuplicateOf(null)
    if (!f.name.toLowerCase().endsWith('.xlsx')) {
      setError('รองรับเฉพาะไฟล์ .xlsx — หากไฟล์เป็น .xls ให้บันทึกใหม่เป็น Excel Workbook ก่อน')
      return
    }
    setFile(f)
  }

  function onDrop(e: DragEvent) {
    e.preventDefault()
    setDragging(false)
    pick(e.dataTransfer.files?.[0])
  }

  async function submit() {
    if (!file) return
    setError(null)
    try {
      const res = await upload.mutateAsync({ file, label: label.trim() })
      if (res.duplicate_of) {
        setDuplicateOf(res.duplicate_of.original_filename)
      }
      setFile(null)
      setLabel('')
      if (inputRef.current) inputRef.current.value = ''
      onUploaded(res.dataset.id)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'อัปโหลดไม่สำเร็จ กรุณาลองใหม่')
    }
  }

  return (
    <section className="flex flex-col gap-3 rounded-xl border border-line bg-paper p-5 shadow-panel">
      <div>
        <h2 className="font-display text-[15px] font-semibold">นำเข้าไฟล์ใหม่</h2>
        <p className="mt-0.5 text-[13px] text-muted">
          ทุกครั้งที่อัปโหลดระบบจะสร้างชุดข้อมูลใหม่ ไม่ทับของเดิม ประวัติจึงอยู่ครบ
        </p>
      </div>

      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className={`flex flex-col items-center gap-2 rounded-xl border border-dashed px-5 py-8 text-center transition
          ${dragging ? 'border-accent bg-accent-wash' : 'border-line bg-sunk/40'}`}
      >
        <Icon icon={file ? FileSpreadsheet : Upload} size={22} className="text-faint" />
        {file ? (
          <p className="text-[13px]">
            <span className="font-semibold">{file.name}</span>
            <span className="text-muted"> · {bytes(file.size)}</span>
          </p>
        ) : (
          <p className="text-[13px] text-muted">ลากไฟล์ .xlsx มาวางที่นี่ หรือเลือกจากเครื่อง</p>
        )}
        <input
          ref={inputRef}
          type="file"
          accept=".xlsx"
          className="hidden"
          onChange={(e) => pick(e.target.files?.[0])}
        />
        <Button type="button" onClick={() => inputRef.current?.click()}>
          เลือกไฟล์
        </Button>
      </div>

      <Input
        label="ชื่อกำกับ (ไม่บังคับ)"
        placeholder="เช่น ข้อมูลถึงเดือนสิงหาคม 2569"
        value={label}
        onChange={(e) => setLabel(e.target.value)}
      />

      {error && (
        <p role="alert" className="flex items-start gap-2 rounded-lg bg-bad-wash px-3 py-2.5 text-[13px] text-bad">
          <Icon icon={TriangleAlert} size={15} className="mt-0.5 shrink-0" />
          {error}
        </p>
      )}

      {duplicateOf && (
        <p className="flex items-start gap-2 rounded-lg bg-warn-wash px-3 py-2.5 text-[13px] text-accent-ink">
          <Icon icon={TriangleAlert} size={15} className="mt-0.5 shrink-0" />
          ไฟล์นี้มีเนื้อหาเหมือนกับ “{duplicateOf}” ที่เคยอัปโหลดไว้ทุกไบต์ — ระบบสร้างชุดใหม่ให้แล้ว
          หากไม่ได้ตั้งใจ ลบชุดที่ซ้ำออกได้
        </p>
      )}

      <Button
        variant="primary"
        icon={Upload}
        disabled={!file}
        loading={upload.isPending}
        onClick={submit}
      >
        อัปโหลดและนำเข้า
      </Button>
    </section>
  )
}

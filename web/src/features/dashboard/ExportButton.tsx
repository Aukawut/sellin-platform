import { useState } from 'react'
import { Download } from 'lucide-react'
import { Button } from '../../components/ui/Button'
import { downloadFromApi } from '../../lib/download'
import { ApiError } from '../../lib/api'
import { buildQuery } from '../../lib/api'
import type { Scope } from './useScope'

/**
 * ส่งออกข้อมูลตามขอบเขตที่กำลังดูอยู่
 *
 * ไฟล์ที่ได้มีทั้งแผ่นสรุปตัวชี้วัดและข้อมูลดิบ โดยใช้หัวคอลัมน์ชุดเดียวกับไฟล์ Template
 * จึงนำกลับเข้าระบบได้ทันทีโดยไม่ต้องแก้อะไร
 */
export function ExportButton({ scope, disabled }: { scope: Scope; disabled?: boolean }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function run() {
    setBusy(true)
    setError(null)
    try {
      const query = buildQuery({
        dataset_id: scope.dataset,
        dc: scope.dc,
        year: scope.year,
        month: scope.month,
      })
      await downloadFromApi(`/export${query}`, 'SalePerformance.xlsx')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'ส่งออกไม่สำเร็จ กรุณาลองใหม่')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <Button
        icon={Download}
        loading={busy}
        disabled={disabled || !scope.dataset}
        onClick={() => void run()}
        className="py-2"
        title="ส่งออกข้อมูลตามฟิลเตอร์ปัจจุบันเป็นไฟล์ Excel"
      >
        Export
      </Button>
      {error && (
        <span role="alert" className="max-w-[180px] text-[11px] text-bad">
          {error}
        </span>
      )}
    </div>
  )
}

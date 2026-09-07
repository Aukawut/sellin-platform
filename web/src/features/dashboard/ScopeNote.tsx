import type { ScopeInfo } from '../../types/api'
import { monthName } from '../../lib/format'

/**
 * บอกผู้ใช้ว่าตัวเลขที่เห็นครอบคลุมอะไร และเตือนเมื่อระบบเลือกเดือนให้เอง
 *
 * จำเป็นเพราะหน้า Stock และกราฟรายวันไม่ได้รวมทุกเดือนเมื่อเลือก "ทุกเดือน"
 * แต่ใช้เดือนล่าสุดที่มีข้อมูลแทน ถ้าไม่บอกไว้ผู้ใช้จะเข้าใจว่าเป็นยอดรวมทั้งปี
 */
export function ScopeNote({ scope, snapshot }: { scope: ScopeInfo; snapshot?: boolean }) {
  const parts = [
    scope.dc_name || 'ทุกศูนย์กระจายสินค้า',
    scope.year ? String(scope.year) : 'ทุกปี',
    scope.month ? monthName(scope.month) : 'ทุกเดือน',
  ]

  return (
    <span className="text-[12.5px] text-muted">
      ขอบเขตข้อมูล: {parts.join(' · ')}
      {snapshot && scope.has_data && (
        <>
          {' · '}
          <span className="text-accent-ink">
            สแนปช็อต {monthName(scope.effective_month)} {scope.effective_year}
            {scope.month_inferred && ' (เดือนล่าสุดที่มีข้อมูล)'}
          </span>
        </>
      )}
    </span>
  )
}

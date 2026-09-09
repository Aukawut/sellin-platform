import type { Kpi } from '../../types/api'
import { deltaText, formatKpi, num, thbShort } from '../../lib/format'

/**
 * ตัวเลขเด่นบนพื้นเรียบ คั่นด้วยเส้นบาง ไม่ใช่การ์ดที่มีขอบเงาเท่ากันทุกใบ
 * สีถูกใช้บอกสถานะเท่านั้น ไม่ได้ใช้บอกว่า "นี่คือการ์ด" ลำดับความสำคัญจึงอ่านออกจากการกวาดตา
 */
export function KpiRow({ kpis, labels }: { kpis: Kpi[]; labels: Record<string, string> }) {
  return (
    <div className="kpi-row grid grid-cols-1 divide-y divide-line rounded-xl border border-line bg-paper shadow-panel sm:grid-cols-2 sm:divide-y-0 lg:grid-cols-4">
      {kpis.map((k, i) => (
        <KpiTile key={k.key} kpi={k} label={labels[k.key] ?? k.key} first={i === 0} />
      ))}
    </div>
  )
}

function KpiTile({ kpi, label, first }: { kpi: Kpi; label: string; first: boolean }) {
  const tone =
    !kpi.available || !kpi.delta
      ? 'text-muted'
      : kpi.delta.direction === 'down'
        ? 'text-bad'
        : 'text-good'

  return (
    <div className={`kpi-tile px-5 py-5 sm:border-l sm:border-line ${first ? 'sm:border-l-0' : ''}`}>
      <p className="text-[12px] font-medium text-muted">{label}</p>
      <p
        className={`tnum mt-1 text-[30px] font-semibold leading-tight tracking-tight ${
          kpi.available ? 'text-ink' : 'text-faint'
        }`}
      >
        {formatKpi(kpi)}
      </p>
      <p className={`mt-1 min-h-[18px] text-[12px] ${tone}`}>
        {kpi.delta ? deltaText(kpi.delta) : secondaryText(kpi)}
      </p>
    </div>
  )
}

/** secondaryText ประกอบบรรทัดรองจากค่าที่ backend ส่งมา backend ไม่ได้ส่งข้อความสำเร็จรูป */
function secondaryText(kpi: Kpi): string {
  const s = kpi.secondary
  if (!kpi.available && !s) return 'ไม่มีข้อมูลให้คำนวณในขอบเขตนี้'
  if (!s) return ''

  switch (s.kind) {
    case 'avg_per_order':
      return `เฉลี่ย ${thbShort(s.value ?? 0)} ต่อออเดอร์`
    case 'active_total':
      return `เปิดใช้งานอยู่ จากทั้งหมด ${num(s.value ?? 0)} ศูนย์`
    case 'customer_status':
      return `สถานะ: ${s.text}`
    case 'cartons':
      return `รวม ${num(s.value ?? 0)} ลัง`
    case 'target_years':
      return `ข้อมูลเป้าหมายมีเฉพาะปี ${s.text}`
    case 'available':
      return `จากของพร้อมขาย ${num(s.value ?? 0)} ลัง`
    case 'months_counted':
      return `คำนวณจาก ${num(s.value ?? 0)} เดือนที่มีข้อมูล`
    case 'source':
      return 'นับจาก Sum Cartoon ของ Sell-In'
    case 'tier':
      return s.text === 'reached' ? 'ถึงเป้าหมายแล้ว' : 'ยังต่ำกว่าเป้าหมาย'
    default:
      return ''
  }
}

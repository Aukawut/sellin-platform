import type { Delta, Kpi } from '../types/api'

// ฝั่ง backend ส่งมาแต่ตัวเลขและความหมาย การประกอบเป็นข้อความไทยเกิดขึ้นที่นี่ที่เดียว
// dashboard เดิมประกอบข้อความไว้ในโค้ดคำนวณ จนแก้คำพูดทีต้องแตะตรรกะไปด้วย

const MONTH_TH = [
  '', 'ม.ค.', 'ก.พ.', 'มี.ค.', 'เม.ย.', 'พ.ค.', 'มิ.ย.',
  'ก.ค.', 'ส.ค.', 'ก.ย.', 'ต.ค.', 'พ.ย.', 'ธ.ค.',
]

export const monthName = (m: number) => MONTH_TH[m] ?? String(m)

export const thb = (n: number) => '฿' + Math.round(n).toLocaleString('en-US')

/** thbShort ย่อหน่วยให้กราฟและการ์ดอ่านได้ในพื้นที่แคบ */
export function thbShort(n: number) {
  const a = Math.abs(n)
  if (a >= 1e6) return '฿' + (n / 1e6).toFixed(2) + 'M'
  if (a >= 1e3) return '฿' + (n / 1e3).toFixed(1) + 'K'
  return '฿' + Math.round(n).toLocaleString('en-US')
}

export const num = (n: number) => Math.round(n).toLocaleString('en-US')

export const cartons = (n: number) =>
  n.toLocaleString('en-US', { maximumFractionDigits: 0 }) + ' ลัง'

export const pct = (n: number, digits = 1) => n.toFixed(digits) + '%'

export function bytes(n: number) {
  if (n >= 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n >= 1024) return (n / 1024).toFixed(0) + ' KB'
  return n + ' B'
}

export function formatKpi(k: Kpi): string {
  if (k.text) return k.text
  if (!k.available) return 'N/A'
  switch (k.format) {
    case 'thb': return thb(k.value)
    case 'percent': return (k.value >= 0 ? '' : '') + k.value.toFixed(1) + '%'
    case 'cartons': return cartons(k.value)
    case 'months': return k.value.toFixed(1) + ' เดือน'
    case 'number': return num(k.value)
    default: return String(k.value)
  }
}

/** deltaText ประกอบประโยคเทียบช่วงเวลา โดยแยกกรณี "ของใหม่" ออกจากการเติบโตปกติ */
export function deltaText(d: Delta): string {
  if (d.is_new) return 'ใหม่ในช่วงนี้'
  const arrow = d.direction === 'down' ? '▼' : '▲'
  const sign = d.pct >= 0 ? '+' : '−'
  const body = `${arrow} ${sign}${Math.abs(d.pct).toFixed(1)}%`
  if (d.base_year && d.prev_year) {
    return `${body} · ${thbShort(d.current)} (${d.base_year}) เทียบ ${thbShort(d.previous)} (${d.prev_year})`
  }
  return body
}

export function formatDate(iso: string) {
  const d = new Date(iso)
  return `${d.getDate()} ${monthName(d.getMonth() + 1)} ${d.getFullYear() + 543} ${String(
    d.getHours(),
  ).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

/** scopeLabel อธิบายขอบเขตของตัวเลขที่กำลังแสดงอยู่ ให้ผู้ใช้รู้ว่ากำลังดูอะไร */
export function scopeLabel(opts: {
  dcName?: string
  year: number | null
  month: number | null
}) {
  const parts = [opts.dcName || 'ทุกศูนย์กระจายสินค้า']
  parts.push(opts.year ? String(opts.year) : 'ทุกปี')
  parts.push(opts.month ? monthName(opts.month) : 'ทุกเดือน')
  return parts.join(' · ')
}

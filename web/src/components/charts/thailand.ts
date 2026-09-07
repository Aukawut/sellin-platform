import { PROVINCES, PROVINCE_ID_BY_NAME } from './thailand-provinces.generated'

export { THAILAND_VIEWBOX, LAKE_PATH, PROVINCES } from './thailand-provinces.generated'
export type { Province } from './thailand-provinces.generated'

const BY_ID = new Map(PROVINCES.map((p) => [p.id, p]))

/**
 * provinceId ค้นรหัสจังหวัดจากชื่อไทย
 *
 * ตัดคำว่า "จังหวัด" และช่องว่างออกก่อน เพราะไฟล์ต้นทางเขียนไม่สม่ำเสมอ
 * และตารางค้นหารองรับการสะกดที่พบจริงในไฟล์ เช่น "พะเยาว์" กับ "ฉะเฉิงเทรา"
 */
export function provinceId(name: string): string | undefined {
  const cleaned = name.replace(/\s+/g, '').replace(/^จังหวัด/, '')
  return PROVINCE_ID_BY_NAME[cleaned]
}

/**
 * dcAnchor หาจุดวางหมุดของศูนย์กระจายสินค้า จากค่าเฉลี่ยจุดกึ่งกลางของจังหวัดที่ศูนย์นั้นดูแล
 *
 * ใช้ขอบเขตจังหวัดจริงแทนพิกัดที่กำหนดไว้ตายตัวใน dashboard เดิม
 * ศูนย์ที่เพิ่มเข้ามาใหม่จึงปรากฏบนแผนที่ได้เองโดยไม่ต้องแก้โค้ด
 */
export function dcAnchor(provinces: string[] | undefined): [number, number] | null {
  const points = (provinces ?? [])
    .map(provinceId)
    .map((id) => (id ? BY_ID.get(id) : undefined))
    .filter(Boolean)
    .map((p) => [p!.cx, p!.cy] as [number, number])

  if (!points.length) return null
  const sum = points.reduce((a, p) => [a[0] + p[0], a[1] + p[1]] as [number, number], [0, 0])
  return [sum[0] / points.length, sum[1] / points.length]
}

/** provinceIdsFor คืนรหัสจังหวัดทั้งหมดที่ศูนย์กลุ่มนี้ครอบคลุม ใช้ระบายสีพื้นที่ให้เห็นความครอบคลุม */
export function provinceIdsFor(provinces: string[] | undefined): string[] {
  return (provinces ?? []).map(provinceId).filter((id): id is string => !!id)
}

export const REGION_COLOR: Record<string, string> = {
  เหนือ: 'var(--c1)',
  อีสาน: 'var(--c2)',
  กลาง: 'var(--c3)',
  ตะวันออก: 'var(--c4)',
  ตะวันตก: 'var(--c5)',
  ใต้: 'var(--c6)',
}

export const DC_COLORS = [
  'var(--c1)', 'var(--c2)', 'var(--c3)', 'var(--c4)',
  'var(--c5)', 'var(--c6)', 'var(--c7)', 'var(--c8)',
]

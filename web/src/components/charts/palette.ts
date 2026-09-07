/**
 * สีของกราฟอ้าง CSS variable ไม่ได้ใส่ค่าฮาร์ดโค้ด
 * SVG รับ var() ได้โดยตรง กราฟจึงเปลี่ยนสีตามธีมเองโดยไม่ต้องมี theme object ซ้อนอีกชั้น
 */
export const SERIES_COLORS = [
  'var(--c1)', 'var(--c2)', 'var(--c3)', 'var(--c4)',
  'var(--c5)', 'var(--c6)', 'var(--c7)', 'var(--c8)',
]

export const seriesColor = (i: number) => SERIES_COLORS[i % SERIES_COLORS.length]

/** สีของกลุ่มสินค้าหลักถูกล็อกไว้ เพื่อให้กลุ่มเดียวกันได้สีเดิมทุกหน้าและทุกกราฟ */
const FIXED: Record<string, string> = {
  'Cha Cha': 'var(--c2)',
  Bakery: 'var(--c3)',
  Mixnut: 'var(--c4)',
  'MK เส้นบุก': 'var(--c1)',
  Popcorn: 'var(--c6)',
  Other: 'var(--c5)',
  'Sun Flower Seed': 'var(--c2)',
  'Non-Sun Flower Seed': 'var(--c1)',
}

export function categoryColor(label: string, fallbackIndex: number) {
  return FIXED[label] ?? seriesColor(fallbackIndex)
}

export const AXIS = {
  stroke: 'var(--line)',
  tick: { fill: 'var(--faint)', fontSize: 11, fontFamily: 'var(--font-mono)' },
}

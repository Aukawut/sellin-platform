/**
 * สร้างไฟล์ข้อมูลแผนที่ประเทศไทยจากแพ็กเกจ @svg-maps/thailand
 *
 * ทำตอน build ไม่ใช่ตอน runtime เพื่อให้หน้าเว็บไม่ต้องพึ่งแพ็กเกจนั้นและไม่ต้องคำนวณจุดกึ่งกลางซ้ำทุกครั้ง
 * รันใหม่เมื่ออัปเกรดแพ็กเกจ (จากโฟลเดอร์ web):  node scripts/generate_thailand_map.mjs
 */
import { writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const map = require('@svg-maps/thailand').default

// ชื่อไทยของแต่ละจังหวัด แพ็กเกจต้นทางให้มาเฉพาะชื่ออังกฤษ
const TH = {
  bkk: 'กรุงเทพมหานคร', spk: 'สมุทรปราการ', nbi: 'นนทบุรี', pte: 'ปทุมธานี',
  aya: 'พระนครศรีอยุธยา', atg: 'อ่างทอง', lri: 'ลพบุรี', sbr: 'สิงห์บุรี',
  cnt: 'ชัยนาท', sri: 'สระบุรี', cbi: 'ชลบุรี', ryg: 'ระยอง', cti: 'จันทบุรี',
  trt: 'ตราด', cco: 'ฉะเชิงเทรา', pri: 'ปราจีนบุรี', nyk: 'นครนายก', skw: 'สระแก้ว',
  nma: 'นครราชสีมา', brm: 'บุรีรัมย์', srn: 'สุรินทร์', ssk: 'ศรีสะเกษ',
  ubn: 'อุบลราชธานี', yst: 'ยโสธร', cpm: 'ชัยภูมิ', acr: 'อำนาจเจริญ',
  bkn: 'บึงกาฬ', nbp: 'หนองบัวลำภู', kkn: 'ขอนแก่น', udn: 'อุดรธานี', lei: 'เลย',
  nki: 'หนองคาย', mkm: 'มหาสารคาม', ret: 'ร้อยเอ็ด', ksn: 'กาฬสินธุ์',
  snk: 'สกลนคร', npm: 'นครพนม', mdh: 'มุกดาหาร', cmi: 'เชียงใหม่', lpn: 'ลำพูน',
  lpg: 'ลำปาง', utd: 'อุตรดิตถ์', pre: 'แพร่', nan: 'น่าน', pyo: 'พะเยา',
  cri: 'เชียงราย', msn: 'แม่ฮ่องสอน', nsn: 'นครสวรรค์', uti: 'อุทัยธานี',
  kpt: 'กำแพงเพชร', tak: 'ตาก', sti: 'สุโขทัย', plk: 'พิษณุโลก', pct: 'พิจิตร',
  pnb: 'เพชรบูรณ์', rbr: 'ราชบุรี', kri: 'กาญจนบุรี', spb: 'สุพรรณบุรี',
  npt: 'นครปฐม', skn: 'สมุทรสาคร', skm: 'สมุทรสงคราม', pbi: 'เพชรบุรี',
  pkn: 'ประจวบคีรีขันธ์', nrt: 'นครศรีธรรมราช', kbi: 'กระบี่', pna: 'พังงา',
  pkt: 'ภูเก็ต', sni: 'สุราษฎร์ธานี', rng: 'ระนอง', cpn: 'ชุมพร', ska: 'สงขลา',
  stn: 'สตูล', trg: 'ตรัง', plg: 'พัทลุง', ptn: 'ปัตตานี', yla: 'ยะลา',
  nwt: 'นราธิวาส',
}

// การสะกดที่พบจริงในไฟล์ Excel ซึ่งต่างจากชื่อทางการ
// เก็บไว้ที่นี่แทนการแก้ข้อมูลต้นทาง เพราะไฟล์ที่อัปโหลดครั้งหน้าก็จะสะกดแบบเดิมอีก
const ALIAS = {
  'ฉะเฉิงเทรา': 'cco',
  'พะเยาว์': 'pyo',
  'สุราษฎร์': 'sni',
  'อยุธยา': 'aya',
  'แม่ฮองสอน': 'msn',
  'กรุงเทพ': 'bkk',
  'กรุงเทพฯ': 'bkk',
  'อยุธยา ': 'aya',
}

/**
 * parsePath อ่าน path ที่ใช้เฉพาะคำสั่ง m (relative) และ z ซึ่งเป็นรูปแบบที่แพ็กเกจนี้ใช้
 * คืนจุดบนเส้นขอบทั้งหมด เพื่อนำไปหาจุดกึ่งกลางสำหรับวางหมุด
 */
function parsePoints(d) {
  const tokens = d.match(/[mz]|-?[\d.]+/gi) || []
  const points = []
  let x = 0, y = 0, startX = 0, startY = 0
  let i = 0
  let first = true

  while (i < tokens.length) {
    const t = tokens[i]
    if (t === 'm' || t === 'M') {
      i++
      const dx = parseFloat(tokens[i++]), dy = parseFloat(tokens[i++])
      // ตาม SVG spec คำสั่ง m ตัวแรกสุดของ path ถือเป็นพิกัดสัมบูรณ์
      if (first) { x = dx; y = dy; first = false } else { x += dx; y += dy }
      startX = x; startY = y
      points.push([x, y])
      // คู่ตัวเลขที่ตามมาหลัง m คือ lineto แบบสัมพัทธ์
      while (i < tokens.length && !/[a-z]/i.test(tokens[i])) {
        x += parseFloat(tokens[i++]); y += parseFloat(tokens[i++])
        points.push([x, y])
      }
    } else if (t === 'z' || t === 'Z') {
      x = startX; y = startY
      i++
    } else {
      i++
    }
  }
  return points
}

/** ลดจำนวนทศนิยมลงเหลือหนึ่งตำแหน่ง — ที่ viewBox 560×1025 ละเอียดกว่านี้ตาไม่เห็นความต่าง */
const trimPrecision = (d) => d.replace(/-?\d+\.\d+/g, (n) => String(Math.round(parseFloat(n) * 10) / 10))

const provinces = []
let waterPath = ''

for (const loc of map.locations) {
  if (loc.id === 'lksg') {
    waterPath = trimPrecision(loc.path)
    continue
  }
  const th = TH[loc.id]
  if (!th) {
    console.warn(`ไม่มีชื่อไทยของ ${loc.id} (${loc.name}) — ข้ามไป`)
    continue
  }
  const pts = parsePoints(loc.path)
  const cx = pts.reduce((a, p) => a + p[0], 0) / pts.length
  const cy = pts.reduce((a, p) => a + p[1], 0) / pts.length

  provinces.push({
    id: loc.id,
    en: loc.name,
    th,
    cx: Math.round(cx * 10) / 10,
    cy: Math.round(cy * 10) / 10,
    d: trimPrecision(loc.path),
  })
}

const lookup = {}
for (const p of provinces) lookup[p.th] = p.id
for (const [name, id] of Object.entries(ALIAS)) lookup[name.trim()] = id

const out = `// ไฟล์นี้ถูกสร้างโดย scripts/generate_thailand_map.mjs — ห้ามแก้ด้วยมือ
// ที่มาข้อมูล: @svg-maps/thailand v${require('@svg-maps/thailand/package.json').version} (สัญญาอนุญาต CC-BY-4.0)
// รันใหม่เมื่ออัปเกรดแพ็กเกจ: node scripts/generate_thailand_map.mjs

export const THAILAND_VIEWBOX = '${map.viewBox}'

export interface Province {
  id: string
  /** ชื่ออังกฤษจากแพ็กเกจต้นทาง */
  en: string
  /** ชื่อไทยทางการ */
  th: string
  /** จุดกึ่งกลางโดยประมาณ ใช้วางหมุดของศูนย์กระจายสินค้า */
  cx: number
  cy: number
  d: string
}

export const PROVINCES: Province[] = ${JSON.stringify(provinces, null, 0)}

/** ทะเลสาบสงขลา วาดทับเพื่อไม่ให้ดูเหมือนเป็นแผ่นดิน */
export const LAKE_PATH = ${JSON.stringify(waterPath)}

/**
 * ค้นรหัสจังหวัดจากชื่อไทย รองรับการสะกดที่พบจริงในไฟล์ Excel ด้วย
 * เช่น "ฉะเฉิงเทรา" และ "พะเยาว์" ซึ่งสะกดต่างจากชื่อทางการ
 */
export const PROVINCE_ID_BY_NAME: Record<string, string> = ${JSON.stringify(lookup, null, 0)}
`

const target = 'src/components/charts/thailand-provinces.generated.ts'
writeFileSync(target, out)

console.log(`เขียน ${target}`)
console.log(`  จังหวัด ${provinces.length} รายการ · ชื่อที่ค้นได้ ${Object.keys(lookup).length} ชื่อ`)
console.log(`  ขนาดไฟล์ ${(out.length / 1024).toFixed(0)} KB (ต้นฉบับ ${(map.locations.reduce((a, l) => a + l.path.length, 0) / 1024).toFixed(0)} KB)`)

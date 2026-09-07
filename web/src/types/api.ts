// รูปร่างของคำตอบจาก API ตรงกับ struct ฝั่ง Go
// backend ส่งตัวเลขและความหมาย ไม่ส่งข้อความที่จัดรูปแบบแล้ว การประกอบประโยคเป็นหน้าที่ของที่นี่

export type Role = 'admin' | 'viewer'

export interface User {
  id: string
  email: string
  display_name: string
  role: Role
  is_active: boolean
  created_at: string
}

export interface LoginResponse {
  user: User
  access_token: string
  expires_at: string
}

export type DatasetStatus = 'processing' | 'ready' | 'failed'

export interface Dataset {
  id: string
  owner_id: string
  owner_name: string
  original_filename: string
  label?: string
  size_bytes: number
  status: DatasetStatus
  row_counts: Record<string, number>
  year_min?: number
  year_max?: number
  error_message?: string
  uploaded_at: string
  finished_at?: string
  deleted_at?: string
  can_delete: boolean
}

export type Severity = 'info' | 'warning' | 'blocking'

export interface ImportIssue {
  sheet: string
  row_no?: number
  severity: Severity
  field?: string
  message: string
}

export interface ImportSummary {
  dataset_id: string
  status: DatasetStatus
  sheets_read: string[]
  row_counts: Record<string, number>
  issue_counts: { info: number; warning: number; blocking: number }
  issues?: ImportIssue[]
}

export interface UploadResult {
  dataset: Dataset
  duplicate_of?: Dataset
}

// ---------- dashboard ----------

export type KpiFormat = 'thb' | 'number' | 'percent' | 'cartons' | 'months' | 'text'
export type Direction = 'up' | 'down' | 'flat'

export interface Delta {
  pct: number
  direction: Direction
  current: number
  previous: number
  base_year?: number
  prev_year?: number
  is_new?: boolean
}

export interface Secondary {
  kind: string
  value?: number
  text?: string
}

export interface Kpi {
  key: string
  value: number
  text?: string
  format: KpiFormat
  available: boolean
  delta?: Delta
  secondary?: Secondary
}

export interface Series {
  key: string
  name: string
  kind?: 'line' | 'bar' | 'dashed'
  color?: string
  points: (number | null)[]
}

export interface Chart {
  labels: string[]
  series: Series[]
  tag?: string
  /** เส้นอ้างอิงแนวนอน เช่นค่าเฉลี่ยในกราฟยอดขายรายวัน */
  reference?: number
}

export interface Slice {
  label: string
  value: number
  pct: number
  yoy?: Delta
}

export interface RankEntry {
  rank: number
  code: string
  name: string
  value: number
  /** สัดส่วนเทียบอันดับหนึ่ง ใช้กำหนดความยาวแถบ */
  share: number
}

export interface MapNode {
  code: string
  name: string
  value: number
  pct: number
  provinces?: string[]
  status?: string
  yoy?: Delta
}

export interface ScopeInfo {
  dataset_id: string
  dc: string
  dc_name: string
  year: number | null
  month: number | null
  effective_year: number
  effective_month: number
  /** true = ระบบเลือกเดือนให้เอง ไม่ใช่ผู้ใช้เลือก */
  month_inferred: boolean
  year_inferred: boolean
  has_data: boolean
}

export interface SellInPage {
  scope: ScopeInfo
  kpis: Kpi[]
  trend: Chart
  daily: Chart
  tracking: Chart
  donut_group: Slice[]
  donut_category: Slice[]
  ranking: RankEntry[]
  network: {
    nodes: MapNode[]
    active_count: number
    total_count: number
    province_count: number
  }
}

export interface SellOutProduct {
  rank: number
  code: string
  description: string
  group_name: string
  category: string
  cartons: number
  revenue: number
  share: number
}

export interface SellOutPage {
  scope: ScopeInfo
  kpis: Kpi[]
  trend: Chart
  year_compare: Chart
  donut_group: Slice[]
  donut_category: Slice[]
  products: SellOutProduct[]
  ranking: RankEntry[]
}

export type StockStatus = 'ok' | 'warn' | 'risk'

export interface StockProduct {
  code: string
  description: string
  group_name: string
  beginning: number
  sell_in: number
  sell_out: number
  ending: number
  cover_months: number | null
  status: StockStatus
  label: string
  product_status: string
}

export interface StockPage {
  scope: ScopeInfo
  kpis: Kpi[]
  trend: Chart
  flow: Chart
  donut_group: Slice[]
  donut_category: Slice[]
  products: StockProduct[]
  alert_count: number
  total_products: number
}

export type PlanTier = 'reached' | 'near' | 'behind' | 'no_target'

export interface PlanRow {
  code: string
  name: string
  target: number
  actual: number
  achievement: number | null
  gap: number
  has_target: boolean
  tier: PlanTier
}

export interface PlanningPage {
  scope: ScopeInfo
  kpis: Kpi[]
  trend: Chart
  category_bar: Chart
  group_trend: Chart
  regions: { nodes: MapNode[]; total: number; best?: string }
  plan: PlanRow[]
}

export interface FilterOptions {
  customers: {
    code: string
    name: string
    status: string
    region?: string
    provinces?: string[]
  }[]
  years: number[]
  months: number[]
  product_groups: { sellout: string[]; stock: string[] }
  stock_statuses: string[]
}

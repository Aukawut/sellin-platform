import { LibraryBig, PackageX } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'
import { Panel, SectionHead } from '../../components/charts/Panel'
import { SeriesChart } from '../../components/charts/SeriesChart'
import { Donut } from '../../components/charts/Donut'
import { DataTable } from '../../components/data/Table'
import type { Column } from '../../components/data/Table'
import { KpiRow } from '../../components/data/KpiTile'
import { Pill } from '../../components/ui/Pill'
import { Select } from '../../components/ui/Field'
import { EmptyState, ErrorState, Spinner } from '../../components/ui/States'
import { ScopeBar } from './ScopeBar'
import { ScopeNote } from './ScopeNote'
import { useFilters, useScope } from './useScope'
import { useStock } from './api'
import { monthName, num } from '../../lib/format'
import type { StockProduct } from '../../types/api'

const LABELS: Record<string, string> = {
  beginning_stock: 'สต๊อกต้นเดือน',
  stock_in: 'รับเข้า (Sell-In)',
  sell_through: 'Sell-Through Rate',
  stock_cover: 'Stock Cover',
}

const STRIPE: Record<string, string> = {
  risk: 'var(--bad)',
  warn: 'var(--accent)',
  ok: 'transparent',
}

const columns: Column<StockProduct>[] = [
  {
    key: 'product',
    header: 'สินค้า',
    render: (p) => (
      <>
        <span className="font-medium">{p.description || p.code}</span>
        <span className="ml-2 text-[11px] text-faint">{p.code}</span>
      </>
    ),
  },
  { key: 'group', header: 'กลุ่ม', width: '118px', render: (p) => <span className="text-muted">{p.group_name}</span> },
  { key: 'beg', header: 'ต้นเดือน', align: 'right', width: '92px', render: (p) => num(p.beginning) },
  { key: 'in', header: 'รับเข้า', align: 'right', width: '86px', render: (p) => num(p.sell_in) },
  { key: 'out', header: 'ขายออก', align: 'right', width: '86px', render: (p) => num(p.sell_out) },
  {
    key: 'end',
    header: 'คงเหลือ',
    align: 'right',
    width: '92px',
    render: (p) => <span className="font-semibold">{num(p.ending)}</span>,
  },
  {
    key: 'cover',
    header: 'Cover (เดือน)',
    align: 'right',
    width: '110px',
    render: (p) => (p.cover_months === null ? <span className="text-faint">—</span> : p.cover_months.toFixed(1)),
  },
  {
    key: 'status',
    header: 'สถานะสต๊อก',
    width: '128px',
    render: (p) => (
      <Pill tone={p.status === 'risk' ? 'bad' : p.status === 'warn' ? 'warn' : 'good'}>{p.label}</Pill>
    ),
  },
  {
    key: 'product_status',
    header: 'สถานะสินค้า',
    width: '116px',
    render: (p) => (
      <Pill tone={p.product_status === 'Non-Active' ? 'bad' : p.product_status === 'Active' ? 'good' : 'neutral'}>
        {p.product_status}
      </Pill>
    ),
  },
]

export function StockDashboard() {
  const { scope, query } = useScope()
  const [params, setParams] = useSearchParams()
  const group = params.get('group') ?? 'ALL'
  const status = params.get('status') ?? 'ALL'
  const filters = useFilters(scope.dataset)
  const page = useStock(query, !!scope.dataset, group, status)

  function setParam(key: string, v: string) {
    const next = new URLSearchParams(params)
    if (v === 'ALL') next.delete(key)
    else next.set(key, v)
    setParams(next, { replace: true })
  }

  return (
    <>
      <ScopeBar
        title="Stock Inventory"
        subtitle="ติดตามสต๊อกคงเหลือ อัตราขายออก และรายการที่ต้องเฝ้าระวัง — ทุกตัวเลขเป็นค่า ณ เดือนเดียว"
        extra={
          <>
            <Select
              label="กลุ่มสินค้า"
              value={group}
              onChange={(e) => setParam('group', e.target.value)}
              className="w-[146px] py-2 text-[13px]"
            >
              <option value="ALL">ทุกกลุ่ม</option>
              {filters.data?.product_groups.stock.map((g) => (
                <option key={g} value={g}>{g}</option>
              ))}
            </Select>
            <Select
              label="สถานะสต๊อก"
              value={status}
              onChange={(e) => setParam('status', e.target.value)}
              className="w-[148px] py-2 text-[13px]"
            >
              <option value="ALL">ทุกสถานะ</option>
              {filters.data?.stock_statuses.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </Select>
          </>
        }
      />

      <main className="mx-auto max-w-[1400px] px-5 pb-16">
        {!scope.dataset && (
          <div className="pt-8">
            <EmptyState icon={LibraryBig} title="เลือกชุดข้อมูลก่อนเริ่ม" detail="เลือกไฟล์จากช่อง “ชุดข้อมูล” ด้านบน" />
          </div>
        )}
        {page.isPending && scope.dataset && <Spinner label="กำลังคำนวณตัวเลข" />}
        {page.isError && (
          <div className="pt-6">
            <ErrorState message={(page.error as Error).message} retry={() => page.refetch()} />
          </div>
        )}

        {page.data && !page.data.scope.has_data && (
          <div className="pt-8">
            <EmptyState
              icon={PackageX}
              title="ไม่มีข้อมูลสต๊อกในขอบเขตที่เลือก"
              detail={`ชุดข้อมูลนี้ไม่มีแถวสต๊อกของ ${
                page.data.scope.dc_name || 'ศูนย์ที่เลือก'
              } ในปีที่เลือก — ลองเปลี่ยนศูนย์หรือปี`}
            />
          </div>
        )}

        {page.data?.scope.has_data && (
          <>
            <SectionHead title="ภาพรวมสต๊อก" note={<ScopeNote scope={page.data.scope} snapshot />} />
            <KpiRow kpis={page.data.kpis} labels={LABELS} />

            <SectionHead title="แนวโน้มสต๊อกและการเคลื่อนไหว" />
            <div className="grid gap-4 lg:grid-cols-2">
              <Panel title="สต๊อกต้นเดือนเทียบปลายเดือน" tag={page.data.trend.tag || 'ทุกศูนย์'}>
                <SeriesChart
                  chart={page.data.trend}
                  height={250}
                  format={(v) => `${num(v)} ลัง`}
                  axisFormat={num}
                  labelFormat={(l) => monthName(Number(l))}
                />
              </Panel>
              <Panel title="รับเข้าเทียบขายออก" tag={page.data.flow.tag || 'ทุกศูนย์'}>
                <SeriesChart
                  chart={page.data.flow}
                  height={250}
                  format={(v) => `${num(v)} ลัง`}
                  axisFormat={num}
                  labelFormat={(l) => monthName(Number(l))}
                />
              </Panel>
            </div>

            <SectionHead title="สัดส่วนสต๊อกคงเหลือ" />
            <div className="grid gap-4 lg:grid-cols-2">
              <Panel title="สัดส่วนตาม Product Group">
                <Donut slices={page.data.donut_group} format={(v) => `${num(v)} ลัง`} />
              </Panel>
              <Panel title="สัดส่วนตาม Product Category">
                <Donut slices={page.data.donut_category} format={(v) => `${num(v)} ลัง`} />
              </Panel>
            </div>

            <SectionHead
              title="สถานะสต๊อกรายสินค้า"
              note={
                <>
                  แสดง {page.data.products.length} จาก {page.data.total_products} รายการ ·{' '}
                  <span className={page.data.alert_count ? 'text-accent-ink' : ''}>
                    ต้องเฝ้าระวัง {page.data.alert_count} รายการ
                  </span>
                </>
              }
            />
            <Panel>
              <DataTable
                columns={columns}
                rows={page.data.products}
                rowKey={(p) => p.code}
                stripe={(p) => STRIPE[p.status]}
                empty={
                  status !== 'ALL'
                    ? `ไม่มีสินค้าที่มีสถานะ “${status}” ในเดือนนี้ — เปลี่ยนสถานะเป็น “ทุกสถานะ” เพื่อดูทั้งหมด`
                    : undefined
                }
              />
            </Panel>
          </>
        )}
      </main>
    </>
  )
}

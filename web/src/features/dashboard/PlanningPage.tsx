import { LibraryBig } from 'lucide-react'
import { Panel, SectionHead } from '../../components/charts/Panel'
import { CategoryBars, SeriesChart } from '../../components/charts/SeriesChart'
import { RegionMap } from '../../components/charts/ThailandMap'
import { DataTable } from '../../components/data/Table'
import type { Column } from '../../components/data/Table'
import { KpiRow } from '../../components/data/KpiTile'
import { Pill } from '../../components/ui/Pill'
import { EmptyState, ErrorState, Spinner } from '../../components/ui/States'
import { ScopeBar } from './ScopeBar'
import { ScopeNote } from './ScopeNote'
import { useFilters, useScope } from './useScope'
import { usePlanning } from './api'
import { monthName, pct, thb, thbShort } from '../../lib/format'
import type { PlanRow } from '../../types/api'

const LABELS: Record<string, string> = {
  actual: 'ยอดขายจริง (Sell-Out)',
  target: 'Target Sale',
  achievement: '% Achievement',
  forecast: 'คาดการณ์ทั้งปี',
}

const TIER_TEXT: Record<PlanRow['tier'], string> = {
  reached: 'ถึงเป้า',
  near: 'ใกล้เป้า',
  behind: 'ต่ำกว่าเป้า',
  no_target: 'ไม่มีเป้า',
}

const columns: Column<PlanRow>[] = [
  { key: 'name', header: 'ศูนย์กระจายสินค้า', render: (r) => <span className="font-medium">{r.name}</span> },
  {
    key: 'target',
    header: 'Target',
    align: 'right',
    width: '128px',
    render: (r) => (r.has_target ? thb(r.target) : <span className="text-faint">—</span>),
  },
  { key: 'actual', header: 'ยอดจริง', align: 'right', width: '128px', render: (r) => thb(r.actual) },
  {
    key: 'ach',
    header: '% Achievement',
    align: 'right',
    width: '116px',
    render: (r) =>
      r.achievement === null ? (
        <span className="text-faint">—</span>
      ) : (
        <span className="font-semibold">{pct(r.achievement)}</span>
      ),
  },
  {
    key: 'gap',
    header: 'ส่วนต่าง',
    align: 'right',
    width: '128px',
    render: (r) =>
      r.has_target ? (
        <span className={r.gap > 0 ? 'text-bad' : 'text-good'}>{thb(Math.abs(r.gap))}</span>
      ) : (
        <span className="text-faint">—</span>
      ),
  },
  {
    key: 'tier',
    header: 'สถานะ',
    width: '106px',
    render: (r) => (
      <Pill
        tone={
          r.tier === 'reached' ? 'good' : r.tier === 'near' ? 'warn' : r.tier === 'behind' ? 'bad' : 'neutral'
        }
      >
        {TIER_TEXT[r.tier]}
      </Pill>
    ),
  },
]

export function PlanningDashboard() {
  const { scope, query } = useScope()
  const filters = useFilters(scope.dataset)
  const page = usePlanning(query, !!scope.dataset)

  return (
    <>
      <ScopeBar
        title="Sale Analysis Planning"
        subtitle="วิเคราะห์ผลงานเทียบเป้าหมาย คาดการณ์สิ้นปี และวางแผนรายศูนย์"
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

        {page.data && (
          <>
            <SectionHead title="ภาพรวมการวางแผน" note={<ScopeNote scope={page.data.scope} />} />
            <KpiRow kpis={page.data.kpis} labels={LABELS} />

            <SectionHead
              title="ยอดจริง เป้าหมาย และคาดการณ์"
              note="เส้นประคือค่าเฉลี่ยของเดือนที่มีข้อมูล คูณด้วยเดือนที่เหลือ"
            />
            <Panel title="รายเดือน" tag={page.data.trend.tag || 'ทุกศูนย์'}>
              <SeriesChart
                chart={page.data.trend}
                height={300}
                format={thbShort}
                labelFormat={(l) => monthName(Number(l))}
              />
            </Panel>

            <SectionHead title="วิเคราะห์ตามหมวดและกลุ่มสินค้า" />
            <div className="grid gap-4 lg:grid-cols-2">
              <Panel title="Target เทียบยอดจริง แยกตาม Product Category">
                <CategoryBars chart={page.data.category_bar} height={270} format={thbShort} />
              </Panel>
              <Panel title="เทรนยอดขายแยกตาม Product Group" tag={page.data.group_trend.tag || 'ทุกศูนย์'}>
                <SeriesChart
                  chart={page.data.group_trend}
                  height={270}
                  format={thbShort}
                  labelFormat={(l) => monthName(Number(l))}
                />
              </Panel>
            </div>

            <SectionHead
              title="สัดส่วนยอดขายตามภาค"
              note={
                page.data.regions.best
                  ? `ภาคที่ทำได้สูงสุดคือภาค${page.data.regions.best} · รวม ${thbShort(page.data.regions.total)}`
                  : undefined
              }
            />
            <Panel title="แผนที่สัดส่วนรายภาค" tag="ขนาดหมุดแปรตามยอดขาย">
              <div className="flex flex-col items-center gap-8 p-4 md:flex-row md:justify-center">
                <div className="w-full max-w-[360px] shrink-0">
                  <RegionMap nodes={page.data.regions.nodes} customers={filters.data?.customers ?? []} />
                </div>
                <ul className="flex w-full max-w-[520px] flex-col justify-center gap-2">
                  {page.data.regions.nodes.map((n) => (
                    <li key={n.code} className="flex items-center gap-2.5 text-[13px]">
                      <span className="min-w-0 flex-1 truncate">ภาค{n.name}</span>
                      <span className="tnum w-14 text-right text-muted">{n.pct.toFixed(1)}%</span>
                      <span className="tnum w-24 text-right font-semibold">{thbShort(n.value)}</span>
                      <span
                        className={`tnum w-16 text-right text-[11px] ${
                          !n.yoy ? 'text-faint' : n.yoy.is_new || n.yoy.direction === 'up' ? 'text-good' : 'text-bad'
                        }`}
                      >
                        {!n.yoy
                          ? 'N/A'
                          : n.yoy.is_new
                            ? '▲ ใหม่'
                            : `${n.yoy.direction === 'up' ? '▲' : '▼'} ${Math.abs(n.yoy.pct).toFixed(1)}%`}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            </Panel>

            <SectionHead
              title="แผนปฏิบัติการรายศูนย์"
              note="เรียงตาม % Achievement · ศูนย์ที่ยังไม่มีเป้าหมายอยู่ท้ายตาราง"
            />
            <Panel>
              <DataTable columns={columns} rows={page.data.plan} rowKey={(r) => r.code} />
            </Panel>
          </>
        )}
      </main>
    </>
  )
}

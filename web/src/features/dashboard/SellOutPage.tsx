import { LibraryBig } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'
import { Panel, SectionHead } from '../../components/charts/Panel'
import { SeriesChart } from '../../components/charts/SeriesChart'
import { Donut } from '../../components/charts/Donut'
import { DataTable } from '../../components/data/Table'
import type { Column } from '../../components/data/Table'
import { KpiRow } from '../../components/data/KpiTile'
import { RankList } from '../../components/data/RankList'
import { Select } from '../../components/ui/Field'
import { EmptyState, ErrorState, Spinner } from '../../components/ui/States'
import { ScopeBar } from './ScopeBar'
import { ScopeNote } from './ScopeNote'
import { useFilters, useScope } from './useScope'
import { useSellOut } from './api'
import { monthName, num, pct, thb, thbShort } from '../../lib/format'
import type { SellOutProduct } from '../../types/api'

const LABELS: Record<string, string> = {
  target: 'Target Sale',
  revenue: 'Revenue',
  achievement: '% เทียบเป้าหมาย',
  growth: 'เทียบยอดปีก่อนหน้า',
}

const columns: Column<SellOutProduct>[] = [
  { key: 'rank', header: '#', width: '52px', render: (p) => <span className="tnum text-faint">{p.rank}</span> },
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
  { key: 'group', header: 'กลุ่ม', width: '120px', render: (p) => <span className="text-muted">{p.group_name}</span> },
  { key: 'cat', header: 'หมวด', width: '150px', render: (p) => <span className="text-muted">{p.category}</span> },
  { key: 'cartons', header: 'ลัง', align: 'right', width: '90px', render: (p) => num(p.cartons) },
  { key: 'revenue', header: 'Revenue', align: 'right', width: '130px', render: (p) => thb(p.revenue) },
  {
    key: 'share',
    header: 'สัดส่วน',
    align: 'right',
    width: '86px',
    render: (p) => <span className="font-semibold">{pct(p.share)}</span>,
  },
]

export function SellOutDashboard() {
  const { scope, query } = useScope()
  const [params, setParams] = useSearchParams()
  const group = params.get('group') ?? 'ALL'
  const filters = useFilters(scope.dataset)
  const page = useSellOut(query, !!scope.dataset, group)

  function setGroup(v: string) {
    const next = new URLSearchParams(params)
    if (v === 'ALL') next.delete('group')
    else next.set('group', v)
    setParams(next, { replace: true })
  }

  return (
    <>
      <ScopeBar
        title="Sell-Out Performance"
        subtitle="ยอดขายออกจากศูนย์กระจายสินค้า เทียบเป้าหมายและปีก่อนหน้า"
        extra={
          <Select
            label="กลุ่มสินค้า"
            value={group}
            onChange={(e) => setGroup(e.target.value)}
            className="w-[150px] py-2 text-[13px]"
          >
            <option value="ALL">ทุกกลุ่ม</option>
            {filters.data?.product_groups.sellout.map((g) => (
              <option key={g} value={g}>{g}</option>
            ))}
          </Select>
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

        {page.data && (
          <>
            <SectionHead title="ภาพรวม Sell-Out" note={<ScopeNote scope={page.data.scope} />} />
            <KpiRow kpis={page.data.kpis} labels={LABELS} />

            <SectionHead title="เทียบเป้าหมายและปีก่อนหน้า" />
            <div className="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
              <Panel title="Target เทียบ Revenue รายเดือน" tag={page.data.trend.tag || 'ทุกศูนย์'}>
                <SeriesChart
                  chart={page.data.trend}
                  height={290}
                  format={thbShort}
                  labelFormat={(l) => monthName(Number(l))}
                />
              </Panel>
              <Panel title="อันดับศูนย์กระจายสินค้า" tag="ตามยอด Revenue">
                <div className="max-h-[290px] overflow-y-auto">
                  <RankList entries={page.data.ranking} highlight={scope.dc} />
                </div>
              </Panel>
            </div>

            <div className="mt-4">
              <Panel title="Revenue เทียบสองปีล่าสุด" tag={page.data.year_compare.tag || 'ทุกศูนย์'}>
                <SeriesChart
                  chart={page.data.year_compare}
                  height={250}
                  format={thbShort}
                  labelFormat={(l) => monthName(Number(l))}
                />
              </Panel>
            </div>

            <SectionHead title="สัดส่วนสินค้าที่ขายออก" note="ตัวเลขขวาสุดคือการเทียบกับปีก่อนหน้า" />
            <div className="grid gap-4 lg:grid-cols-2">
              <Panel title="สัดส่วนตาม Product Group">
                <Donut slices={page.data.donut_group} format={(v) => `${num(v)} ลัง`} showYoY />
              </Panel>
              <Panel title="สัดส่วนตาม Product Category">
                <Donut slices={page.data.donut_category} format={(v) => `${num(v)} ลัง`} showYoY />
              </Panel>
            </div>

            <SectionHead
              title="ตารางสรุปสินค้าที่ขายออก"
              note={`${page.data.products.length} รายการ${group !== 'ALL' ? ` · กลุ่ม ${group}` : ''} · เรียงตาม Revenue`}
            />
            <Panel>
              <DataTable
                columns={columns}
                rows={page.data.products}
                rowKey={(p) => p.code}
                empty={
                  group !== 'ALL'
                    ? `ไม่มีสินค้ากลุ่ม “${group}” ในขอบเขตที่เลือก — ลองเปลี่ยนกลุ่มสินค้าหรือล้างฟิลเตอร์`
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

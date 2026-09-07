import { Building2, LibraryBig } from 'lucide-react'
import { Panel, SectionHead } from '../../components/charts/Panel'
import { SeriesChart } from '../../components/charts/SeriesChart'
import { Donut } from '../../components/charts/Donut'
import { NetworkMap } from '../../components/charts/ThailandMap'
import { KpiRow } from '../../components/data/KpiTile'
import { RankList } from '../../components/data/RankList'
import { EmptyState, ErrorState, Spinner } from '../../components/ui/States'
import { ScopeBar } from './ScopeBar'
import { ScopeNote } from './ScopeNote'
import { useScope } from './useScope'
import { useSellIn } from './api'
import { monthName, num, thbShort } from '../../lib/format'

const LABELS: Record<string, string> = {
  revenue: 'Revenue',
  orders: 'จำนวนออเดอร์',
  active_store: 'ศูนย์กระจายสินค้า',
  growth: 'เทียบยอดปีก่อนหน้า',
}

export function SellInDashboard() {
  const { scope, query } = useScope()
  const page = useSellIn(query, !!scope.dataset)

  return (
    <>
      <ScopeBar
        title="Sell-In Performance"
        subtitle="ภาพรวมยอดสั่งซื้อสินค้าเข้าศูนย์กระจายสินค้า — ความต่อเนื่อง แนวโน้ม และสัดส่วน SKU"
      />

      <main className="mx-auto max-w-[1400px] px-5 pb-16">
        {!scope.dataset && (
          <div className="pt-8">
            <EmptyState
              icon={LibraryBig}
              title="เลือกชุดข้อมูลก่อนเริ่ม"
              detail="เลือกไฟล์จากช่อง “ชุดข้อมูล” ด้านบน ระบบจะสร้าง dashboard จากไฟล์ที่เลือก"
            />
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
            <SectionHead title="ภาพรวม Sell-In" note={<ScopeNote scope={page.data.scope} />} />
            <KpiRow kpis={page.data.kpis} labels={LABELS} />

            <SectionHead title="แนวโน้มยอดขาย" />
            <div className="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
              <Panel title="Revenue รายเดือน เทียบสองปีล่าสุด" tag={page.data.trend.tag || 'ทุกศูนย์'}>
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

            <SectionHead title="รายละเอียดยอดสั่งซื้อ" />
            <div className="grid gap-4 lg:grid-cols-2">
              <Panel
                title="ยอดขายรายวันภายในเดือน"
                tag={
                  page.data.scope.has_data
                    ? `${monthName(page.data.scope.effective_month)} ${page.data.scope.effective_year}${
                        page.data.scope.month_inferred ? ' (ล่าสุด)' : ''
                      }`
                    : '—'
                }
              >
                <SeriesChart chart={page.data.daily} height={250} format={thbShort} />
              </Panel>
              <Panel title="ติดตามยอดสั่งซื้อ Non-Sun Flower Seed" tag="Target เทียบยอดจริง">
                <SeriesChart chart={page.data.tracking} height={250} format={num} />
              </Panel>
            </div>

            <SectionHead title="สัดส่วนสินค้าและเครือข่ายศูนย์กระจายสินค้า" />
            <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1.2fr)]">
              <Panel title="สัดส่วนตาม Product Group">
                <Donut slices={page.data.donut_group} format={(v) => `${num(v)} ลัง`} />
              </Panel>
              <Panel title="สัดส่วนตาม Product Category">
                <Donut slices={page.data.donut_category} format={(v) => `${num(v)} ลัง`} />
              </Panel>
              <Panel
                title="เครือข่ายทั่วประเทศ"
                tag={`${page.data.network.active_count} ศูนย์ · ${page.data.network.province_count} จังหวัด`}
              >
                <div className="p-3">
                  <NetworkMap nodes={page.data.network.nodes} selected={scope.dc || undefined} />
                  <p className="mt-2 flex items-center justify-center gap-1.5 text-[12px] text-muted">
                    <Building2 size={12} strokeWidth={1.75} />
                    พื้นที่ระบายสีคือจังหวัดที่แต่ละศูนย์ดูแล
                  </p>
                </div>
              </Panel>
            </div>
          </>
        )}
      </main>
    </>
  )
}

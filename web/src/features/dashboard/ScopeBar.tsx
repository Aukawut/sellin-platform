import { useNavigate } from 'react-router-dom'
import { FileSpreadsheet, RotateCcw } from 'lucide-react'
import { Select } from '../../components/ui/Field'
import { Icon } from '../../components/ui/Icon'
import { useFilters, useScope } from './useScope'
import { ExportButton } from './ExportButton'
import { useDatasets } from '../datasets/api'
import { monthName } from '../../lib/format'

interface Props {
  title: string
  subtitle: string
  /** ฟิลเตอร์เฉพาะหน้า เช่นกลุ่มสินค้าและสถานะสต๊อก */
  extra?: React.ReactNode
}

export function ScopeBar({ title, subtitle, extra }: Props) {
  const { scope, setScope } = useScope()
  const filters = useFilters(scope.dataset)
  const datasets = useDatasets(false)
  const navigate = useNavigate()

  const ready = datasets.data?.filter((d) => d.status === 'ready') ?? []
  const current = ready.find((d) => d.id === scope.dataset)
  const dirty = scope.dc !== 'ALL' || scope.year !== 'ALL' || scope.month !== 'ALL'

  return (
    <div className="sticky top-14 z-30 border-b border-line bg-paper/85 backdrop-blur">
      <div className="mx-auto flex max-w-[1400px] flex-wrap items-end justify-between gap-x-5 gap-y-3 px-5 py-3">
        <div className="min-w-0">
          <h1 className="font-display text-[17px] font-bold tracking-tight">{title}</h1>
          <p className="mt-0.5 max-w-[62ch] text-[12.5px] text-muted">{subtitle}</p>
        </div>

        <div className="flex flex-wrap items-end gap-2.5">
          <label className="flex flex-col gap-1.5">
            <span className="flex items-center gap-1 text-[11px] font-semibold uppercase tracking-[0.08em] text-muted">
              <Icon icon={FileSpreadsheet} size={11} />
              ชุดข้อมูล
            </span>
            <select
              value={scope.dataset}
              onChange={(e) => setScope({ dataset: e.target.value })}
              className="w-[200px] cursor-pointer rounded-lg border border-line bg-paper px-3 py-2 text-[13px] font-medium"
            >
              {!current && <option value="">— เลือกไฟล์ —</option>}
              {ready.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.label || d.original_filename}
                </option>
              ))}
            </select>
          </label>

          <Select
            label="ศูนย์กระจายสินค้า"
            value={scope.dc}
            onChange={(e) => setScope({ dc: e.target.value })}
            className="w-[190px] py-2 text-[13px]"
          >
            <option value="ALL">ทุกศูนย์ (รวม)</option>
            {filters.data?.customers.map((c) => (
              <option key={c.code} value={c.code}>
                {c.name}
              </option>
            ))}
          </Select>

          <Select
            label="ปี"
            value={scope.year}
            onChange={(e) => setScope({ year: e.target.value })}
            className="w-[104px] py-2 text-[13px]"
          >
            <option value="ALL">ทุกปี</option>
            {filters.data?.years.map((y) => (
              <option key={y} value={String(y)}>
                {y}
              </option>
            ))}
          </Select>

          <Select
            label="เดือน"
            value={scope.month}
            onChange={(e) => setScope({ month: e.target.value })}
            className="w-[118px] py-2 text-[13px]"
          >
            <option value="ALL">ทุกเดือน</option>
            {filters.data?.months.map((m) => (
              <option key={m} value={String(m)}>
                {monthName(m)}
              </option>
            ))}
          </Select>

          {extra}

          <ExportButton scope={scope} />

          {dirty && (
            <button
              type="button"
              onClick={() => setScope({ dc: 'ALL', year: 'ALL', month: 'ALL' })}
              title="ล้างฟิลเตอร์ทั้งหมด"
              className="grid size-[38px] place-items-center rounded-lg border border-line text-muted transition hover:bg-sunk hover:text-ink"
            >
              <Icon icon={RotateCcw} size={15} label="ล้างฟิลเตอร์" />
            </button>
          )}
        </div>
      </div>

      {!scope.dataset && (
        <div className="mx-auto max-w-[1400px] px-5 pb-3">
          <button
            type="button"
            onClick={() => navigate('/datasets')}
            className="w-full rounded-lg bg-warn-wash px-4 py-2.5 text-left text-[13px] text-accent-ink"
          >
            ยังไม่ได้เลือกชุดข้อมูล — เลือกจากรายการด้านบน หรือไปที่คลังไฟล์เพื่ออัปโหลดไฟล์ใหม่
          </button>
        </div>
      )}
    </div>
  )
}

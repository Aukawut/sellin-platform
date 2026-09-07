import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import type { Slice } from '../../types/api'
import { categoryColor } from './palette'
import { ChartTooltip } from './Tooltip'
import { deltaText } from '../../lib/format'

interface Props {
  slices: Slice[]
  format: (v: number) => string
  /** แสดงป้ายเทียบปีก่อนในรายการด้านล่าง ใช้เฉพาะหน้า Sell-Out */
  showYoY?: boolean
}

export function Donut({ slices, format, showYoY }: Props) {
  if (!slices.length) {
    return (
      <p className="px-4 py-10 text-center text-[13px] text-muted">
        ไม่มีข้อมูลในขอบเขตที่เลือก
      </p>
    )
  }

  return (
    <div className="p-4">
      <div className="h-[168px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={slices}
              dataKey="value"
              nameKey="label"
              innerRadius="58%"
              outerRadius="88%"
              paddingAngle={1.5}
              stroke="var(--paper)"
              strokeWidth={2}
            >
              {slices.map((s, i) => (
                <Cell key={s.label} fill={categoryColor(s.label, i)} />
              ))}
            </Pie>
            <Tooltip content={<ChartTooltip format={format} />} />
          </PieChart>
        </ResponsiveContainer>
      </div>

      <ul className="mt-3 flex flex-col gap-1.5">
        {slices.map((s, i) => (
          <li key={s.label} className="flex items-center gap-2.5 text-[12.5px]">
            <span
              className="size-2.5 shrink-0 rounded-[3px]"
              style={{ background: categoryColor(s.label, i) }}
              aria-hidden
            />
            <span className="min-w-0 flex-1 truncate">{s.label}</span>
            <span className="tnum shrink-0 text-muted">{format(s.value)}</span>
            <span className="tnum w-12 shrink-0 text-right font-semibold">{s.pct.toFixed(1)}%</span>
            {showYoY && (
              <span
                className={`tnum w-16 shrink-0 text-right text-[11px] ${
                  !s.yoy ? 'text-faint' : s.yoy.is_new || s.yoy.direction === 'up' ? 'text-good' : 'text-bad'
                }`}
                title={s.yoy ? deltaText(s.yoy) : 'ไม่มีข้อมูลปีก่อนหน้าให้เทียบ'}
              >
                {!s.yoy
                  ? 'N/A'
                  : s.yoy.is_new
                    ? '▲ ใหม่'
                    : `${s.yoy.direction === 'up' ? '▲' : '▼'} ${Math.abs(s.yoy.pct).toFixed(1)}%`}
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

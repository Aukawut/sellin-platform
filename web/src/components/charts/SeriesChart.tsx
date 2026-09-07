import {
  Bar, BarChart, CartesianGrid, ComposedChart, Legend, Line,
  ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import type { Chart } from '../../types/api'
import { AXIS, seriesColor } from './palette'
import { ChartTooltip } from './Tooltip'

interface Props {
  chart: Chart
  height?: number
  /** แปลงค่าบนแกน y และใน tooltip — กราฟเงินกับกราฟจำนวนลังใช้คนละแบบ */
  format: (v: number) => string
  axisFormat?: (v: number) => string
  labelFormat?: (l: string) => string
  referenceLabel?: string
}

/** toRows แปลงโครงสร้าง {labels, series} ของ API เป็นรูปแบบที่ Recharts ต้องการ */
function toRows(chart: Chart) {
  return chart.labels.map((label, i) => {
    const row: Record<string, string | number | null> = { label }
    chart.series.forEach((s) => {
      row[s.key] = s.points[i] ?? null
    })
    return row
  })
}

export function SeriesChart({
  chart, height = 260, format, axisFormat, labelFormat, referenceLabel,
}: Props) {
  const rows = toRows(chart)
  const yFormat = axisFormat ?? format
  const hasBar = chart.series.some((s) => s.kind === 'bar')

  const legend = chart.series.length > 1

  return (
    <div style={{ height }} className="px-2 pb-2 pt-4">
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={rows} margin={{ top: 4, right: 10, bottom: 0, left: 4 }}>
          <CartesianGrid stroke={AXIS.stroke} strokeDasharray="3 3" vertical={false} />
          <XAxis
            dataKey="label"
            tick={AXIS.tick}
            tickLine={false}
            axisLine={{ stroke: AXIS.stroke }}
            tickFormatter={labelFormat}
            interval="preserveStartEnd"
          />
          <YAxis
            tick={AXIS.tick}
            tickLine={false}
            axisLine={false}
            width={58}
            tickFormatter={(v: number) => yFormat(v)}
          />
          <Tooltip
            cursor={hasBar ? { fill: 'var(--sunk)' } : { stroke: AXIS.stroke }}
            content={<ChartTooltip format={format} labelFormat={labelFormat} />}
          />
          {legend && (
            <Legend
              verticalAlign="top"
              align="right"
              height={26}
              iconType={hasBar ? 'square' : 'plainline'}
              iconSize={hasBar ? 10 : 14}
              wrapperStyle={{ fontSize: 12, color: 'var(--muted)' }}
            />
          )}

          {chart.reference !== undefined && (
            <ReferenceLine
              y={chart.reference}
              stroke="var(--bad)"
              strokeDasharray="5 4"
              strokeWidth={1.5}
              label={{
                value: `${referenceLabel ?? 'ค่าเฉลี่ย'}: ${format(chart.reference)}`,
                position: 'insideTopRight',
                fill: 'var(--bad)',
                fontSize: 11,
                fontWeight: 600,
              }}
            />
          )}

          {chart.series.map((s, i) =>
            s.kind === 'bar' ? (
              <Bar
                key={s.key}
                dataKey={s.key}
                name={s.name}
                fill={s.color ?? seriesColor(i)}
                radius={[3, 3, 0, 0]}
                maxBarSize={38}
              />
            ) : (
              <Line
                key={s.key}
                type="monotone"
                dataKey={s.key}
                name={s.name}
                stroke={s.color ?? seriesColor(i)}
                strokeWidth={s.kind === 'dashed' ? 2 : 2.2}
                strokeDasharray={s.kind === 'dashed' ? '6 4' : undefined}
                dot={{ r: 2.5, strokeWidth: 0 }}
                activeDot={{ r: 4.5 }}
                connectNulls={false}
              />
            ),
          )}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  )
}

/** BarsOnly ใช้กับกราฟที่มีแต่แท่งและป้ายแกน x เป็นข้อความยาว เช่นชื่อกลุ่มสินค้า */
export function CategoryBars({ chart, height = 260, format }: Props) {
  const rows = toRows(chart)
  return (
    <div style={{ height }} className="px-2 pb-2 pt-4">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={rows} margin={{ top: 4, right: 10, bottom: 0, left: 4 }}>
          <CartesianGrid stroke={AXIS.stroke} strokeDasharray="3 3" vertical={false} />
          <XAxis dataKey="label" tick={AXIS.tick} tickLine={false} axisLine={{ stroke: AXIS.stroke }} />
          <YAxis tick={AXIS.tick} tickLine={false} axisLine={false} width={58} tickFormatter={(v: number) => format(v)} />
          <Tooltip cursor={{ fill: 'var(--sunk)' }} content={<ChartTooltip format={format} />} />
          <Legend
            verticalAlign="top"
            align="right"
            height={26}
            iconType="square"
            iconSize={10}
            wrapperStyle={{ fontSize: 12, color: 'var(--muted)' }}
          />
          {chart.series.map((s, i) => (
            <Bar key={s.key} dataKey={s.key} name={s.name} fill={seriesColor(i)} radius={[3, 3, 0, 0]} maxBarSize={44} />
          ))}
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}

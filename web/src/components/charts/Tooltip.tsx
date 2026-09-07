
interface Entry {
  name?: string
  value?: number | string
  color?: string
  dataKey?: string | number
}

interface Props {
  active?: boolean
  payload?: Entry[]
  label?: string | number
  /** แปลงค่าตัวเลขเป็นข้อความ ต่างกันไปตามว่ากราฟนั้นเป็นเงินหรือจำนวนลัง */
  format: (v: number) => string
  /** แปลงป้ายแกน x เช่นเลขเดือนเป็นชื่อเดือนไทย */
  labelFormat?: (l: string) => string
}

export function ChartTooltip({ active, payload, label, format, labelFormat }: Props) {
  if (!active || !payload?.length) return null

  const rows = payload.filter((p) => p.value !== null && p.value !== undefined)
  if (!rows.length) return null

  return (
    <div className="rounded-lg border border-line bg-paper px-3 py-2 text-[12px] shadow-panel">
      <p className="mb-1 font-semibold text-ink">
        {labelFormat ? labelFormat(String(label)) : label}
      </p>
      <ul className="flex flex-col gap-0.5">
        {rows.map((p, i) => (
          <li key={i} className="flex items-center gap-2 whitespace-nowrap">
            <span className="size-2 shrink-0 rounded-[2px]" style={{ background: p.color }} aria-hidden />
            <span className="text-muted">{p.name}</span>
            <span className="tnum ml-auto font-semibold text-ink">
              {typeof p.value === 'number' ? format(p.value) : p.value}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

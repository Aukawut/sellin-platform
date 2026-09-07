import type { ReactNode } from 'react'

export interface Column<T> {
  key: string
  header: string
  align?: 'left' | 'right'
  width?: string
  render: (row: T) => ReactNode
}

interface Props<T> {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
  /** แถบสีบางด้านซ้ายของแถว ใช้บอกความรุนแรงโดยไม่ต้องอ่านข้อความ */
  stripe?: (row: T) => string | undefined
  empty?: ReactNode
}

export function DataTable<T>({ columns, rows, rowKey, stripe, empty }: Props<T>) {
  if (!rows.length) {
    return (
      <div className="px-4 py-10 text-center text-[13px] text-muted">
        {empty ?? 'ไม่มีข้อมูลในขอบเขตที่เลือก'}
      </div>
    )
  }

  return (
    // ตารางกว้างเลื่อนในกรอบของตัวเอง หน้าเว็บทั้งหน้าจึงไม่เลื่อนออกด้านข้าง
    <div className="overflow-x-auto">
      <table className="w-full min-w-[640px] border-collapse text-[13px]">
        <thead>
          <tr>
            {columns.map((c) => (
              <th
                key={c.key}
                style={{ width: c.width }}
                className={`whitespace-nowrap border-b border-line bg-sunk px-3 py-2.5 text-[11px]
                  font-semibold uppercase tracking-[0.07em] text-muted
                  ${c.align === 'right' ? 'text-right' : 'text-left'}`}
              >
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const color = stripe?.(row)
            return (
              <tr key={rowKey(row)} className="border-b border-line-soft last:border-b-0">
                {columns.map((c, ci) => (
                  <td
                    key={c.key}
                    className={`px-3 py-2.5 align-top ${c.align === 'right' ? 'tnum text-right' : ''}`}
                    style={
                      ci === 0 && color
                        ? { boxShadow: `inset 3px 0 0 ${color}` }
                        : undefined
                    }
                  >
                    {c.render(row)}
                  </td>
                ))}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

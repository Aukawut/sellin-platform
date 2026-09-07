import type { RankEntry } from '../../types/api'
import { thbShort } from '../../lib/format'

/**
 * อันดับแสดงเป็นตัวเลขแบบ tabular พร้อมแถบสัดส่วน แทนเหรียญ emoji ของ dashboard เดิม
 * แถบบอกได้ว่าอันดับสองห่างจากอันดับหนึ่งแค่ไหน ซึ่งเหรียญบอกไม่ได้
 */
export function RankList({ entries, highlight }: { entries: RankEntry[]; highlight?: string }) {
  if (!entries.length) {
    return <p className="px-4 py-10 text-center text-[13px] text-muted">ไม่มีข้อมูลในขอบเขตที่เลือก</p>
  }

  return (
    <ul className="flex flex-col">
      {entries.map((e) => {
        const active = highlight && highlight === e.code
        return (
          <li
            key={e.code}
            className={`flex items-center gap-3 border-b border-line-soft px-4 py-2.5 last:border-b-0 ${
              active ? 'bg-accent-wash' : ''
            }`}
          >
            <span
              className={`tnum w-6 shrink-0 text-right text-[12px] font-semibold ${
                e.rank <= 3 ? 'text-accent-ink' : 'text-faint'
              }`}
            >
              {e.rank}
            </span>

            <div className="min-w-0 flex-1">
              <p className="truncate text-[13px] font-medium">{e.name}</p>
              <div className="mt-1 h-1 overflow-hidden rounded-full bg-sunk">
                <div
                  className={`h-full rounded-full ${e.rank <= 3 ? 'bg-accent' : 'bg-faint/50'}`}
                  style={{ width: `${Math.max(e.share * 100, 1.5)}%` }}
                />
              </div>
            </div>

            <span className="tnum shrink-0 text-[12.5px] font-semibold">{thbShort(e.value)}</span>
          </li>
        )
      })}
    </ul>
  )
}

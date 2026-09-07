import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Icon } from './Icon'

export function Spinner({ label = 'กำลังโหลด' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2.5 py-14 text-sm text-muted">
      <span
        className="size-4 animate-spin rounded-full border-2 border-line border-t-accent"
        aria-hidden
      />
      {label}
    </div>
  )
}

interface EmptyProps {
  icon: LucideIcon
  title: string
  /** บอกด้วยว่าทำไมถึงว่าง ไม่ใช่แค่ว่าไม่มีข้อมูล */
  detail?: string
  action?: ReactNode
}

export function EmptyState({ icon, title, detail, action }: EmptyProps) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-line px-6 py-12 text-center">
      <span className="grid size-11 place-items-center rounded-xl bg-sunk text-faint">
        <Icon icon={icon} size={20} />
      </span>
      <div>
        <p className="font-display text-[15px] font-semibold">{title}</p>
        {detail && <p className="mt-1 max-w-[46ch] text-[13px] text-muted">{detail}</p>}
      </div>
      {action}
    </div>
  )
}

export function ErrorState({ message, retry }: { message: string; retry?: () => void }) {
  return (
    <div
      role="alert"
      className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-bad-wash px-4 py-3.5 text-[13px] text-bad"
    >
      <span>{message}</span>
      {retry && (
        <button
          type="button"
          onClick={retry}
          className="font-semibold underline underline-offset-2"
        >
          ลองใหม่
        </button>
      )}
    </div>
  )
}

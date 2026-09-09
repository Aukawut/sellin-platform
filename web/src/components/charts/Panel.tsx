import type { ReactNode } from 'react'

export function Panel({
  title, tag, children, className = '',
}: {
  title?: string
  tag?: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <section className={`chart-panel rounded-2xl border border-line bg-paper shadow-panel ${className}`}>
      {(title || tag) && (
        <header className="flex flex-wrap items-center justify-between gap-2 border-b border-line-soft px-5 py-4">
          {title && <h3 className="font-display text-[14px] font-semibold">{title}</h3>}
          {tag && (
            <span className="rounded-full bg-info-wash px-2 py-0.5 text-[11px] font-medium text-muted">
              {tag}
            </span>
          )}
        </header>
      )}
      {children}
    </section>
  )
}

export function SectionHead({ title, note }: { title: string; note?: ReactNode }) {
  return (
    <div className="mb-4 mt-8 flex flex-wrap items-baseline justify-between gap-2">
      <h2 className="flex items-center gap-2 font-display text-[15px] font-semibold">
        <span className="h-4 w-1 rounded-full bg-accent" aria-hidden />
        {title}
      </h2>
      {note && <span className="text-[12.5px] text-muted">{note}</span>}
    </div>
  )
}

type Tone = 'good' | 'warn' | 'bad' | 'info' | 'neutral'

const tones: Record<Tone, string> = {
  good: 'bg-good-wash text-good',
  warn: 'bg-warn-wash text-accent-ink',
  bad: 'bg-bad-wash text-bad',
  info: 'bg-info-wash text-info',
  neutral: 'bg-sunk text-muted',
}

export function Pill({ tone = 'neutral', children }: { tone?: Tone; children: React.ReactNode }) {
  return (
    <span
      className={`inline-block whitespace-nowrap rounded-full px-2 py-0.5 text-[11px] font-semibold ${tones[tone]}`}
    >
      {children}
    </span>
  )
}

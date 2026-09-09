import type { ButtonHTMLAttributes, ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Icon } from './Icon'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  icon?: LucideIcon
  loading?: boolean
  children?: ReactNode
}

const styles: Record<Variant, string> = {
  primary: 'bg-accent text-[var(--on-accent)] shadow-sm hover:brightness-110 border-transparent',
  secondary: 'bg-paper text-ink border-line hover:bg-sunk',
  ghost: 'bg-transparent text-muted border-transparent hover:bg-sunk hover:text-ink',
  danger: 'bg-bad-wash text-bad border-transparent hover:brightness-95',
}

export function Button({ variant = 'secondary', icon, loading, children, className = '', disabled, ...rest }: Props) {
  return (
    <button
      {...rest}
      disabled={disabled || loading}
      className={`ui-button inline-flex min-h-10 items-center justify-center gap-2 rounded-lg border px-3.5 py-2
        text-[13px] font-semibold transition disabled:cursor-not-allowed disabled:opacity-55
        ${styles[variant]} ${className}`}
    >
      {icon && !loading && <Icon icon={icon} size={16} />}
      {loading && (
        <span
          className="size-3.5 animate-spin rounded-full border-2 border-current border-t-transparent"
          aria-hidden
        />
      )}
      {children}
    </button>
  )
}

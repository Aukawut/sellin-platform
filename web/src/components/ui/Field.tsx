import type { InputHTMLAttributes, SelectHTMLAttributes, ReactNode } from 'react'

const control =
  'ui-control min-h-10 w-full rounded-lg border border-line bg-paper px-3 py-2 text-[14px] text-ink ' +
  'placeholder:text-faint transition focus:border-accent'

export function Label({ children }: { children: ReactNode }) {
  return (
    <span className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted">
      {children}
    </span>
  )
}

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  hint?: string
}

export function Input({ label, hint, className = '', ...rest }: InputProps) {
  return (
    <label className="flex flex-col gap-1.5">
      {label && <Label>{label}</Label>}
      <input {...rest} className={`${control} ${className}`} />
      {hint && <span className="text-xs text-faint">{hint}</span>}
    </label>
  )
}

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
}

export function Select({ label, className = '', children, ...rest }: SelectProps) {
  return (
    <label className="flex flex-col gap-1.5">
      {label && <Label>{label}</Label>}
      <select {...rest} className={`${control} cursor-pointer pr-8 ${className}`}>
        {children}
      </select>
    </label>
  )
}

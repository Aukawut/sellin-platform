import { useEffect, useState } from 'react'
import { Monitor, Moon, Sun } from 'lucide-react'
import { Icon } from './Icon'

type Theme = 'light' | 'dark' | 'system'
const KEY = 'sellin.theme'

function read(): Theme {
  try {
    const v = localStorage.getItem(KEY)
    return v === 'light' || v === 'dark' ? v : 'system'
  } catch {
    // เบราว์เซอร์โหมดส่วนตัวอาจห้ามอ่าน storage — ถือว่าใช้ค่าของระบบ
    return 'system'
  }
}

const options: { value: Theme; icon: typeof Sun; label: string }[] = [
  { value: 'light', icon: Sun, label: 'สว่าง' },
  { value: 'dark', icon: Moon, label: 'มืด' },
  { value: 'system', icon: Monitor, label: 'ตามระบบ' },
]

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(read)

  useEffect(() => {
    const root = document.documentElement
    if (theme === 'system') {
      root.removeAttribute('data-theme')
    } else {
      root.setAttribute('data-theme', theme)
    }
    try {
      if (theme === 'system') localStorage.removeItem(KEY)
      else localStorage.setItem(KEY, theme)
    } catch {
      /* ธีมยังใช้ได้ในหน้านี้ แม้จำค่าไว้ครั้งหน้าไม่ได้ */
    }
  }, [theme])

  return (
    <div
      role="radiogroup"
      aria-label="ธีมของหน้าจอ"
      className="flex items-center gap-0.5 rounded-lg border border-line bg-paper p-0.5"
    >
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          role="radio"
          aria-checked={theme === o.value}
          title={o.label}
          onClick={() => setTheme(o.value)}
          className={`grid size-7 place-items-center rounded-[6px] transition ${
            theme === o.value ? 'bg-sunk text-ink' : 'text-faint hover:text-ink'
          }`}
        >
          <Icon icon={o.icon} size={14} label={o.label} />
        </button>
      ))}
    </div>
  )
}

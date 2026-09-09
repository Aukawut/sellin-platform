import { NavLink, Outlet, useSearchParams } from 'react-router-dom'
import { Layers, LibraryBig, LogOut, Settings, Target, TrendingUp, Warehouse } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Icon } from '../components/ui/Icon'
import { ThemeToggle } from '../components/ui/ThemeToggle'
import { useAuth } from '../features/auth/AuthContext'

const tabs: { to: string; label: string; icon: LucideIcon }[] = [
  { to: '/dashboard/sellin', label: 'Sell-In', icon: TrendingUp },
  { to: '/dashboard/sellout', label: 'Sell-Out', icon: Target },
  { to: '/dashboard/stock', label: 'สต๊อก', icon: Layers },
  { to: '/dashboard/planning', label: 'วางแผน', icon: Warehouse },
]

/**
 * แถบบนบางแทน sidebar เต็มความสูงของ dashboard เดิม
 * คืนพื้นที่แนวนอนให้ตาราง ซึ่งเป็นเนื้อหาที่กว้างที่สุดในระบบ
 */
export function AppShell() {
  const { user, logout } = useAuth()
  const [params] = useSearchParams()
  const dataset = params.get('dataset')
  const suffix = dataset ? `?dataset=${dataset}` : ''

  return (
    <div className="min-h-dvh">
      <header className="app-header sticky top-0 z-40">
        <div className="app-header-inner mx-auto flex max-w-[1400px] items-center gap-4 px-5">
          <NavLink to={`/datasets`} aria-label="Sell-In Performance — คลังไฟล์ข้อมูล" className="app-brand flex shrink-0 items-center gap-2.5">
            <span className="grid size-8 place-items-center rounded-lg bg-accent text-white">
              <Icon icon={TrendingUp} size={17} />
            </span>
            <span className="font-display text-[14px] font-bold tracking-tight">
              Sell-In Performance
            </span>
          </NavLink>

          <nav aria-label="เมนูหลัก" className="app-nav flex min-w-0 flex-1 items-center gap-1 overflow-x-auto">
            <NavLink
              to="/datasets"
              className={({ isActive }) => tabClass(isActive)}
              title="คลังไฟล์ข้อมูล"
            >
              <Icon icon={LibraryBig} size={15} />
              <span className="whitespace-nowrap">คลังไฟล์</span>
            </NavLink>

            <span className="mx-1 h-5 w-px shrink-0 bg-line" aria-hidden />

            {tabs.map((t) => (
              <NavLink key={t.to} to={`${t.to}${suffix}`} aria-label={t.label} className={({ isActive }) => tabClass(isActive)}>
                <Icon icon={t.icon} size={15} />
                <span className="whitespace-nowrap">{t.label}</span>
              </NavLink>
            ))}
          </nav>

          <div className="app-account flex shrink-0 items-center gap-2">
            <ThemeToggle />
            <div className="hidden text-right lg:block">
              <p className="text-[12.5px] font-semibold leading-tight">{user?.display_name}</p>
              <p className="text-[11px] leading-tight text-faint">
                {user?.role === 'admin' ? 'ผู้ดูแลระบบ' : 'ผู้ใช้งาน'}
              </p>
            </div>
            <NavLink
              to="/settings"
              title="ตั้งค่าระบบ"
              className={({ isActive }) =>
                `grid size-8 place-items-center rounded-lg transition ${
                  isActive ? 'bg-accent-wash text-accent-ink' : 'text-faint hover:bg-sunk hover:text-ink'
                }`
              }
            >
              <Icon icon={Settings} size={15} label="ตั้งค่าระบบ" />
            </NavLink>
            <button
              type="button"
              onClick={() => void logout()}
              title="ออกจากระบบ"
              className="grid size-8 place-items-center rounded-lg text-faint transition hover:bg-sunk hover:text-ink"
            >
              <Icon icon={LogOut} size={15} label="ออกจากระบบ" />
            </button>
          </div>
        </div>
      </header>

      <Outlet />
    </div>
  )
}

function tabClass(isActive: boolean) {
  return `app-tab flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-[13px] font-semibold transition ${
    isActive ? 'app-tab-active' : 'app-tab-idle'
  }`
}

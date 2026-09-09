import { useState } from 'react'
import type { FormEvent } from 'react'
import { LogIn, TrendingUp, TriangleAlert } from 'lucide-react'
import { useAuth } from './AuthContext'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Field'
import { Icon } from '../../components/ui/Icon'
import { ApiError } from '../../lib/api'
import { LoginParticles } from './LoginParticles'

export function LoginPage() {
  const { login } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await login(email.trim(), password)
    } catch (err) {
      // ข้อความจากเซิร์ฟเวอร์เป็นภาษาไทยและบอกวิธีแก้อยู่แล้ว จึงแสดงตรง ๆ
      setError(err instanceof ApiError ? err.message : 'เชื่อมต่อเซิร์ฟเวอร์ไม่ได้ ตรวจสอบว่าระบบเปิดอยู่หรือไม่')
      setBusy(false)
    }
  }

  return (
    <div className="login-page grid min-h-dvh place-items-center px-5 py-10">
      <LoginParticles />
      <main className="login-content w-full max-w-[420px]">
        <div className="login-brand mb-8 flex items-center gap-3">
          <span className="grid size-10 place-items-center rounded-xl bg-accent text-white">
            <Icon icon={TrendingUp} size={20} />
          </span>
          <div>
            <h1 className="text-[19px] font-bold leading-tight tracking-tight">
              Sell-In Performance
            </h1>
            <p className="text-xs text-muted">Value Plus Worldwide</p>
          </div>
        </div>

        <form
          onSubmit={handleSubmit}
          className="login-form flex flex-col gap-5 rounded-2xl border border-line bg-paper p-7 shadow-panel"
        >
          <div className="mb-1">
            <h2 className="text-[24px] font-bold">เข้าสู่ระบบ</h2>
            <p className="mt-1 text-[13px] text-muted">จัดการข้อมูลและติดตามผลการขายของคุณ</p>
          </div>
          <Input
            label="อีเมล"
            type="email"
            autoComplete="username"
            required
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@prospira.com"
          />
          <Input
            label="รหัสผ่าน"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />

          {error && (
            <p
              role="alert"
              className="flex items-start gap-2 rounded-lg bg-bad-wash px-3 py-2.5 text-[13px] text-bad"
            >
              <Icon icon={TriangleAlert} size={15} className="mt-0.5 shrink-0" />
              {error}
            </p>
          )}

          <Button type="submit" variant="primary" icon={LogIn} loading={busy} className="mt-1 py-2.5">
            เข้าสู่ระบบ
          </Button>
        </form>

        <p className="mt-5 text-center text-xs leading-relaxed text-faint">
          ระบบนี้ไม่เปิดให้สมัครเอง หากยังไม่มีบัญชี
          <br />
          กรุณาติดต่อผู้ดูแลระบบเพื่อขอเปิดสิทธิ์
        </p>
      </main>
    </div>
  )
}

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { api, setUnauthorizedHandler } from '../../lib/api'
import type { User } from '../../types/api'

interface AuthState {
  user: User | null
  /** true ระหว่างกำลังกู้เซสชันตอนเปิดหน้าเว็บครั้งแรก */
  restoring: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [restoring, setRestoring] = useState(true)

  useEffect(() => {
    let cancelled = false

    // access token อยู่ในหน่วยความจำ จึงหายทุกครั้งที่รีเฟรชหน้า
    // แต่ refresh cookie ยังอยู่ ระบบจึงพากลับเข้าสู่ระบบให้เองโดยผู้ใช้ไม่ต้องพิมพ์รหัสซ้ำ
    api.restoreSession().then((u) => {
      if (!cancelled) {
        setUser(u)
        setRestoring(false)
      }
    })

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    // เมื่อการหมุน token ล้มเหลว ให้พาออกจากระบบทันทีแทนที่จะปล่อยให้หน้าเว็บพังเงียบ ๆ
    setUnauthorizedHandler(() => setUser(null))
    return () => setUnauthorizedHandler(null)
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const data = await api.login(email, password)
    setUser(data.user)
  }, [])

  const logout = useCallback(async () => {
    await api.logout()
    setUser(null)
  }, [])

  const value = useMemo(() => ({ user, restoring, login, logout }), [user, restoring, login, logout])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth ต้องถูกเรียกภายใน AuthProvider')
  return ctx
}

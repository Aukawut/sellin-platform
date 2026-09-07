import type { LoginResponse, User } from '../types/api'

const BASE = '/api/v1'

/** ApiError พก HTTP status มาด้วย เพื่อให้ผู้เรียกแยกได้ว่าควรพาไปหน้าล็อกอินหรือแค่แสดงข้อความ */
export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

// access token อยู่ในหน่วยความจำเท่านั้น ไม่เก็บลง localStorage
// เพราะถ้าหน้าเว็บมีช่องโหว่ XSS สคริปต์จะอ่าน token จาก storage ได้ทันที
// ส่วน refresh token เป็น HttpOnly cookie ซึ่ง JavaScript อ่านไม่ได้อยู่แล้ว
let accessToken: string | null = null
let onUnauthorized: (() => void) | null = null

export function setAccessToken(token: string | null) {
  accessToken = token
}

export function setUnauthorizedHandler(fn: (() => void) | null) {
  onUnauthorized = fn
}

/** authHeader ให้ตัวดาวน์โหลดไฟล์ยืม token ไปใช้ได้ โดยไม่ต้องเปิดตัวแปรให้แก้จากภายนอก */
export function authHeader(): Record<string, string> {
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
}

/** คำขอหมุน token ที่กำลังทำงานอยู่ ใช้ร่วมกันเพื่อไม่ให้หลาย request หมุนพร้อมกันจนชนกัน */
let refreshing: Promise<boolean> | null = null

async function refreshAccessToken(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      try {
        const res = await fetch(`${BASE}/auth/refresh`, {
          method: 'POST',
          credentials: 'include',
        })
        if (!res.ok) return false
        const data: LoginResponse = await res.json()
        accessToken = data.access_token
        return true
      } catch {
        return false
      } finally {
        // ปล่อยตัวล็อกในรอบถัดไปของ event loop เพื่อให้ทุก request ที่รออยู่ได้ผลลัพธ์เดียวกัน
        setTimeout(() => {
          refreshing = null
        }, 0)
      }
    })()
  }
  return refreshing
}

interface RequestOptions {
  method?: string
  body?: unknown
  signal?: AbortSignal
  /** ใช้ภายในเพื่อกันการวนหมุน token ไม่รู้จบ */
  retried?: boolean
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {}
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`

  let body: BodyInit | undefined
  if (opts.body instanceof FormData) {
    body = opts.body
  } else if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(opts.body)
  }

  const res = await fetch(`${BASE}${path}`, {
    method: opts.method ?? 'GET',
    headers,
    body,
    credentials: 'include',
    signal: opts.signal,
  })

  // access token อายุ 15 นาที การหมดอายุระหว่างใช้งานเป็นเรื่องปกติ
  // จึงหมุน token แล้วยิงซ้ำให้เงียบ ๆ แทนที่จะเด้งผู้ใช้ออกกลางคัน
  if (res.status === 401 && !opts.retried) {
    if (await refreshAccessToken()) {
      return request<T>(path, { ...opts, retried: true })
    }
    accessToken = null
    onUnauthorized?.()
    throw new ApiError('เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่', 401)
  }

  if (!res.ok) {
    let message = `คำขอล้มเหลว (${res.status})`
    try {
      const data = await res.json()
      if (typeof data?.error === 'string') message = data.error
    } catch {
      /* บางกรณีเซิร์ฟเวอร์ไม่ได้ตอบเป็น JSON เช่นถูก proxy ตัดกลางทาง */
    }
    throw new ApiError(message, res.status)
  }

  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  async login(email: string, password: string) {
    const data = await request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: { email, password },
    })
    accessToken = data.access_token
    return data
  },

  async logout() {
    try {
      await request('/auth/logout', { method: 'POST' })
    } finally {
      accessToken = null
    }
  },

  /** พยายามกู้เซสชันจาก refresh cookie ตอนเปิดหน้าเว็บครั้งแรก */
  async restoreSession(): Promise<User | null> {
    if (!(await refreshAccessToken())) return null
    try {
      const { user } = await request<{ user: User }>('/me')
      return user
    } catch {
      return null
    }
  },

  get: <T>(path: string, signal?: AbortSignal) => request<T>(path, { signal }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}

/** buildQuery ข้ามค่าที่เป็น undefined เพื่อไม่ให้ query string มีพารามิเตอร์ว่างเปล่า */
export function buildQuery(params: Record<string, string | number | undefined>) {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') q.set(k, String(v))
  }
  const s = q.toString()
  return s ? `?${s}` : ''
}

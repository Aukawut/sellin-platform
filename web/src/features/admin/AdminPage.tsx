import { useState } from 'react'
import type { FormEvent } from 'react'
import { Check, KeyRound, MapPin, UserPlus } from 'lucide-react'
import { Panel, SectionHead } from '../../components/charts/Panel'
import { Button } from '../../components/ui/Button'
import { Input, Select } from '../../components/ui/Field'
import { Icon } from '../../components/ui/Icon'
import { Pill } from '../../components/ui/Pill'
import { DataTable } from '../../components/data/Table'
import type { Column } from '../../components/data/Table'
import { ErrorState, Spinner } from '../../components/ui/States'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../../lib/api'
import { formatDate } from '../../lib/format'
import {
  useChangeOwnPassword, useCreateUser, useRegions, useResetPassword,
  useSetActive, useSetRegion, useUsers,
} from './api'
import type { RegionRow } from './api'
import type { Role, User } from '../../types/api'

export function AdminPage() {
  const { user } = useAuth()

  return (
    <main className="mx-auto flex max-w-[1180px] flex-col gap-2 px-5 pb-16">
      <div className="pt-7">
        <h1 className="font-display text-[20px] font-bold tracking-tight">ตั้งค่าระบบ</h1>
        <p className="mt-0.5 text-[13px] text-muted">
          จัดการผู้ใช้ ภาคของศูนย์กระจายสินค้า และรหัสผ่านของคุณเอง
        </p>
      </div>

      <OwnPasswordSection />
      {user?.role === 'admin' && (
        <>
          <UsersSection currentUserId={user.id} />
          <RegionsSection />
        </>
      )}
    </main>
  )
}

/* ---------- รหัสผ่านของตัวเอง ---------- */

function OwnPasswordSection() {
  const change = useChangeOwnPassword()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setDone(false)
    if (next !== confirm) {
      setError('รหัสผ่านใหม่ทั้งสองช่องไม่ตรงกัน')
      return
    }
    try {
      await change.mutateAsync({ current_password: current, new_password: next })
      setCurrent('')
      setNext('')
      setConfirm('')
      setDone(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'เปลี่ยนรหัสผ่านไม่สำเร็จ')
    }
  }

  return (
    <>
      <SectionHead title="รหัสผ่านของคุณ" note="ต้องยาวอย่างน้อย 12 ตัวอักษร" />
      <Panel>
        <form onSubmit={submit} className="grid gap-4 p-5 md:grid-cols-3">
          <Input
            label="รหัสผ่านปัจจุบัน"
            type="password"
            autoComplete="current-password"
            required
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
          />
          <Input
            label="รหัสผ่านใหม่"
            type="password"
            autoComplete="new-password"
            required
            minLength={12}
            value={next}
            onChange={(e) => setNext(e.target.value)}
          />
          <Input
            label="ยืนยันรหัสผ่านใหม่"
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <div className="md:col-span-3 flex flex-wrap items-center gap-3">
            <Button type="submit" variant="primary" icon={KeyRound} loading={change.isPending}>
              เปลี่ยนรหัสผ่าน
            </Button>
            {done && (
              <span className="flex items-center gap-1.5 text-[13px] text-good">
                <Icon icon={Check} size={14} />
                เปลี่ยนรหัสผ่านแล้ว
              </span>
            )}
            {error && (
              <span role="alert" className="text-[13px] text-bad">
                {error}
              </span>
            )}
          </div>
        </form>
      </Panel>
    </>
  )
}

/* ---------- ผู้ใช้ ---------- */

function UsersSection({ currentUserId }: { currentUserId: string }) {
  const users = useUsers()
  const setActive = useSetActive()
  const reset = useResetPassword()
  const [resetting, setResetting] = useState<User | null>(null)

  const columns: Column<User>[] = [
    {
      key: 'name',
      header: 'ผู้ใช้',
      render: (u) => (
        <>
          <span className="font-medium">{u.display_name}</span>
          {u.id === currentUserId && <span className="ml-2 text-[11px] text-faint">(คุณ)</span>}
          <div className="text-[12px] text-muted">{u.email}</div>
        </>
      ),
    },
    {
      key: 'role',
      header: 'สิทธิ์',
      width: '130px',
      render: (u) => (
        <Pill tone={u.role === 'admin' ? 'info' : 'neutral'}>
          {u.role === 'admin' ? 'ผู้ดูแลระบบ' : 'ผู้ใช้งาน'}
        </Pill>
      ),
    },
    {
      key: 'status',
      header: 'สถานะ',
      width: '120px',
      render: (u) => (
        <Pill tone={u.is_active ? 'good' : 'bad'}>{u.is_active ? 'ใช้งานอยู่' : 'ปิดใช้งาน'}</Pill>
      ),
    },
    {
      key: 'created',
      header: 'สร้างเมื่อ',
      width: '170px',
      render: (u) => <span className="text-muted">{formatDate(u.created_at)}</span>,
    },
    {
      key: 'actions',
      header: '',
      width: '210px',
      render: (u) => (
        <div className="flex justify-end gap-2">
          <Button onClick={() => setResetting(u)}>ตั้งรหัสใหม่</Button>
          <Button
            variant={u.is_active ? 'danger' : 'secondary'}
            disabled={u.id === currentUserId && u.is_active}
            title={
              u.id === currentUserId && u.is_active
                ? 'ปิดการใช้งานบัญชีของตัวเองไม่ได้'
                : undefined
            }
            onClick={() => setActive.mutate({ id: u.id, active: !u.is_active })}
          >
            {u.is_active ? 'ปิดใช้งาน' : 'เปิดใช้งาน'}
          </Button>
        </div>
      ),
    },
  ]

  return (
    <>
      <SectionHead
        title="ผู้ใช้งานระบบ"
        note={`${users.data?.length ?? 0} บัญชี · ระบบนี้ไม่เปิดให้สมัครเอง`}
      />
      {users.isPending && <Spinner label="กำลังโหลดรายชื่อผู้ใช้" />}
      {users.isError && <ErrorState message="โหลดรายชื่อผู้ใช้ไม่สำเร็จ" retry={() => users.refetch()} />}
      {users.data && (
        <Panel>
          <DataTable columns={columns} rows={users.data} rowKey={(u) => u.id} />
        </Panel>
      )}

      <CreateUserForm />

      {resetting && (
        <ResetPasswordDialog
          user={resetting}
          busy={reset.isPending}
          onClose={() => setResetting(null)}
          onSubmit={async (password) => {
            await reset.mutateAsync({ id: resetting.id, password })
            setResetting(null)
          }}
        />
      )}
    </>
  )
}

function CreateUserForm() {
  const create = useCreateUser()
  const [form, setForm] = useState({ email: '', display_name: '', password: '', role: 'viewer' as Role })
  const [error, setError] = useState<string | null>(null)
  const [created, setCreated] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setCreated(null)
    try {
      const res = await create.mutateAsync(form)
      setCreated(res.user.email)
      setForm({ email: '', display_name: '', password: '', role: 'viewer' })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'สร้างบัญชีไม่สำเร็จ')
    }
  }

  return (
    <Panel title="เพิ่มผู้ใช้ใหม่" className="mt-4">
      <form onSubmit={submit} className="grid gap-4 p-5 md:grid-cols-4">
        <Input
          label="อีเมล"
          type="email"
          required
          value={form.email}
          onChange={(e) => setForm({ ...form, email: e.target.value })}
        />
        <Input
          label="ชื่อที่แสดง"
          required
          value={form.display_name}
          onChange={(e) => setForm({ ...form, display_name: e.target.value })}
        />
        <Input
          label="รหัสผ่านเริ่มต้น"
          type="password"
          required
          minLength={12}
          hint="อย่างน้อย 12 ตัวอักษร"
          value={form.password}
          onChange={(e) => setForm({ ...form, password: e.target.value })}
        />
        <Select
          label="สิทธิ์"
          value={form.role}
          onChange={(e) => setForm({ ...form, role: e.target.value as Role })}
        >
          <option value="viewer">ผู้ใช้งาน — ดูและอัปโหลดได้</option>
          <option value="admin">ผู้ดูแลระบบ — จัดการทุกอย่าง</option>
        </Select>

        <div className="md:col-span-4 flex flex-wrap items-center gap-3">
          <Button type="submit" variant="primary" icon={UserPlus} loading={create.isPending}>
            สร้างบัญชี
          </Button>
          {created && (
            <span className="flex items-center gap-1.5 text-[13px] text-good">
              <Icon icon={Check} size={14} />
              สร้างบัญชี {created} แล้ว — แจ้งรหัสผ่านให้เจ้าตัวเปลี่ยนทันทีที่เข้าใช้ครั้งแรก
            </span>
          )}
          {error && (
            <span role="alert" className="text-[13px] text-bad">
              {error}
            </span>
          )}
        </div>
      </form>
    </Panel>
  )
}

function ResetPasswordDialog({
  user, busy, onClose, onSubmit,
}: {
  user: User
  busy: boolean
  onClose: () => void
  onSubmit: (password: string) => Promise<void>
}) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-ink/40 p-5">
      <div className="w-full max-w-[420px] rounded-xl border border-line bg-paper p-5 shadow-panel">
        <h3 className="font-display text-[15px] font-semibold">ตั้งรหัสผ่านใหม่</h3>
        <p className="mt-1 text-[13px] text-muted">
          สำหรับ {user.display_name} ({user.email})
        </p>
        <p className="mt-2 rounded-lg bg-warn-wash px-3 py-2 text-[12.5px] text-accent-ink">
          การตั้งรหัสใหม่จะทำให้ผู้ใช้คนนี้ถูกออกจากระบบทุกอุปกรณ์ทันที
        </p>

        <form
          className="mt-4 flex flex-col gap-3"
          onSubmit={async (e) => {
            e.preventDefault()
            setError(null)
            try {
              await onSubmit(password)
            } catch (err) {
              setError(err instanceof ApiError ? err.message : 'ตั้งรหัสผ่านใหม่ไม่สำเร็จ')
            }
          }}
        >
          <Input
            label="รหัสผ่านใหม่"
            type="password"
            required
            minLength={12}
            autoFocus
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          {error && (
            <span role="alert" className="text-[13px] text-bad">
              {error}
            </span>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" onClick={onClose}>
              ยกเลิก
            </Button>
            <Button type="submit" variant="primary" loading={busy}>
              ตั้งรหัสผ่าน
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}

/* ---------- ภาคของศูนย์กระจายสินค้า ---------- */

function RegionsSection() {
  const regions = useRegions()
  const setRegion = useSetRegion()
  const [saved, setSaved] = useState<string | null>(null)

  const cols: Column<RegionRow>[] = [
    { key: 'code', header: 'รหัส', width: '100px', render: (r) => <span className="tnum">{r.code}</span> },
    { key: 'name', header: 'ศูนย์กระจายสินค้า', render: (r) => <span className="font-medium">{r.name}</span> },
    {
      key: 'status',
      header: 'สถานะ',
      width: '110px',
      render: (r) => <Pill tone={r.active ? 'good' : 'neutral'}>{r.active ? 'Active' : 'ปิดแล้ว'}</Pill>,
    },
    {
      key: 'region',
      header: 'ภาค',
      width: '190px',
      render: (r) => (
        <select
          value={r.region}
          disabled={setRegion.isPending}
          onChange={(e) => {
            setRegion.mutate({ code: r.code, region: e.target.value })
            setSaved(r.code)
          }}
          className="w-full cursor-pointer rounded-lg border border-line bg-paper px-2.5 py-1.5 text-[13px]"
        >
          {!r.region && <option value="">— ยังไม่ระบุ —</option>}
          {regions.data?.options.map((o) => (
            <option key={o} value={o}>
              ภาค{o}
            </option>
          ))}
        </select>
      ),
    },
    {
      key: 'saved',
      header: '',
      width: '90px',
      render: (r) =>
        saved === r.code && !setRegion.isPending ? (
          <span className="flex items-center gap-1 text-[12px] text-good">
            <Icon icon={Check} size={13} />
            บันทึกแล้ว
          </span>
        ) : null,
    },
  ]

  return (
    <>
      <SectionHead
        title="ภาคของศูนย์กระจายสินค้า"
        note="ใช้กับแผนที่และกราฟสัดส่วนรายภาคในหน้าวางแผน"
      />
      <p className="mb-3 flex items-start gap-2 rounded-lg bg-info-wash px-3.5 py-2.5 text-[12.5px] text-info">
        <Icon icon={MapPin} size={14} className="mt-0.5 shrink-0" />
        ไฟล์ Excel ไม่มีคอลัมน์ภาค ค่าที่ตั้งไว้ที่นี่จึงถูกใช้ข้ามทุกชุดข้อมูล
        และไม่หายไปเมื่ออัปโหลดไฟล์ใหม่
      </p>

      {regions.isPending && <Spinner label="กำลังโหลดรายชื่อศูนย์" />}
      {regions.isError && (
        <ErrorState message="โหลดรายชื่อศูนย์ไม่สำเร็จ" retry={() => regions.refetch()} />
      )}
      {regions.data && (
        <Panel>
          <DataTable columns={cols} rows={regions.data.regions} rowKey={(r) => r.code} />
        </Panel>
      )}
    </>
  )
}

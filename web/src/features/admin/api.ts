import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../../lib/api'
import type { Role, User } from '../../types/api'

export interface RegionRow {
  code: string
  name: string
  region: string
  active: boolean
}

export function useUsers() {
  return useQuery({
    queryKey: ['admin', 'users'],
    queryFn: () => api.get<{ users: User[] }>('/admin/users'),
    select: (d) => d.users,
  })
}

export function useRegions() {
  return useQuery({
    queryKey: ['admin', 'regions'],
    queryFn: () => api.get<{ regions: RegionRow[]; options: string[] }>('/admin/regions'),
  })
}

export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { email: string; display_name: string; password: string; role: Role }) =>
      api.post<{ user: User }>('/admin/users', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

export function useSetActive() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) =>
      api.post(`/admin/users/${id}/active`, { active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

export function useResetPassword() {
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      api.post(`/admin/users/${id}/password`, { password }),
  })
}

export function useSetRegion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ code, region }: { code: string; region: string }) =>
      api.post(`/admin/regions/${encodeURIComponent(code)}`, { region }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'regions'] })
      // ภาคมีผลกับแผนที่และกราฟสัดส่วน จึงต้องล้างผลลัพธ์ที่แคชไว้ของ dashboard ด้วย
      qc.invalidateQueries({ queryKey: ['dashboard'] })
      qc.invalidateQueries({ queryKey: ['filters'] })
    },
  })
}

export function useChangeOwnPassword() {
  return useMutation({
    mutationFn: (body: { current_password: string; new_password: string }) =>
      api.post('/me/password', body),
  })
}

import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, buildQuery } from '../../lib/api'
import type { FilterOptions } from '../../types/api'

export interface Scope {
  dataset: string
  dc: string
  year: string
  month: string
}

/**
 * ฟิลเตอร์ทั้งหมดอยู่ใน URL ไม่ใช่ใน state ของ component
 *
 * ทำให้ส่งลิงก์มุมมองเดียวกันให้กันได้ ปุ่มย้อนกลับของเบราว์เซอร์ทำงานถูกต้อง
 * และการรีเฟรชหน้าไม่ทำให้ฟิลเตอร์หาย
 */
export function useScope() {
  const [params, setParams] = useSearchParams()

  const scope = useMemo<Scope>(
    () => ({
      dataset: params.get('dataset') ?? '',
      dc: params.get('dc') ?? 'ALL',
      year: params.get('year') ?? 'ALL',
      month: params.get('month') ?? 'ALL',
    }),
    [params],
  )

  const setScope = useCallback(
    (patch: Partial<Scope>) => {
      const next = new URLSearchParams(params)
      for (const [k, v] of Object.entries(patch)) {
        if (!v || v === 'ALL') {
          if (k === 'dataset') continue
          next.delete(k)
        } else {
          next.set(k, v)
        }
      }
      setParams(next, { replace: true })
    },
    [params, setParams],
  )

  /** query string ที่ส่งไป API — ค่าที่เป็น ALL ถูกส่งไปตรง ๆ เพราะ backend เข้าใจอยู่แล้ว */
  const query = useMemo(
    () => buildQuery({ dataset_id: scope.dataset, dc: scope.dc, year: scope.year, month: scope.month }),
    [scope],
  )

  return { scope, setScope, query }
}

export function useFilters(datasetId: string) {
  return useQuery({
    queryKey: ['filters', datasetId],
    enabled: !!datasetId,
    queryFn: () => api.get<FilterOptions>(`/datasets/${datasetId}/filters`),
    staleTime: Infinity,
  })
}

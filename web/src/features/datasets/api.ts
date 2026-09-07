import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, buildQuery, request } from '../../lib/api'
import type { Dataset, ImportIssue, ImportSummary, UploadResult } from '../../types/api'

export const datasetKeys = {
  list: (deleted: boolean) => ['datasets', { deleted }] as const,
  detail: (id: string) => ['dataset', id] as const,
  issues: (id: string) => ['dataset-issues', id] as const,
}

export function useDatasets(deleted = false) {
  return useQuery({
    queryKey: datasetKeys.list(deleted),
    queryFn: () =>
      api.get<{ datasets: Dataset[] }>(`/datasets${buildQuery({ deleted: deleted ? 'true' : undefined })}`),
    select: (d) => d.datasets,
  })
}

/**
 * ติดตามสถานะการนำเข้าโดยถามซ้ำทุก 1.5 วินาที และหยุดเองเมื่อเสร็จหรือล้มเหลว
 * การนำเข้าไฟล์ขนาดเทมเพลตจริงใช้เวลาราวสองวินาที จึงไม่ต้องพึ่ง websocket
 */
export function useDatasetDetail(id: string | null, poll = false) {
  return useQuery({
    queryKey: datasetKeys.detail(id ?? ''),
    enabled: !!id,
    queryFn: () => api.get<{ dataset: Dataset; import: ImportSummary }>(`/datasets/${id}`),
    refetchInterval: (query) => {
      if (!poll) return false
      return query.state.data?.dataset.status === 'processing' ? 1500 : false
    },
  })
}

export function useDatasetIssues(id: string | null) {
  return useQuery({
    queryKey: datasetKeys.issues(id ?? ''),
    enabled: !!id,
    queryFn: () => api.get<{ issues: ImportIssue[]; total: number }>(`/datasets/${id}/issues?limit=200`),
  })
}

export function useUploadDataset() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ file, label }: { file: File; label: string }) => {
      const form = new FormData()
      form.append('file', file)
      if (label) form.append('label', label)
      return request<UploadResult>('/datasets', { method: 'POST', body: form })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['datasets'] }),
  })
}

export function useDeleteDataset() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.del(`/datasets/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['datasets'] }),
  })
}

export function useRestoreDataset() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post(`/datasets/${id}/restore`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['datasets'] }),
  })
}

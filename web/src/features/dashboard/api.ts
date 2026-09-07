import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import type { PlanningPage, SellInPage, SellOutPage, StockPage } from '../../types/api'

function usePage<T>(name: string, query: string, enabled: boolean, extra = '') {
  return useQuery({
    queryKey: ['dashboard', name, query + extra],
    enabled,
    queryFn: () => api.get<T>(`/dashboard/${name}${query}${extra}`),
  })
}

export const useSellIn = (query: string, enabled: boolean) =>
  usePage<SellInPage>('sellin', query, enabled)

export const useSellOut = (query: string, enabled: boolean, group: string) =>
  usePage<SellOutPage>('sellout', query, enabled, group && group !== 'ALL' ? `&group=${encodeURIComponent(group)}` : '')

export const useStock = (query: string, enabled: boolean, group: string, status: string) =>
  usePage<StockPage>(
    'stock', query, enabled,
    (group && group !== 'ALL' ? `&group=${encodeURIComponent(group)}` : '') +
    (status && status !== 'ALL' ? `&status=${encodeURIComponent(status)}` : ''),
  )

export const usePlanning = (query: string, enabled: boolean) =>
  usePage<PlanningPage>('planning', query, enabled)

// Typed TanStack Query hooks shared by every module page. Each module is
// identified by its API path segment (e.g. "applications", "study-sessions").
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from '@tanstack/react-query'
import { api, buildQuery } from './client'
import type { BaseRecord, RecordInput } from './types'

interface ListResponse<T> {
  items: T[]
}

// useListQuery fetches a module's rows with search/filter params.
export function useListQuery<T extends BaseRecord>(
  module: string,
  params: Record<string, string> = {},
): UseQueryResult<T[]> {
  return useQuery({
    queryKey: [module, params],
    queryFn: async () => {
      const res = await api.get<ListResponse<T>>(`/api/v1/${module}${buildQuery(params)}`)
      return res.items
    },
    placeholderData: (prev) => prev, // keep rows visible while a search refetches
  })
}

// useCreateMutation posts a new record and refreshes the module's lists.
export function useCreateMutation<T extends BaseRecord>(module: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: RecordInput<T>) => api.post<T>(`/api/v1/${module}`, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [module] }),
  })
}

// useUpdateMutation rewrites an existing record.
export function useUpdateMutation<T extends BaseRecord>(module: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: RecordInput<T> }) =>
      api.put<T>(`/api/v1/${module}/${id}`, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: [module] }),
  })
}

// useDeleteMutation removes a record.
export function useDeleteMutation(module: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete<{ status: string }>(`/api/v1/${module}/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: [module] }),
  })
}

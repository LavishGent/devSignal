// Fixed-choice option lists (statuses, categories, stages). They are served
// by GET /api/v1/options from the lookup_values table, so a new option added
// in the database appears in every dropdown without a frontend change.
import { useQuery } from '@tanstack/react-query'
import { api } from './client'

export interface Option {
  value: string
  label: string
}

// Keys match the backend lookup kinds.
export type OptionKind =
  | 'application_status'
  | 'work_mode'
  | 'contact_relationship'
  | 'interview_stage'
  | 'interview_outcome'
  | 'skill_category'
  | 'certification_status'
  | 'project_status'
  | 'goal_category'
  | 'goal_status'

export type Options = Record<OptionKind, Option[]>

const EMPTY: Option[] = []

// useOptions returns the option list for one kind. The full set is fetched
// once per session and shared by every page through the query cache; an
// empty list is returned while it loads so selects render without flicker.
export function useOptions(kind: OptionKind): Option[] {
  const { data } = useQuery({
    queryKey: ['options'],
    queryFn: () => api.get<Options>('/api/v1/options'),
    staleTime: 5 * 60 * 1000,
  })
  return data?.[kind] ?? EMPTY
}

// Types and query hooks for the cross-module endpoints (dashboard,
// recommendations, reports, global search).
import { useQuery } from '@tanstack/react-query'
import { api } from './client'

export interface DayMinutes {
  date: string
  minutes: number
}

export interface UpcomingInterview {
  id: string
  company: string
  position: string
  stage: string
  scheduledAt: string
}

export interface SkillGap {
  id: string
  name: string
  proficiency: number
  target: number
  gap: number
}

export interface DashboardSummary {
  generatedAt: string
  totalApplications: number
  applicationsByStatus: Record<string, number>
  upcomingInterviews: UpcomingInterview[]
  studyMinutesLast30: number
  studyByDay: DayMinutes[]
  topSkillGaps: SkillGap[]
  openGoals: number
  overdueGoals: number
  activeProjects: number
  earnedCertifications: number
  expiringCertifications: number
}

export interface Recommendation {
  rule: string
  priority: number
  title: string
  detail: string
  module: string
  recordId?: string
}

export interface ReportInfo {
  id: string
  title: string
  description: string
}

export interface ReportData {
  id: string
  title: string
  generatedAt: string
  columns: { key: string; label: string }[]
  rows: string[][]
}

export interface SearchItem {
  id: string
  title: string
  subtitle: string
  updatedAt: string
}

export interface SearchGroup {
  module: string
  label: string
  items: SearchItem[]
}

export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api.get<DashboardSummary>('/api/v1/dashboard'),
  })
}

export function useRecommendations(limit?: number) {
  return useQuery({
    queryKey: ['recommendations', limit ?? 0],
    queryFn: async () => {
      const qs = limit ? `?limit=${limit}` : ''
      const res = await api.get<{ items: Recommendation[] }>(`/api/v1/recommendations${qs}`)
      return res.items
    },
  })
}

export function useReportCatalog() {
  return useQuery({
    queryKey: ['reports'],
    queryFn: async () => {
      const res = await api.get<{ items: ReportInfo[] }>('/api/v1/reports')
      return res.items
    },
  })
}

export function useReport(id: string | null) {
  return useQuery({
    queryKey: ['reports', id],
    queryFn: () => api.get<ReportData>(`/api/v1/reports/${id}`),
    enabled: id !== null,
  })
}

export function useGlobalSearch(q: string) {
  return useQuery({
    queryKey: ['global-search', q],
    queryFn: async () => {
      const res = await api.get<{ query: string; groups: SearchGroup[] }>(
        `/api/v1/search?q=${encodeURIComponent(q)}`,
      )
      return res.groups
    },
    enabled: q.trim().length > 0,
    placeholderData: (prev) => prev,
  })
}

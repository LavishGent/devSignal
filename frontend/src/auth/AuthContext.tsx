// Session-aware auth state. The session itself is an HttpOnly cookie; the
// SPA only tracks whether /auth/me succeeds.
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '../api/client'
import type { User } from '../api/types'

interface AuthState {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, displayName: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const qc = useQueryClient()

  useEffect(() => {
    api
      .get<User>('/api/v1/auth/me')
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const u = await api.post<User>('/api/v1/auth/login', { email, password })
    setUser(u)
  }, [])

  const register = useCallback(
    async (email: string, displayName: string, password: string) => {
      const u = await api.post<User>('/api/v1/auth/register', { email, displayName, password })
      setUser(u)
    },
    [],
  )

  const logout = useCallback(async () => {
    try {
      await api.post('/api/v1/auth/logout')
    } catch (err) {
      if (!(err instanceof ApiError)) throw err
      // Revocation failing server-side still ends the local session.
    }
    setUser(null)
    qc.clear() // drop all cached career data on logout
  }, [qc])

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

// useAuth exposes the auth state, using it outside AuthProvider is a bug.
export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}

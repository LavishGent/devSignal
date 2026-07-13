import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import type { ReactNode } from 'react'
import { AuthProvider, useAuth } from './auth/AuthContext'
import Layout from './components/Layout'
import Login from './pages/Login'
import Register from './pages/Register'
import Applications from './pages/Applications'
import Contacts from './pages/Contacts'
import Interviews from './pages/Interviews'
import Skills from './pages/Skills'
import StudySessions from './pages/StudySessions'
import Certifications from './pages/Certifications'
import Projects from './pages/Projects'
import Goals from './pages/Goals'
import Dashboard from './pages/Dashboard'
import Reports from './pages/Reports'
import Search from './pages/Search'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 15_000 },
  },
})

// RequireAuth gates the app shell behind a live session.
function RequireAuth({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="loading">Loading…</div>
  if (!user) return <Navigate to="/login" replace />
  return children
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route
              element={
                <RequireAuth>
                  <Layout />
                </RequireAuth>
              }
            >
              <Route path="/" element={<Dashboard />} />
              <Route path="/applications" element={<Applications />} />
              <Route path="/contacts" element={<Contacts />} />
              <Route path="/interviews" element={<Interviews />} />
              <Route path="/skills" element={<Skills />} />
              <Route path="/study-sessions" element={<StudySessions />} />
              <Route path="/certifications" element={<Certifications />} />
              <Route path="/projects" element={<Projects />} />
              <Route path="/goals" element={<Goals />} />
              <Route path="/reports" element={<Reports />} />
              <Route path="/search" element={<Search />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  )
}

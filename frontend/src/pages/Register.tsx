import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiError } from '../api/client'
import { useAuth } from '../auth/AuthContext'
import FormField from '../components/FormField'

// Register creates an account; validation mirrors the server's rules and
// server field errors are surfaced per field.
export default function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!email.trim()) errs.email = 'Email is required'
    if (!displayName.trim()) errs.displayName = 'Display name is required'
    if (password.length < 8) errs.password = 'Password must be at least 8 characters'
    setFieldErrors(errs)
    return Object.keys(errs).length === 0
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (!validate()) return
    setBusy(true)
    try {
      await register(email, displayName, password)
      navigate('/')
    } catch (err) {
      if (err instanceof ApiError) {
        setFieldErrors(err.fields)
        if (Object.keys(err.fields).length === 0) setError(err.message)
      } else {
        setError('Something went wrong; try again.')
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-wrap">
      <div className="card auth-card">
        <div className="auth-logo">
          <span className="logo">DS</span> DevSignal
        </div>
        <h1>Create your account</h1>
        <p className="auth-sub">Track applications, skills, and goals in one place.</p>
        {error ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{error}</div> : null}
        <form onSubmit={onSubmit}>
          <FormField label="Email" error={fieldErrors.email}>
            <input
              className="input"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              aria-invalid={!!fieldErrors.email}
            />
          </FormField>
          <FormField label="Display name" error={fieldErrors.displayName}>
            <input
              className="input"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              autoComplete="name"
              aria-invalid={!!fieldErrors.displayName}
            />
          </FormField>
          <FormField label="Password" error={fieldErrors.password} hint="At least 8 characters.">
            <input
              className="input"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
              aria-invalid={!!fieldErrors.password}
            />
          </FormField>
          <button className="btn btn-primary" type="submit" disabled={busy}>
            {busy ? 'Creating account…' : 'Create account'}
          </button>
        </form>
        <div className="auth-alt">
          Already have an account? <Link to="/login">Sign in</Link>
        </div>
      </div>
    </div>
  )
}

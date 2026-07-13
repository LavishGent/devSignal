import type { ReactNode } from 'react'

interface FormFieldProps {
  label: string
  error?: string
  hint?: string
  span2?: boolean
  children: ReactNode
}

// FormField pairs a label with a control and shows its validation problem.
export default function FormField({ label, error, hint, span2, children }: FormFieldProps) {
  return (
    <label className={'field' + (span2 ? ' span-2' : '')}>
      <span className="field-label">{label}</span>
      {children}
      {error ? <span className="field-error">{error}</span> : null}
      {!error && hint ? <span className="field-hint">{hint}</span> : null}
    </label>
  )
}

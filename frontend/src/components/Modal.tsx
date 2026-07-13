import { useEffect, type ReactNode } from 'react'

interface ModalProps {
  title: string
  small?: boolean
  onClose: () => void
  children: ReactNode
}

// Modal centers content over a dimmed backdrop; Escape or backdrop-click closes.
export default function Modal({ title, small, onClose, children }: ModalProps) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div
      className="modal-overlay"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className={'modal' + (small ? ' modal-sm' : '')} role="dialog" aria-modal="true" aria-label={title}>
        <h2 className="modal-title">{title}</h2>
        {children}
      </div>
    </div>
  )
}

import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { Icon, type IconName } from '@/components/Icon'

type ToastType = 'success' | 'error' | 'warning' | 'info'
type Toast = { id: number; type: ToastType; message: string }

const ToastContext = createContext<(type: ToastType, message: string) => void>(() => {})

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const idRef = useRef(0)

  const push = useCallback((type: ToastType, message: string) => {
    const id = ++idRef.current
    setToasts((prev) => [...prev, { id, type, message }])
    window.setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, 5000)
  }, [])

  const value = useMemo(() => push, [push])

  return (
    <ToastContext.Provider value={value}>
      {children}
      {createPortal(
        <div className="toast-container" role="region" aria-label="notifications">
          {toasts.map((t) => (
            <div key={t.id} className={`toast toast-${t.type}`} role="status">
              <Icon name={toastIcon(t.type)} size={16} className="toast-icon" />
              <span>{t.message}</span>
            </div>
          ))}
        </div>,
        document.body,
      )}
    </ToastContext.Provider>
  )
}

function toastIcon(type: ToastType): IconName {
  switch (type) {
    case 'success':
      return 'checkCircle'
    case 'error':
      return 'xCircle'
    case 'warning':
      return 'alertTriangle'
    default:
      return 'info'
  }
}

export function useToast() {
  return useContext(ToastContext)
}

export function useToastApi() {
  const push = useToast()
  return useMemo(
    () => ({
      success: (m: string) => push('success', m),
      error: (m: string) => push('error', m),
      warning: (m: string) => push('warning', m),
      info: (m: string) => push('info', m),
    }),
    [push],
  )
}

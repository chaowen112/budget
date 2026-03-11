import { useEffect, useState, useCallback, createContext, useContext, type ReactNode } from 'react'
import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react'

type ToastVariant = 'success' | 'error' | 'info' | 'warning'

interface ToastItem {
  id: string
  message: string
  variant: ToastVariant
}

interface ToastContextValue {
  toast: (message: string, variant?: ToastVariant) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

const ICON_MAP: Record<ToastVariant, typeof Info> = {
  success: CheckCircle2,
  error: AlertCircle,
  info: Info,
  warning: AlertCircle,
}

const STYLE_MAP: Record<ToastVariant, string> = {
  success:
    'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300',
  error:
    'border-red-200 bg-red-50 text-red-800 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300',
  info:
    'border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-300',
  warning:
    'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300',
}

function ToastMessage({ item, onDismiss }: { item: ToastItem; onDismiss: (id: string) => void }) {
  const Icon = ICON_MAP[item.variant]

  useEffect(() => {
    const timer = setTimeout(() => onDismiss(item.id), 4000)
    return () => clearTimeout(timer)
  }, [item.id, onDismiss])

  return (
    <div
      className={`flex items-start gap-2.5 px-4 py-3 rounded-xl border shadow-lg text-sm animate-in slide-in-from-right fade-in duration-200 ${STYLE_MAP[item.variant]}`}
    >
      <Icon className="h-4 w-4 flex-shrink-0 mt-0.5" />
      <span className="flex-1">{item.message}</span>
      <button
        onClick={() => onDismiss(item.id)}
        className="h-5 w-5 flex items-center justify-center rounded opacity-60 hover:opacity-100 transition-opacity"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

let counter = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const toast = useCallback((message: string, variant: ToastVariant = 'error') => {
    const id = `toast-${++counter}`
    setToasts((prev) => [...prev, { id, message, variant }])
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed top-4 right-4 z-[60] flex flex-col gap-2 max-w-sm pointer-events-auto">
        {toasts.map((item) => (
          <ToastMessage key={item.id} item={item} onDismiss={dismiss} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}

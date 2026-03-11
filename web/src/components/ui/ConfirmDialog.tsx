import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'
import { Button } from './Button'

interface ConfirmOptions {
  title?: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'danger' | 'default'
}

interface ConfirmContextValue {
  confirm: (options: ConfirmOptions) => Promise<boolean>
}

const ConfirmContext = createContext<ConfirmContextValue | null>(null)

export function useConfirm() {
  const ctx = useContext(ConfirmContext)
  if (!ctx) throw new Error('useConfirm must be used within ConfirmProvider')
  return ctx
}

interface PromiseRef {
  resolve: (value: boolean) => void
  options: ConfirmOptions
}

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [pending, setPending] = useState<PromiseRef | null>(null)

  const confirm = useCallback((options: ConfirmOptions): Promise<boolean> => {
    return new Promise<boolean>((resolve) => {
      setPending({ resolve, options })
    })
  }, [])

  const handleClose = (result: boolean) => {
    pending?.resolve(result)
    setPending(null)
  }

  const opts = pending?.options
  const isDanger = opts?.variant === 'danger'

  return (
    <ConfirmContext.Provider value={{ confirm }}>
      {children}
      {opts && (
        <div className="fixed inset-0 z-[70] flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/40 dark:bg-black/60 backdrop-blur-sm animate-in fade-in duration-150"
            onClick={() => handleClose(false)}
          />
          <div className="relative z-10 w-full max-w-sm bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl shadow-2xl animate-in fade-in zoom-in-95 duration-150">
            <div className="px-6 py-5 space-y-3">
              <div className="flex items-start gap-3">
                <div
                  className={`h-9 w-9 rounded-xl flex items-center justify-center flex-shrink-0 ${
                    isDanger
                      ? 'bg-red-50 dark:bg-red-500/10'
                      : 'bg-amber-50 dark:bg-amber-500/10'
                  }`}
                >
                  <AlertTriangle
                    className={`h-4.5 w-4.5 ${
                      isDanger ? 'text-red-500 dark:text-red-400' : 'text-amber-500 dark:text-amber-400'
                    }`}
                  />
                </div>
                <div>
                  <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-50">
                    {opts.title || 'Confirm'}
                  </h3>
                  <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-1">{opts.message}</p>
                </div>
              </div>
            </div>
            <div className="px-6 py-4 border-t border-zinc-200 dark:border-zinc-800 flex justify-end gap-2">
              <Button variant="secondary" size="sm" onClick={() => handleClose(false)}>
                {opts.cancelLabel || 'Cancel'}
              </Button>
              <Button
                variant={isDanger ? 'danger' : 'primary'}
                size="sm"
                onClick={() => handleClose(true)}
              >
                {opts.confirmLabel || 'Confirm'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </ConfirmContext.Provider>
  )
}

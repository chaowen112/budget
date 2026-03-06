import type { ReactNode } from 'react'
import { X } from 'lucide-react'

interface ModalProps {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
  /** Optional footer rendered below children */
  footer?: ReactNode
  size?: 'sm' | 'md' | 'lg'
}

const sizeMap = { sm: 'max-w-sm', md: 'max-w-md', lg: 'max-w-lg' }

export function Modal({ open, onClose, title, children, footer, size = 'md' }: ModalProps) {
  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 dark:bg-black/60 backdrop-blur-sm animate-in fade-in duration-150"
        onClick={onClose}
      />

      {/* Panel */}
      <div
        className={`
          relative z-10 w-full ${sizeMap[size]}
          bg-white dark:bg-zinc-900
          border border-zinc-200 dark:border-zinc-800
          rounded-2xl shadow-2xl
          animate-in fade-in zoom-in-95 duration-150
        `}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-zinc-200 dark:border-zinc-800">
          <h2 className="text-base font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
            {title}
          </h2>
          <button
            onClick={onClose}
            className="
              h-8 w-8 flex items-center justify-center rounded-lg
              text-zinc-400 dark:text-zinc-500
              hover:text-zinc-600 dark:hover:text-zinc-300
              hover:bg-zinc-100 dark:hover:bg-zinc-800
              transition-colors duration-150
            "
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 max-h-[70vh] overflow-y-auto">
          {children}
        </div>

        {/* Footer */}
        {footer && (
          <div className="px-6 py-4 border-t border-zinc-200 dark:border-zinc-800">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}

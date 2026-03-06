type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info' | 'violet'

interface BadgeProps {
  children: React.ReactNode
  variant?: BadgeVariant
  className?: string
}

const variantMap: Record<BadgeVariant, string> = {
  default: 'bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300',
  success: 'bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
  warning: 'bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400',
  danger:  'bg-red-50 dark:bg-red-500/10 text-red-700 dark:text-red-400',
  info:    'bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400',
  violet:  'bg-violet-50 dark:bg-violet-500/10 text-violet-700 dark:text-violet-400',
}

export function Badge({ children, variant = 'default', className = '' }: BadgeProps) {
  return (
    <span
      className={`
        inline-flex items-center gap-1
        px-2 py-0.5 rounded-md
        text-xs font-medium
        ${variantMap[variant]}
        ${className}
      `}
    >
      {children}
    </span>
  )
}

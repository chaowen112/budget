interface ProgressBarProps {
  value: number      // 0-100
  max?: number
  variant?: 'default' | 'success' | 'warning' | 'danger'
  size?: 'sm' | 'md'
  animated?: boolean
  className?: string
}

const variantMap = {
  default: 'bg-violet-500',
  success: 'bg-emerald-500',
  warning: 'bg-amber-500',
  danger:  'bg-red-500',
}

const sizeMap = { sm: 'h-1.5', md: 'h-2' }

export function ProgressBar({
  value,
  max = 100,
  variant = 'default',
  size = 'sm',
  animated = false,
  className = '',
}: ProgressBarProps) {
  const pct = Math.min((value / max) * 100, 100)

  return (
    <div
      className={`w-full bg-zinc-200 dark:bg-zinc-700/60 rounded-full overflow-hidden ${sizeMap[size]} ${className}`}
    >
      <div
        className={`h-full rounded-full transition-all duration-500 ${variantMap[variant]} ${animated ? 'animate-pulse' : ''}`}
        style={{ width: `${pct}%` }}
      />
    </div>
  )
}

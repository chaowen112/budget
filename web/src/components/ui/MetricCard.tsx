import type { LucideIcon } from 'lucide-react'

interface MetricCardProps {
  label: string
  value: string
  icon: LucideIcon
  /** e.g. "+12.3% savings rate" */
  subtext?: string
  subtextPositive?: boolean
  /** Color accent for the icon background */
  accent?: 'emerald' | 'red' | 'violet' | 'blue' | 'amber' | 'zinc'
  className?: string
}

const accentMap: Record<NonNullable<MetricCardProps['accent']>, { bg: string; icon: string }> = {
  emerald: { bg: 'bg-emerald-500/10 dark:bg-emerald-500/10', icon: 'text-emerald-500' },
  red:     { bg: 'bg-red-500/10 dark:bg-red-500/10',         icon: 'text-red-500' },
  violet:  { bg: 'bg-violet-500/10 dark:bg-violet-500/10',   icon: 'text-violet-500' },
  blue:    { bg: 'bg-blue-500/10 dark:bg-blue-500/10',       icon: 'text-blue-500' },
  amber:   { bg: 'bg-amber-500/10 dark:bg-amber-500/10',     icon: 'text-amber-500' },
  zinc:    { bg: 'bg-zinc-200 dark:bg-zinc-800',             icon: 'text-zinc-500 dark:text-zinc-400' },
}

export function MetricCard({
  label,
  value,
  icon: Icon,
  subtext,
  subtextPositive,
  accent = 'zinc',
  className = '',
}: MetricCardProps) {
  const colors = accentMap[accent]

  return (
    <div
      className={`
        bg-white dark:bg-zinc-900
        border border-zinc-200 dark:border-zinc-800
        rounded-2xl p-6 shadow-sm
        transition-all duration-200
        hover:border-zinc-300 dark:hover:border-zinc-700
        hover:shadow-md hover:-translate-y-0.5
        ${className}
      `}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium tracking-wide text-zinc-500 dark:text-zinc-400 uppercase mb-2">
            {label}
          </p>
          <p className="text-2xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50 truncate">
            {value}
          </p>
          {subtext && (
            <p
              className={`mt-1.5 text-xs font-medium flex items-center gap-0.5 ${
                subtextPositive === undefined
                  ? 'text-zinc-500 dark:text-zinc-400'
                  : subtextPositive
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : 'text-red-600 dark:text-red-400'
              }`}
            >
              {subtext}
            </p>
          )}
        </div>
        <div className={`h-10 w-10 flex-shrink-0 rounded-xl flex items-center justify-center ${colors.bg}`}>
          <Icon className={`h-5 w-5 ${colors.icon}`} />
        </div>
      </div>
    </div>
  )
}

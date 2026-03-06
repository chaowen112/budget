import type { ReactNode } from 'react'

interface BentoCardProps {
  children: ReactNode
  className?: string
  colSpan?: 1 | 2 | 3
  rowSpan?: 1 | 2
  hover?: boolean
  /** Render as a plain div with no background — for when you need full custom control */
  bare?: boolean
}

export function BentoCard({
  children,
  className = '',
  colSpan = 1,
  rowSpan = 1,
  hover = true,
  bare = false,
}: BentoCardProps) {
  const colClass =
    colSpan === 2 ? 'col-span-1 md:col-span-2' : colSpan === 3 ? 'col-span-1 md:col-span-3' : 'col-span-1'
  const rowClass = rowSpan === 2 ? 'row-span-2' : ''
  const baseClass = bare
    ? ''
    : 'bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl shadow-sm'
  const hoverClass =
    hover && !bare
      ? 'transition-all duration-200 hover:border-zinc-300 dark:hover:border-zinc-700 hover:shadow-md hover:-translate-y-0.5'
      : ''

  return (
    <div className={`${colClass} ${rowClass} ${baseClass} ${hoverClass} ${className}`}>
      {children}
    </div>
  )
}

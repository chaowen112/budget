import type { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'

const baseInput = `
  w-full px-3 py-2 rounded-xl text-sm
  bg-zinc-50 dark:bg-zinc-800/60
  border border-zinc-200 dark:border-zinc-700
  text-zinc-900 dark:text-zinc-100
  placeholder:text-zinc-400 dark:placeholder:text-zinc-500
  focus:outline-none focus:ring-2 focus:ring-violet-500/40 focus:border-violet-500/60
  transition-colors duration-150
  disabled:opacity-50 disabled:cursor-not-allowed
`

interface LabelProps {
  children: React.ReactNode
  htmlFor?: string
  hint?: string
}

export function Label({ children, htmlFor, hint }: LabelProps) {
  return (
    <label htmlFor={htmlFor} className="flex items-center justify-between mb-1.5">
      <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">{children}</span>
      {hint && <span className="text-xs text-zinc-400 dark:text-zinc-500">{hint}</span>}
    </label>
  )
}

export function Input({ className = '', ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={`${baseInput} ${className}`} {...props} />
}

export function Select({
  className = '',
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={`${baseInput} ${className}`} {...props}>
      {children}
    </select>
  )
}

export function Textarea({ className = '', ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={`${baseInput} resize-none ${className}`} {...props} />
}

interface FormFieldProps {
  label: string
  htmlFor?: string
  hint?: string
  children: React.ReactNode
  className?: string
}

export function FormField({ label, htmlFor, hint, children, className = '' }: FormFieldProps) {
  return (
    <div className={`space-y-0 ${className}`}>
      <Label htmlFor={htmlFor} hint={hint}>{label}</Label>
      {children}
    </div>
  )
}

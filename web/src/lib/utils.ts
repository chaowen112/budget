import type { Money } from '../types'

export function formatMoney(money: Money | undefined | null): string {
  if (!money) return '$0.00'
  
  const amount = parseFloat(money.amount || '0')
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: money.currency || 'SGD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

export function moneyToNumber(money: Money | undefined | null): number {
  if (!money) return 0
  return parseFloat(money.amount || '0')
}

export function numberToMoney(amount: number, currency: string = 'SGD'): Money {
  return { amount: amount.toFixed(2), currency }
}

export function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

export function formatDateShort(dateString: string): string {
  const date = new Date(dateString)
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
  }).format(date)
}

export function getMonthName(month: number): string {
  const date = new Date(2000, month - 1)
  return new Intl.DateTimeFormat('en-US', { month: 'long' }).format(date)
}

export function classNames(...classes: (string | boolean | undefined)[]): string {
  return classes.filter(Boolean).join(' ')
}

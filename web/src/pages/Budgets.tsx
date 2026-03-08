import { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { budgetApi, categoryApi, transactionApi } from '../api'
import { formatDate, numberToMoney, moneyToNumber } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import { useCurrency } from '../store/CurrencyContext'
import type { Budget, Category, PeriodType, Transaction } from '../types'
import { Plus, Pencil, Trash2, List } from 'lucide-react'
import { Button, Modal, FormField, Input, Select, ProgressBar, Badge } from '../components/ui'

const CREATE_CATEGORY_OPTION = '__create_new_category__'

function getOrdinalSuffix(day: number) {
  if (day >= 11 && day <= 13) return 'th'
  const last = day % 10
  if (last === 1) return 'st'
  if (last === 2) return 'nd'
  if (last === 3) return 'rd'
  return 'th'
}

function addMonths(date: Date, months: number) {
  const d = new Date(date)
  const day = d.getDate()
  d.setDate(1)
  d.setMonth(d.getMonth() + months)
  const lastDay = new Date(d.getFullYear(), d.getMonth() + 1, 0).getDate()
  d.setDate(Math.min(day, lastDay))
  return d
}

function addYears(date: Date, years: number) {
  const d = new Date(date)
  const month = d.getMonth()
  const day = d.getDate()
  d.setFullYear(d.getFullYear() + years, month, 1)
  const lastDay = new Date(d.getFullYear(), month + 1, 0).getDate()
  d.setDate(Math.min(day, lastDay))
  return d
}

function startOfDay(date: Date) {
  const d = new Date(date)
  d.setHours(0, 0, 0, 0)
  return d
}

function toUtcStartOfDateISOString(date: Date) {
  return new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0, 0)).toISOString()
}

function toUtcEndOfDateISOString(date: Date) {
  return new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 59, 999)).toISOString()
}

function getCycleEndDate(startDate: Date, periodType: PeriodType, now: Date) {
  const start = startOfDay(startDate)
  const current = new Date(now)

  if (periodType === 'PERIOD_TYPE_WEEKLY') {
    const msPerDay = 1000 * 60 * 60 * 24
    const days = Math.max(0, Math.floor((startOfDay(current).getTime() - start.getTime()) / msPerDay))
    const cycles = Math.floor(days / 7)
    const cycleStart = new Date(start)
    cycleStart.setDate(start.getDate() + cycles * 7)
    return new Date(cycleStart.getTime() + (7 * msPerDay) - 1)
  }

  if (periodType === 'PERIOD_TYPE_MONTHLY') {
    let cycleStart = new Date(start)
    while (addMonths(cycleStart, 1) <= current) {
      cycleStart = addMonths(cycleStart, 1)
    }
    return new Date(addMonths(cycleStart, 1).getTime() - 1)
  }

  let cycleStart = new Date(start)
  while (addYears(cycleStart, 1) <= current) {
    cycleStart = addYears(cycleStart, 1)
  }
  return new Date(addYears(cycleStart, 1).getTime() - 1)
}

function getCycleStartDate(startDate: Date, periodType: PeriodType, now: Date) {
  const start = startOfDay(startDate)
  const current = new Date(now)

  if (periodType === 'PERIOD_TYPE_WEEKLY') {
    const msPerDay = 1000 * 60 * 60 * 24
    const days = Math.max(0, Math.floor((startOfDay(current).getTime() - start.getTime()) / msPerDay))
    const cycles = Math.floor(days / 7)
    const cycleStart = new Date(start)
    cycleStart.setDate(start.getDate() + cycles * 7)
    return cycleStart
  }

  if (periodType === 'PERIOD_TYPE_MONTHLY') {
    let cycleStart = new Date(start)
    while (addMonths(cycleStart, 1) <= current) {
      cycleStart = addMonths(cycleStart, 1)
    }
    return cycleStart
  }

  let cycleStart = new Date(start)
  while (addYears(cycleStart, 1) <= current) {
    cycleStart = addYears(cycleStart, 1)
  }
  return cycleStart
}

export default function Budgets() {
  const { user } = useAuth()
  const { formatConverted } = useCurrency()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingBudget, setEditingBudget] = useState<Budget | null>(null)
  const [activeBudgetForTransactions, setActiveBudgetForTransactions] = useState<Budget | null>(null)
  const [budgetCategoryId, setBudgetCategoryId] = useState('')
  const [showQuickCategoryForm, setShowQuickCategoryForm] = useState(false)
  const [quickCategoryName, setQuickCategoryName] = useState('')

  const { data: budgetStatuses, isLoading } = useQuery({
    queryKey: ['budgetStatuses'],
    queryFn: () => budgetApi.getAllStatuses(),
  })

  const { data: categories } = useQuery({
    queryKey: ['categories', 'expense'],
    queryFn: () => categoryApi.list('TRANSACTION_TYPE_EXPENSE'),
  })

  const activeBudgetDateRange = useMemo(() => {
    if (!activeBudgetForTransactions) return null
    const now = new Date()
    const startDate = new Date(activeBudgetForTransactions.startDate)
    const start = getCycleStartDate(startDate, activeBudgetForTransactions.periodType, now)
    const end = getCycleEndDate(startDate, activeBudgetForTransactions.periodType, now)
    return { start, end }
  }, [activeBudgetForTransactions])

  const { data: relatedTransactionsData, isLoading: isLoadingRelatedTransactions } = useQuery({
    queryKey: [
      'budgetRelatedTransactions',
      activeBudgetForTransactions?.id,
      activeBudgetDateRange?.start?.toISOString(),
      activeBudgetDateRange?.end?.toISOString(),
    ],
    enabled: !!activeBudgetForTransactions && !!activeBudgetDateRange,
    queryFn: () =>
      transactionApi.list({
        categoryId: activeBudgetForTransactions!.categoryId,
        type: 'TRANSACTION_TYPE_EXPENSE',
        startDate: toUtcStartOfDateISOString(activeBudgetDateRange!.start),
        endDate: toUtcEndOfDateISOString(activeBudgetDateRange!.end),
        pageSize: 200,
      }),
  })

  const createMutation = useMutation({
    mutationFn: budgetApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgetStatuses'] })
      setIsModalOpen(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof budgetApi.update>[1] }) =>
      budgetApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgetStatuses'] })
      setIsModalOpen(false)
      setEditingBudget(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: budgetApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgetStatuses'] })
    },
  })

  const createCategoryMutation = useMutation({
    mutationFn: categoryApi.create,
    onSuccess: (createdCategory) => {
      queryClient.setQueryData<Category[]>(['categories', 'expense'], (existing = []) => {
        if (existing.some((c) => c.id === createdCategory.id)) {
          return existing
        }
        return [...existing, createdCategory].sort((a, b) => a.name.localeCompare(b.name))
      })
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      queryClient.invalidateQueries({ queryKey: ['categories', 'expense'] })
      setBudgetCategoryId(createdCategory.id)
      setQuickCategoryName('')
      setShowQuickCategoryForm(false)
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const amount = parseFloat(formData.get('amount') as string)

    if (editingBudget) {
      updateMutation.mutate({
        id: editingBudget.id,
        data: { amount: numberToMoney(amount, user?.baseCurrency || 'SGD') },
      })
    } else {
      createMutation.mutate({
        categoryId: (formData.get('categoryId') as string) || budgetCategoryId,
        amount: numberToMoney(amount, user?.baseCurrency || 'SGD'),
        periodType: formData.get('periodType') as PeriodType,
        startDate: formData.get('startDate') as string,
      })
    }
  }

  const getPeriodLabel = (periodType: PeriodType) => {
    switch (periodType) {
      case 'PERIOD_TYPE_WEEKLY': return 'Weekly'
      case 'PERIOD_TYPE_MONTHLY': return 'Monthly'
      case 'PERIOD_TYPE_YEARLY': return 'Yearly'
      default: return periodType
    }
  }

  const formatStartDay = (startDate: string, periodType: PeriodType) => {
    const d = new Date(startDate)
    if (Number.isNaN(d.getTime())) return '—'

    if (periodType === 'PERIOD_TYPE_WEEKLY') {
      return d.toLocaleDateString(undefined, { weekday: 'long' })
    }
    if (periodType === 'PERIOD_TYPE_MONTHLY') {
      return `${d.getDate()}${getOrdinalSuffix(d.getDate())}`
    }
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }

  const getDaysLeftInCurrentCycle = (startDate: string, periodType: PeriodType) => {
    const start = new Date(startDate)
    const now = new Date()
    if (Number.isNaN(start.getTime())) return 0

    const cycleEnd = getCycleEndDate(start, periodType, now)
    const msPerDay = 1000 * 60 * 60 * 24
    const diff = cycleEnd.getTime() - now.getTime()
    return Math.max(0, Math.ceil(diff / msPerDay))
  }

  const closeModal = () => {
    setIsModalOpen(false)
    setEditingBudget(null)
    setBudgetCategoryId('')
    setShowQuickCategoryForm(false)
    setQuickCategoryName('')
  }

  const handleBudgetCategoryChange = (value: string) => {
    if (value === CREATE_CATEGORY_OPTION) {
      setShowQuickCategoryForm(true)
      setBudgetCategoryId('')
      return
    }

    setShowQuickCategoryForm(false)
    setBudgetCategoryId(value)
  }

  const handleQuickExpenseCategoryCreate = () => {
    const trimmed = quickCategoryName.trim()
    if (!trimmed) return
    createCategoryMutation.mutate({
      name: trimmed,
      type: 'TRANSACTION_TYPE_EXPENSE',
    })
  }

  const hasExpenseCategories = (categories?.length || 0) > 0
  const relatedTransactions: Transaction[] = relatedTransactionsData?.transactions || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            Budgets
          </h1>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">
            Set spending limits for each category
          </p>
        </div>
        <Button
          icon={<Plus className="h-4 w-4" />}
          onClick={() => {
            setEditingBudget(null)
            setBudgetCategoryId('')
            setIsModalOpen(true)
          }}
        >
          Add Budget
        </Button>
      </div>

      {!hasExpenseCategories && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300">
          You have no expense categories yet. Create one in this form or in{' '}
          <Link to="/settings" className="font-medium underline underline-offset-2">
            Settings
          </Link>{' '}
          before creating budgets.
        </div>
      )}

      {/* Budget Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full flex items-center justify-center py-16">
            <svg className="animate-spin h-6 w-6 text-violet-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
        ) : !budgetStatuses || budgetStatuses.length === 0 ? (
          <div className="col-span-full flex flex-col items-center justify-center py-16 text-zinc-400 dark:text-zinc-500 bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800">
            <p className="text-sm">No budgets set up yet.</p>
            <p className="text-xs mt-1">Create your first budget to start tracking.</p>
          </div>
        ) : (
          budgetStatuses.map((status) => {
            const isOverBudget = status.percentageUsed > 100
            const isWarning = status.percentageUsed > 80 && !isOverBudget
            const progressVariant = isOverBudget ? 'danger' : isWarning ? 'warning' : 'success'
            const badgeVariant = isOverBudget ? 'danger' : isWarning ? 'warning' : 'success'
            const startDay = formatStartDay(status.budget.startDate, status.budget.periodType)
            const daysLeft = getDaysLeftInCurrentCycle(status.budget.startDate, status.budget.periodType)

            return (
              <div
                key={status.budget.id}
                className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm hover:-translate-y-0.5 hover:shadow-md transition-all duration-200 group"
              >
                {/* Card Header */}
                <div className="flex items-start justify-between mb-4">
                  <div className="min-w-0 flex-1">
                    <h3 className="font-semibold text-zinc-900 dark:text-zinc-50 truncate">
                      {status.budget.categoryName}
                    </h3>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs text-zinc-500 dark:text-zinc-400">
                        {getPeriodLabel(status.budget.periodType)}
                      </span>
                      <Badge variant={badgeVariant}>
                        {status.percentageUsed.toFixed(0)}% used
                      </Badge>
                    </div>
                  </div>
                  <div className="flex gap-1 ml-2">
                    <button
                      onClick={() => {
                        setActiveBudgetForTransactions(status.budget)
                      }}
                      className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-violet-600 dark:hover:text-violet-400 hover:bg-violet-50 dark:hover:bg-violet-500/10 transition-colors duration-150"
                      title="View related transactions"
                    >
                      <List className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                    </button>
                    <button
                      onClick={() => {
                        setEditingBudget(status.budget)
                        setIsModalOpen(true)
                      }}
                      className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors duration-150"
                    >
                      <Pencil className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Delete this budget?')) {
                          deleteMutation.mutate(status.budget.id)
                        }
                      }}
                      className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors duration-150"
                    >
                      <Trash2 className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                    </button>
                  </div>
                </div>

                {/* Progress */}
                <ProgressBar
                  value={status.percentageUsed}
                  variant={progressVariant}
                  size="md"
                  className="mb-3"
                />

                {/* Stats */}
                <div className="space-y-1.5">
                  <div className="flex justify-between text-xs">
                    <span className="text-zinc-500 dark:text-zinc-400">Start Day</span>
                    <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                      {startDay}
                    </span>
                  </div>
                  <div className="flex justify-between text-xs">
                    <span className="text-zinc-500 dark:text-zinc-400">Days Left</span>
                    <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                      {daysLeft}
                    </span>
                  </div>
                  <div className="flex justify-between text-xs">
                    <span className="text-zinc-500 dark:text-zinc-400">Spent</span>
                    <span className={`font-medium tabular-nums ${isOverBudget ? 'text-red-500 dark:text-red-400' : 'text-zinc-900 dark:text-zinc-100'}`}>
                      {formatConverted(status.spent)}
                    </span>
                  </div>
                  <div className="flex justify-between text-xs">
                    <span className="text-zinc-500 dark:text-zinc-400">Budget</span>
                    <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                      {formatConverted(status.budget.amount)}
                    </span>
                  </div>
                  <div className="flex justify-between text-xs pt-1.5 border-t border-zinc-100 dark:border-zinc-800">
                    <span className="text-zinc-500 dark:text-zinc-400">Remaining</span>
                    <span
                      className={`font-semibold tabular-nums ${
                        moneyToNumber(status.remaining) < 0
                          ? 'text-red-500 dark:text-red-400'
                          : 'text-emerald-600 dark:text-emerald-400'
                      }`}
                    >
                      {formatConverted(status.remaining)}
                    </span>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Modal */}
      <Modal
        open={isModalOpen}
        onClose={closeModal}
        title={editingBudget ? 'Edit Budget' : 'Add Budget'}
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={closeModal}>Cancel</Button>
            <Button
              type="submit"
              form="budget-form"
              loading={createMutation.isPending || updateMutation.isPending}
            >
              {editingBudget ? 'Update' : 'Add'}
            </Button>
          </div>
        }
      >
        <form id="budget-form" onSubmit={handleSubmit} className="space-y-4">
          {!editingBudget && (
            <>
              <FormField label="Category">
                <Select
                  name="categoryId"
                  required
                  value={budgetCategoryId}
                  onChange={(e) => handleBudgetCategoryChange(e.target.value)}
                >
                  <option value="">Select category</option>
                  <option value={CREATE_CATEGORY_OPTION}>+ Create new category...</option>
                  {categories?.map((cat) => (
                    <option key={cat.id} value={cat.id}>{cat.name}</option>
                  ))}
                </Select>
              </FormField>

              {showQuickCategoryForm && (
                <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-2">
                  <Input
                    type="text"
                    value={quickCategoryName}
                    onChange={(e) => setQuickCategoryName(e.target.value)}
                    placeholder="Expense category name"
                  />
                  <div className="flex gap-2 justify-end">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => {
                        setShowQuickCategoryForm(false)
                        setQuickCategoryName('')
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      loading={createCategoryMutation.isPending}
                      onClick={handleQuickExpenseCategoryCreate}
                    >
                      Add Category
                    </Button>
                  </div>
                </div>
              )}

              <FormField label="Period">
                <Select name="periodType" required>
                  <option value="PERIOD_TYPE_MONTHLY">Monthly</option>
                  <option value="PERIOD_TYPE_WEEKLY">Weekly</option>
                  <option value="PERIOD_TYPE_YEARLY">Yearly</option>
                </Select>
              </FormField>

              <FormField label="Start Date">
                <Input
                  type="date"
                  name="startDate"
                  required
                  defaultValue={new Date().toISOString().split('T')[0]}
                />
              </FormField>
            </>
          )}

          <FormField label="Amount">
            <Input
              type="number"
              name="amount"
              step="0.01"
              min="0"
              required
              defaultValue={editingBudget ? moneyToNumber(editingBudget.amount) : ''}
              placeholder="0.00"
            />
          </FormField>
        </form>
      </Modal>

      <Modal
        open={!!activeBudgetForTransactions}
        onClose={() => setActiveBudgetForTransactions(null)}
        title={activeBudgetForTransactions ? `${activeBudgetForTransactions.categoryName} Transactions` : 'Related Transactions'}
        footer={
          <div className="flex justify-end">
            <Button variant="secondary" onClick={() => setActiveBudgetForTransactions(null)}>Close</Button>
          </div>
        }
      >
        <div className="space-y-3">
          {activeBudgetDateRange && (
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              Current cycle: {formatDate(activeBudgetDateRange.start.toISOString())} - {formatDate(activeBudgetDateRange.end.toISOString())}
            </p>
          )}

          {isLoadingRelatedTransactions ? (
            <div className="py-8 text-center text-sm text-zinc-500 dark:text-zinc-400">Loading transactions...</div>
          ) : relatedTransactions.length === 0 ? (
            <div className="py-8 text-center text-sm text-zinc-500 dark:text-zinc-400">No transactions in this budget cycle.</div>
          ) : (
            <div className="max-h-80 overflow-auto rounded-xl border border-zinc-200 dark:border-zinc-800 divide-y divide-zinc-100 dark:divide-zinc-800">
              {relatedTransactions.map((tx) => (
                <div key={tx.id} className="px-3 py-2.5 flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-sm text-zinc-900 dark:text-zinc-100 truncate">
                      {tx.description || tx.categoryName}
                    </p>
                    <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                      {formatDate(tx.transactionDate)}
                    </p>
                  </div>
                  <span className="text-sm font-medium tabular-nums text-red-500 dark:text-red-400 whitespace-nowrap">
                    {formatConverted(tx.amount)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </Modal>
    </div>
  )
}

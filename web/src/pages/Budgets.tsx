import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { budgetApi, categoryApi } from '../api'
import { numberToMoney, moneyToNumber } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import { useCurrency } from '../store/CurrencyContext'
import type { Budget, Category, PeriodType } from '../types'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { Button, Modal, FormField, Input, Select, ProgressBar, Badge } from '../components/ui'

const CREATE_CATEGORY_OPTION = '__create_new_category__'

export default function Budgets() {
  const { user } = useAuth()
  const { formatConverted } = useCurrency()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingBudget, setEditingBudget] = useState<Budget | null>(null)
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
                  <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity duration-150 ml-2">
                    <button
                      onClick={() => {
                        setEditingBudget(status.budget)
                        setIsModalOpen(true)
                      }}
                      className="h-7 w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors duration-150"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Delete this budget?')) {
                          deleteMutation.mutate(status.budget.id)
                        }
                      }}
                      className="h-7 w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors duration-150"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
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
    </div>
  )
}

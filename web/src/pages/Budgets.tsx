import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { budgetApi, categoryApi } from '../api'
import { formatMoney, numberToMoney, moneyToNumber } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import type { Budget, PeriodType } from '../types'
import { Plus, Pencil, Trash2, X } from 'lucide-react'

export default function Budgets() {
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingBudget, setEditingBudget] = useState<Budget | null>(null)

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
        categoryId: formData.get('categoryId') as string,
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

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Budgets</h1>
          <p className="text-gray-500">Set spending limits for each category</p>
        </div>
        <button
          onClick={() => {
            setEditingBudget(null)
            setIsModalOpen(true)
          }}
          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus className="h-5 w-5 mr-2" />
          Add Budget
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : !budgetStatuses || budgetStatuses.length === 0 ? (
          <div className="col-span-full text-center py-12 text-gray-500 bg-white rounded-xl border border-gray-100">
            No budgets set up yet. Create your first budget to start tracking.
          </div>
        ) : (
          budgetStatuses.map((status) => {
            const isOverBudget = status.percentageUsed > 100
            const isWarning = status.percentageUsed > 80 && !isOverBudget
            return (
              <div key={status.budget.id} className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
                <div className="flex items-start justify-between mb-4">
                  <div>
                    <h3 className="font-semibold text-gray-900">{status.budget.categoryName}</h3>
                    <p className="text-sm text-gray-500">{getPeriodLabel(status.budget.periodType)}</p>
                  </div>
                  <div className="flex gap-1">
                    <button
                      onClick={() => {
                        setEditingBudget(status.budget)
                        setIsModalOpen(true)
                      }}
                      className="p-2 text-gray-400 hover:text-blue-600 transition-colors"
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Delete this budget?')) {
                          deleteMutation.mutate(status.budget.id)
                        }
                      }}
                      className="p-2 text-gray-400 hover:text-red-600 transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Spent</span>
                    <span className={`font-medium ${isOverBudget ? 'text-red-600' : 'text-gray-900'}`}>
                      {formatMoney(status.spent)}
                    </span>
                  </div>
                  <div className="h-3 bg-gray-200 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${
                        isOverBudget ? 'bg-red-500' : isWarning ? 'bg-yellow-500' : 'bg-green-500'
                      }`}
                      style={{ width: `${Math.min(status.percentageUsed, 100)}%` }}
                    />
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Budget</span>
                    <span className="font-medium text-gray-900">{formatMoney(status.budget.amount)}</span>
                  </div>
                  <div className="flex justify-between text-sm pt-2 border-t border-gray-100">
                    <span className="text-gray-600">Remaining</span>
                    <span className={`font-medium ${moneyToNumber(status.remaining) < 0 ? 'text-red-600' : 'text-green-600'}`}>
                      {formatMoney(status.remaining)}
                    </span>
                  </div>
                </div>

                <div className="mt-4 text-center">
                  <span className={`text-2xl font-bold ${
                    isOverBudget ? 'text-red-600' : isWarning ? 'text-yellow-600' : 'text-green-600'
                  }`}>
                    {status.percentageUsed.toFixed(0)}%
                  </span>
                  <span className="text-gray-500 text-sm ml-1">used</span>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-md">
            <div className="flex items-center justify-between p-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold">
                {editingBudget ? 'Edit Budget' : 'Add Budget'}
              </h2>
              <button onClick={() => { setIsModalOpen(false); setEditingBudget(null) }} className="p-2 text-gray-400 hover:text-gray-600">
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="p-4 space-y-4">
              {!editingBudget && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Category</label>
                    <select
                      name="categoryId"
                      required
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">Select category</option>
                      {categories?.map((cat) => (
                        <option key={cat.id} value={cat.id}>{cat.name}</option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Period</label>
                    <select
                      name="periodType"
                      required
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="PERIOD_TYPE_MONTHLY">Monthly</option>
                      <option value="PERIOD_TYPE_WEEKLY">Weekly</option>
                      <option value="PERIOD_TYPE_YEARLY">Yearly</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Start Date</label>
                    <input
                      type="date"
                      name="startDate"
                      required
                      defaultValue={new Date().toISOString().split('T')[0]}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                </>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Amount</label>
                <input
                  type="number"
                  name="amount"
                  step="0.01"
                  min="0"
                  required
                  defaultValue={editingBudget ? moneyToNumber(editingBudget.amount) : ''}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="0.00"
                />
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => { setIsModalOpen(false); setEditingBudget(null) }}
                  className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending || updateMutation.isPending}
                  className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  {editingBudget ? 'Update' : 'Add'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

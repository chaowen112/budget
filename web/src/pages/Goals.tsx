import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { goalApi } from '../api'
import { formatMoney, numberToMoney, moneyToNumber, formatDate } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import type { SavingGoal } from '../types'
import { Plus, Pencil, Trash2, X, Target, TrendingUp } from 'lucide-react'

export default function Goals() {
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [isProgressModalOpen, setIsProgressModalOpen] = useState(false)
  const [editingGoal, setEditingGoal] = useState<SavingGoal | null>(null)
  const [selectedGoal, setSelectedGoal] = useState<SavingGoal | null>(null)

  const { data: goalsProgress, isLoading } = useQuery({
    queryKey: ['goalsProgress'],
    queryFn: () => goalApi.getAllProgress(),
  })

  const createMutation = useMutation({
    mutationFn: goalApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goalsProgress'] })
      setIsModalOpen(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof goalApi.update>[1] }) =>
      goalApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goalsProgress'] })
      setIsModalOpen(false)
      setEditingGoal(null)
    },
  })

  const updateProgressMutation = useMutation({
    mutationFn: ({ id, amount }: { id: string; amount: ReturnType<typeof numberToMoney> }) =>
      goalApi.updateProgress(id, amount),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goalsProgress'] })
      setIsProgressModalOpen(false)
      setSelectedGoal(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: goalApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goalsProgress'] })
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const targetAmount = parseFloat(formData.get('targetAmount') as string)
    const deadline = formData.get('deadline') as string

    if (editingGoal) {
      updateMutation.mutate({
        id: editingGoal.id,
        data: {
          name: formData.get('name') as string,
          targetAmount: numberToMoney(targetAmount, user?.baseCurrency || 'SGD'),
          deadline: deadline || undefined,
          notes: formData.get('notes') as string,
        },
      })
    } else {
      createMutation.mutate({
        name: formData.get('name') as string,
        targetAmount: numberToMoney(targetAmount, user?.baseCurrency || 'SGD'),
        deadline: deadline || undefined,
        notes: formData.get('notes') as string,
      })
    }
  }

  const handleProgressSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!selectedGoal) return
    const formData = new FormData(e.currentTarget)
    const amount = parseFloat(formData.get('amount') as string)
    updateProgressMutation.mutate({
      id: selectedGoal.id,
      amount: numberToMoney(amount, user?.baseCurrency || 'SGD'),
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Saving Goals</h1>
          <p className="text-gray-500">Track your progress towards financial goals</p>
        </div>
        <button
          onClick={() => {
            setEditingGoal(null)
            setIsModalOpen(true)
          }}
          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="h-5 w-5 mr-2" />
          Add Goal
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : !goalsProgress || goalsProgress.length === 0 ? (
          <div className="col-span-full text-center py-12 text-gray-500 bg-white rounded-xl border border-gray-100">
            <Target className="h-12 w-12 mx-auto mb-4 text-gray-300" />
            No saving goals yet. Create your first goal to start saving.
          </div>
        ) : (
          goalsProgress.map((progress) => {
            const goal = progress.goal
            const isCompleted = progress.percentageComplete >= 100
            return (
              <div key={goal.id} className={`bg-white rounded-xl p-6 shadow-sm border ${isCompleted ? 'border-green-200 bg-green-50' : 'border-gray-100'}`}>
                <div className="flex items-start justify-between mb-4">
                  <div>
                    <h3 className="font-semibold text-gray-900">{goal.name}</h3>
                    {goal.deadline && (
                      <p className="text-sm text-gray-500">Target: {formatDate(goal.deadline)}</p>
                    )}
                  </div>
                  <div className="flex gap-1">
                    {!isCompleted && (
                      <button
                        onClick={() => {
                          setSelectedGoal(goal)
                          setIsProgressModalOpen(true)
                        }}
                        className="p-2 text-gray-400 hover:text-green-600"
                        title="Add Progress"
                      >
                        <TrendingUp className="h-4 w-4" />
                      </button>
                    )}
                    <button
                      onClick={() => {
                        setEditingGoal(goal)
                        setIsModalOpen(true)
                      }}
                      className="p-2 text-gray-400 hover:text-blue-600"
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Delete this goal?')) {
                          deleteMutation.mutate(goal.id)
                        }
                      }}
                      className="p-2 text-gray-400 hover:text-red-600"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Saved</span>
                    <span className="font-medium text-gray-900">{formatMoney(goal.currentAmount)}</span>
                  </div>
                  <div className="h-3 bg-gray-200 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${isCompleted ? 'bg-green-500' : 'bg-blue-500'}`}
                      style={{ width: `${Math.min(progress.percentageComplete, 100)}%` }}
                    />
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Target</span>
                    <span className="font-medium text-gray-900">{formatMoney(goal.targetAmount)}</span>
                  </div>
                </div>

                <div className="mt-4 pt-4 border-t border-gray-100 space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Progress</span>
                    <span className={`font-bold ${isCompleted ? 'text-green-600' : 'text-blue-600'}`}>
                      {progress.percentageComplete.toFixed(1)}%
                    </span>
                  </div>
                  {!isCompleted && progress.daysRemaining > 0 && (
                    <>
                      <div className="flex justify-between text-sm">
                        <span className="text-gray-600">Days remaining</span>
                        <span className="text-gray-900">{progress.daysRemaining}</span>
                      </div>
                      <div className="flex justify-between text-sm">
                        <span className="text-gray-600">Need/month</span>
                        <span className={`font-medium ${progress.isOnTrack ? 'text-green-600' : 'text-yellow-600'}`}>
                          {formatMoney(progress.requiredMonthlySaving)}
                        </span>
                      </div>
                    </>
                  )}
                  {isCompleted && (
                    <div className="text-center text-green-600 font-medium">
                      Goal Completed!
                    </div>
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Create/Edit Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-md">
            <div className="flex items-center justify-between p-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold">{editingGoal ? 'Edit Goal' : 'Add Goal'}</h2>
              <button onClick={() => { setIsModalOpen(false); setEditingGoal(null) }} className="p-2 text-gray-400 hover:text-gray-600">
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="p-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Goal Name</label>
                <input
                  type="text"
                  name="name"
                  required
                  defaultValue={editingGoal?.name || ''}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., Emergency Fund"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Target Amount</label>
                <input
                  type="number"
                  name="targetAmount"
                  step="0.01"
                  min="0"
                  required
                  defaultValue={editingGoal ? moneyToNumber(editingGoal.targetAmount) : ''}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="0.00"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Target Date (Optional)</label>
                <input
                  type="date"
                  name="deadline"
                  defaultValue={editingGoal?.deadline?.split('T')[0] || ''}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Notes</label>
                <textarea
                  name="notes"
                  rows={2}
                  defaultValue={editingGoal?.notes || ''}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="Additional notes..."
                />
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => { setIsModalOpen(false); setEditingGoal(null) }} className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50">
                  Cancel
                </button>
                <button type="submit" disabled={createMutation.isPending || updateMutation.isPending} className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
                  {editingGoal ? 'Update' : 'Add'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Progress Modal */}
      {isProgressModalOpen && selectedGoal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-md">
            <div className="flex items-center justify-between p-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold">Update Progress</h2>
              <button onClick={() => { setIsProgressModalOpen(false); setSelectedGoal(null) }} className="p-2 text-gray-400 hover:text-gray-600">
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleProgressSubmit} className="p-4 space-y-4">
              <p className="text-gray-600">
                Current: <strong>{formatMoney(selectedGoal.currentAmount)}</strong> / {formatMoney(selectedGoal.targetAmount)}
              </p>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">New Total Amount</label>
                <input
                  type="number"
                  name="amount"
                  step="0.01"
                  min="0"
                  required
                  defaultValue={moneyToNumber(selectedGoal.currentAmount)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => { setIsProgressModalOpen(false); setSelectedGoal(null) }} className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50">
                  Cancel
                </button>
                <button type="submit" disabled={updateProgressMutation.isPending} className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50">
                  Update
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

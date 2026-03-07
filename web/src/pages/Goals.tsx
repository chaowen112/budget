import { useEffect, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { goalApi } from '../api'
import { numberToMoney, moneyToNumber, formatDate } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import { useCurrency } from '../store/CurrencyContext'
import type { GoalProgress, SavingGoal } from '../types'
import { Plus, Pencil, Trash2, Target, TrendingUp, CheckCircle2, Clock } from 'lucide-react'
import { Button, Modal, FormField, Input, Textarea, ProgressBar, Badge, Select } from '../components/ui'
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  Legend,
} from 'recharts'

type AllocationCadence = 'weekly' | 'monthly'

interface GoalAutoAllocation {
  goalId: string
  goalName: string
  daysRemaining: number
  capAmount: number
  allocation: number
  currentAmount: number
  projectedAmount: number
}

export default function Goals() {
  const { user } = useAuth()
  const { formatConverted } = useCurrency()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [isProgressModalOpen, setIsProgressModalOpen] = useState(false)
  const [isAutoAllocateModalOpen, setIsAutoAllocateModalOpen] = useState(false)
  const [editingGoal, setEditingGoal] = useState<SavingGoal | null>(null)
  const [selectedGoal, setSelectedGoal] = useState<SavingGoal | null>(null)
  const [goalFormTargetAmount, setGoalFormTargetAmount] = useState('')
  const [goalFormInitialAmount, setGoalFormInitialAmount] = useState('')
  const [goalFormDeadline, setGoalFormDeadline] = useState('')
  const [autoAllocateAmount, setAutoAllocateAmount] = useState('')
  const [autoAllocateCadence, setAutoAllocateCadence] = useState<AllocationCadence>('monthly')
  const [historyGoalId, setHistoryGoalId] = useState('')

  const { data: goalsProgress, isLoading } = useQuery({
    queryKey: ['goalsProgress'],
    queryFn: () => goalApi.getAllProgress(),
  })

  const { data: goalHistory } = useQuery({
    queryKey: ['goalHistory', historyGoalId],
    queryFn: () => goalApi.getHistory(historyGoalId),
    enabled: !!historyGoalId,
  })

  useEffect(() => {
    if (!historyGoalId && goalsProgress && goalsProgress.length > 0) {
      setHistoryGoalId(goalsProgress[0].goal.id)
    }
  }, [goalsProgress, historyGoalId])

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
    mutationFn: ({
      id,
      amount,
      source,
    }: {
      id: string
      amount: ReturnType<typeof numberToMoney>
      source?: string
    }) => goalApi.updateProgress(id, amount, source),
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

  const autoAllocateMutation = useMutation({
    mutationFn: async (allocations: GoalAutoAllocation[]) => {
      const currency = user?.baseCurrency || 'SGD'
      for (const item of allocations) {
        if (item.allocation <= 0) {
          continue
        }

        await goalApi.updateProgress(item.goalId, numberToMoney(item.projectedAmount, currency), 'auto_allocate')
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goalsProgress'] })
      setIsAutoAllocateModalOpen(false)
      setAutoAllocateAmount('')
    },
  })

  const calculateRequiredWeeklySaving = (progress: GoalProgress) => {
    if (progress.daysRemaining <= 0) {
      return progress.amountRemaining
    }

    const weeksRemaining = Math.max(1, Math.ceil(progress.daysRemaining / 7))
    const weeklyAmount = moneyToNumber(progress.amountRemaining) / weeksRemaining
    return numberToMoney(weeklyAmount, progress.amountRemaining.currency)
  }

  const calculateAllocationCap = (progress: GoalProgress, cadence: AllocationCadence) => {
    const monthlyNeed = moneyToNumber(progress.requiredMonthlySaving)
    const weeklyNeed = moneyToNumber(calculateRequiredWeeklySaving(progress))
    const remaining = moneyToNumber(progress.amountRemaining)
    const rawCap = cadence === 'monthly' ? monthlyNeed : weeklyNeed
    return Math.max(0, Math.min(rawCap, remaining))
  }

  const buildAutoAllocations = (progressList: GoalProgress[], availableAmount: number, cadence: AllocationCadence) => {
    if (Number.isNaN(availableAmount) || availableAmount <= 0) {
      return { allocations: [] as GoalAutoAllocation[], unallocated: 0 }
    }

    const eligible = progressList
      .filter((p) => p.daysRemaining > 0)
      .filter((p) => p.percentageComplete < 100)
      .filter((p) => moneyToNumber(p.amountRemaining) > 0)
      .sort((a, b) => a.daysRemaining - b.daysRemaining)

    let pool = availableAmount
    const allocations: GoalAutoAllocation[] = []

    for (const progress of eligible) {
      if (pool <= 0) {
        break
      }

      const capAmount = calculateAllocationCap(progress, cadence)
      const allocation = Math.min(pool, capAmount)
      const currentAmount = moneyToNumber(progress.goal.currentAmount)
      allocations.push({
        goalId: progress.goal.id,
        goalName: progress.goal.name,
        daysRemaining: progress.daysRemaining,
        capAmount,
        allocation,
        currentAmount,
        projectedAmount: currentAmount + allocation,
      })

      pool -= allocation
    }

    return { allocations, unallocated: Math.max(0, pool) }
  }

  const parsedAutoAllocateAmount = parseFloat(autoAllocateAmount)
  const { allocations: autoAllocations, unallocated: unallocatedAutoAmount } = buildAutoAllocations(
    goalsProgress || [],
    parsedAutoAllocateAmount,
    autoAllocateCadence,
  )

  const shortMoney = (value: number) => {
    const abs = Math.abs(value)
    if (abs >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}m`
    if (abs >= 1_000) return `${(value / 1_000).toFixed(1)}k`
    return value.toFixed(0)
  }

  const selectedGoalProgress = (goalsProgress || []).find((p) => p.goal.id === historyGoalId) || (goalsProgress || [])[0]
  const effectiveHistoryGoalId = historyGoalId || selectedGoalProgress?.goal.id || ''

  const goalLineData = (() => {
    if (!selectedGoalProgress) {
      return [] as Array<{ date: string; actual: number; estimated: number | null }>
    }

    const goal = selectedGoalProgress.goal
    const now = new Date()
    const createdAt = new Date(goal.createdAt)
    const deadline = goal.deadline ? new Date(goal.deadline) : null
    const targetAmount = moneyToNumber(goal.targetAmount)

    const contributions = (goalHistory?.contributions || []).map((item) => ({
      at: new Date(item.recordedAt),
      delta: parseFloat(item.amountDelta) || 0,
      balanceAfter: parseFloat(item.balanceAfter) || 0,
    })).sort((a, b) => a.at.getTime() - b.at.getTime())

    const snapshots = (goalHistory?.history || []).map((item) => ({
      at: new Date(item.recordedAt),
      amount: parseFloat(item.amount) || 0,
    })).sort((a, b) => a.at.getTime() - b.at.getTime())

    let initialAmount = 0
    if (contributions.length > 0) {
      initialAmount = contributions[0].balanceAfter - contributions[0].delta
    } else {
      const firstSnapshot = snapshots[0]
      const sameDayInitial = firstSnapshot
        ? firstSnapshot.at.toDateString() === createdAt.toDateString()
        : false
      initialAmount = sameDayInitial ? firstSnapshot.amount : 0
    }

    const endDate = deadline && deadline > now ? deadline : now
    const startDay = new Date(createdAt.getFullYear(), createdAt.getMonth(), createdAt.getDate())
    const endDay = new Date(endDate.getFullYear(), endDate.getMonth(), endDate.getDate())

    let contributionIndex = 0
    let snapshotIndex = 0
    let lastActual = initialAmount
    const points: Array<{ date: string; actual: number; estimated: number | null }> = []

    for (let day = new Date(startDay); day <= endDay; day.setDate(day.getDate() + 1)) {
      const dayEnd = new Date(day)
      dayEnd.setHours(23, 59, 59, 999)

      if (contributions.length > 0) {
        while (contributionIndex < contributions.length && contributions[contributionIndex].at <= dayEnd) {
          lastActual += contributions[contributionIndex].delta
          contributionIndex += 1
        }
      } else {
        while (snapshotIndex < snapshots.length && snapshots[snapshotIndex].at <= dayEnd) {
          lastActual = snapshots[snapshotIndex].amount
          snapshotIndex += 1
        }
      }

      let estimated: number | null = null
      if (deadline) {
        const total = deadline.getTime() - createdAt.getTime()
        if (total <= 0) {
          estimated = targetAmount
        } else {
          const elapsed = Math.min(Math.max(dayEnd.getTime() - createdAt.getTime(), 0), total)
          estimated = initialAmount + ((targetAmount - initialAmount) * elapsed) / total
        }
      }

      points.push({
        date: day.toISOString().slice(0, 10),
        actual: lastActual,
        estimated,
      })
    }

    return points
  })()

  const calculateNeedFromInputs = (target: number, current: number, deadline: string) => {
    if (!deadline || Number.isNaN(target) || target <= 0) return null

    const targetDate = new Date(deadline)
    const now = new Date()
    const msRemaining = targetDate.getTime() - now.getTime()
    if (msRemaining <= 0) return null

    const daysRemaining = Math.ceil(msRemaining / (1000 * 60 * 60 * 24))
    const amountRemaining = Math.max(0, target - current)
    const weeksRemaining = Math.max(1, Math.ceil(daysRemaining / 7))
    const monthsRemaining = Math.max(1, daysRemaining / 30.44)
    const currency = user?.baseCurrency || 'SGD'

    return {
      weekly: numberToMoney(amountRemaining / weeksRemaining, currency),
      monthly: numberToMoney(amountRemaining / monthsRemaining, currency),
    }
  }

  const openCreateGoalModal = () => {
    setEditingGoal(null)
    setGoalFormTargetAmount('')
    setGoalFormInitialAmount('')
    setGoalFormDeadline('')
    setIsModalOpen(true)
  }

  const openEditGoalModal = (goal: SavingGoal) => {
    setEditingGoal(goal)
    setGoalFormTargetAmount(String(moneyToNumber(goal.targetAmount)))
    setGoalFormInitialAmount(String(moneyToNumber(goal.currentAmount)))
    setGoalFormDeadline(goal.deadline?.split('T')[0] || '')
    setIsModalOpen(true)
  }

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const targetAmount = parseFloat(goalFormTargetAmount)
    const initialAmount = parseFloat(goalFormInitialAmount || '0')
    const deadline = goalFormDeadline
    const currency = user?.baseCurrency || 'SGD'

    if (editingGoal) {
      updateMutation.mutate({
        id: editingGoal.id,
        data: {
          name: formData.get('name') as string,
          targetAmount: numberToMoney(targetAmount, currency),
          deadline: deadline || undefined,
          notes: formData.get('notes') as string,
        },
      })
    } else {
      try {
        const createdGoal = await createMutation.mutateAsync({
          name: formData.get('name') as string,
          targetAmount: numberToMoney(targetAmount, currency),
          deadline: deadline || undefined,
          notes: formData.get('notes') as string,
        })

        if (!Number.isNaN(initialAmount) && initialAmount > 0) {
          await updateProgressMutation.mutateAsync({
            id: createdGoal.id,
            amount: numberToMoney(initialAmount, currency),
            source: 'initial_deposit',
          })
        }
      } catch {
        return
      }
    }
  }

  const handleProgressSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!selectedGoal) return
    const formData = new FormData(e.currentTarget)
    const delta = parseFloat(formData.get('amountDelta') as string)
    if (Number.isNaN(delta)) return
    const currentAmount = moneyToNumber(selectedGoal.currentAmount)
    const nextTotal = currentAmount + delta
    if (nextTotal < 0) {
      alert('Amount cannot make goal balance below 0')
      return
    }

    updateProgressMutation.mutate({
      id: selectedGoal.id,
      amount: numberToMoney(nextTotal, user?.baseCurrency || 'SGD'),
      source: delta >= 0 ? 'manual_add' : 'manual_withdraw',
    })
  }

  const closeModal = () => {
    setIsModalOpen(false)
    setEditingGoal(null)
    setGoalFormTargetAmount('')
    setGoalFormInitialAmount('')
    setGoalFormDeadline('')
  }

  const closeProgressModal = () => {
    setIsProgressModalOpen(false)
    setSelectedGoal(null)
  }

  const closeAutoAllocateModal = () => {
    setIsAutoAllocateModalOpen(false)
    setAutoAllocateAmount('')
    setAutoAllocateCadence('monthly')
  }

  const handleAutoAllocate = () => {
    const actionable = autoAllocations.filter((a) => a.allocation > 0)
    if (actionable.length === 0) {
      return
    }
    autoAllocateMutation.mutate(actionable)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            Saving Goals
          </h1>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">
            Track your progress towards financial goals
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            onClick={() => setIsAutoAllocateModalOpen(true)}
            disabled={!goalsProgress || goalsProgress.length === 0}
          >
            Auto-Allocate
          </Button>
          <Button
            icon={<Plus className="h-4 w-4" />}
            onClick={openCreateGoalModal}
          >
            Add Goal
          </Button>
        </div>
      </div>

      {/* Goals Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full flex items-center justify-center py-16">
            <svg className="animate-spin h-6 w-6 text-violet-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
        ) : !goalsProgress || goalsProgress.length === 0 ? (
          <div className="col-span-full flex flex-col items-center justify-center py-16 bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800">
            <Target className="h-10 w-10 mb-3 text-zinc-300 dark:text-zinc-600" />
            <p className="text-sm text-zinc-500 dark:text-zinc-400">No saving goals yet.</p>
            <p className="text-xs text-zinc-400 dark:text-zinc-500 mt-1">Create your first goal to start saving.</p>
          </div>
        ) : (
          goalsProgress.map((progress) => {
            const goal = progress.goal
            const isCompleted = progress.percentageComplete >= 100
            const progressVariant = isCompleted ? 'success' : progress.isOnTrack ? 'default' : 'warning'

            return (
              <div
                key={goal.id}
                className={`rounded-2xl border p-5 shadow-sm hover:-translate-y-0.5 hover:shadow-md transition-all duration-200 group ${
                  isCompleted
                    ? 'bg-emerald-50 dark:bg-emerald-500/5 border-emerald-200 dark:border-emerald-500/20'
                    : 'bg-white dark:bg-zinc-900 border-zinc-200 dark:border-zinc-800'
                }`}
              >
                {/* Card Header */}
                <div className="flex items-start justify-between mb-4">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      {isCompleted ? (
                        <CheckCircle2 className="h-4 w-4 text-emerald-500 flex-shrink-0" />
                      ) : (
                        <Target className="h-4 w-4 text-zinc-400 dark:text-zinc-500 flex-shrink-0" />
                      )}
                      <h3 className="font-semibold text-zinc-900 dark:text-zinc-50 truncate">
                        {goal.name}
                      </h3>
                    </div>
                    {goal.deadline && (
                      <div className="flex items-center gap-1 text-xs text-zinc-500 dark:text-zinc-400">
                        <Clock className="h-3 w-3" />
                        <span>{formatDate(goal.deadline)}</span>
                      </div>
                    )}
                  </div>
                  <div className="flex gap-1 ml-2">
                    {!isCompleted && (
                      <button
                        onClick={() => {
                          setSelectedGoal(goal)
                          setIsProgressModalOpen(true)
                        }}
                        className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-emerald-600 dark:hover:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-500/10 transition-colors duration-150"
                        title="Update Progress"
                      >
                        <TrendingUp className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                      </button>
                    )}
                    <button
                      onClick={() => {
                        openEditGoalModal(goal)
                      }}
                      className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors duration-150"
                    >
                      <Pencil className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Delete this goal?')) {
                          deleteMutation.mutate(goal.id)
                        }
                      }}
                      className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors duration-150"
                    >
                      <Trash2 className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                    </button>
                  </div>
                </div>

                <ProgressBar
                  value={progress.percentageComplete}
                  variant={progressVariant}
                  size="md"
                  className="mb-3"
                />

                {/* Stats */}
                <div className="space-y-1.5">
                  <div className="flex justify-between text-xs">
                    <span className="text-zinc-500 dark:text-zinc-400">Saved</span>
                    <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                      {formatConverted(goal.currentAmount)}
                    </span>
                  </div>
                  <div className="flex justify-between text-xs">
                    <span className="text-zinc-500 dark:text-zinc-400">Target</span>
                    <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                      {formatConverted(goal.targetAmount)}
                    </span>
                  </div>
                </div>

                {/* Footer */}
                <div className="mt-3 pt-3 border-t border-zinc-100 dark:border-zinc-800 space-y-1.5">
                  {isCompleted ? (
                    <div className="flex items-center justify-center gap-1.5 text-emerald-600 dark:text-emerald-400">
                      <CheckCircle2 className="h-4 w-4" />
                      <span className="text-sm font-semibold">Goal Completed!</span>
                    </div>
                  ) : (
                    <>
                      <div className="flex justify-between text-xs items-center">
                        <span className="text-zinc-500 dark:text-zinc-400">Progress</span>
                        <Badge variant={progress.isOnTrack ? 'success' : 'warning'}>
                          {progress.percentageComplete.toFixed(1)}%
                        </Badge>
                      </div>
                      {progress.daysRemaining > 0 && (
                        <>
                          <div className="flex justify-between text-xs">
                            <span className="text-zinc-500 dark:text-zinc-400">Days remaining</span>
                            <span className="text-zinc-900 dark:text-zinc-100">{progress.daysRemaining}</span>
                          </div>
                          <div className="flex justify-between text-xs">
                            <span className="text-zinc-500 dark:text-zinc-400">Need/week</span>
                            <span className={`font-medium tabular-nums ${
                              progress.isOnTrack
                                ? 'text-emerald-600 dark:text-emerald-400'
                                : 'text-amber-600 dark:text-amber-400'
                            }`}>
                              {formatConverted(calculateRequiredWeeklySaving(progress))}
                            </span>
                          </div>
                          <div className="flex justify-between text-xs">
                            <span className="text-zinc-500 dark:text-zinc-400">Need/month</span>
                            <span className={`font-medium tabular-nums ${
                              progress.isOnTrack
                                ? 'text-emerald-600 dark:text-emerald-400'
                                : 'text-amber-600 dark:text-amber-400'
                            }`}>
                              {formatConverted(progress.requiredMonthlySaving)}
                            </span>
                          </div>
                        </>
                      )}
                    </>
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>

      {selectedGoalProgress && (
        <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-4">
            <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">Goal Path: Estimated vs Actual</h2>
            <div className="w-full sm:w-64">
              <Select value={effectiveHistoryGoalId} onChange={(e) => setHistoryGoalId(e.target.value)}>
                {(goalsProgress || []).map((p) => (
                  <option key={p.goal.id} value={p.goal.id}>{p.goal.name}</option>
                ))}
              </Select>
            </div>
          </div>
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={goalLineData}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(113,113,122,0.15)" />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 11, fill: '#71717a' }}
                  axisLine={false}
                  tickLine={false}
                  interval="preserveStartEnd"
                  minTickGap={36}
                  tickFormatter={(value: string) => {
                    const d = new Date(value)
                    const day = d.getDate()
                    if (day === 1 || day === 15) {
                      return `${d.getMonth() + 1}/${day}`
                    }
                    return ''
                  }}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: '#71717a' }}
                  axisLine={false}
                  tickLine={false}
                  tickCount={4}
                  tickFormatter={(value: number) => shortMoney(value)}
                />
                <Tooltip
                  formatter={(value: number | undefined, name: string | undefined) => [formatConverted(numberToMoney(value ?? 0, user?.baseCurrency || 'SGD')), name || 'Value']}
                  labelFormatter={(value: unknown) => formatDate(String(value ?? ''))}
                  contentStyle={{
                    background: '#18181b',
                    border: '1px solid #3f3f46',
                    borderRadius: 12,
                    fontSize: 12,
                    color: '#fff',
                  }}
                />
                <Legend wrapperStyle={{ fontSize: 11, color: '#71717a' }} />
                <Line type="monotone" dataKey="actual" stroke="#10b981" strokeWidth={2.5} dot={false} name="Actual" />
                <Line type="monotone" dataKey="estimated" stroke="#8b5cf6" strokeWidth={2} strokeDasharray="6 4" dot={false} name="Estimated" connectNulls />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Auto Allocate Modal */}
      <Modal
        open={isAutoAllocateModalOpen}
        onClose={closeAutoAllocateModal}
        title="Auto-Allocate Goals"
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={closeAutoAllocateModal}>Cancel</Button>
            <Button
              onClick={handleAutoAllocate}
              loading={autoAllocateMutation.isPending}
              disabled={autoAllocations.every((a) => a.allocation <= 0)}
            >
              Apply Allocation
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <FormField label="Available Amount">
              <Input
                type="number"
                step="0.01"
                min="0"
                value={autoAllocateAmount}
                onChange={(e) => setAutoAllocateAmount(e.target.value)}
                placeholder="0.00"
              />
            </FormField>

            <FormField label="Allocation Cadence">
              <Select
                value={autoAllocateCadence}
                onChange={(e) => setAutoAllocateCadence(e.target.value as AllocationCadence)}
              >
                <option value="weekly">Weekly (cap = need/week)</option>
                <option value="monthly">Monthly (cap = need/month)</option>
              </Select>
            </FormField>
          </div>

          <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-2">
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              Urgency-first allocation applies only to goals with a target date. Each goal is capped at its {autoAllocateCadence === 'weekly' ? 'need/week' : 'need/month'} amount.
            </p>
            <div className="space-y-1.5 max-h-60 overflow-y-auto">
              {autoAllocations.length > 0 ? (
                autoAllocations.map((item) => (
                  <div
                    key={item.goalId}
                    className="flex items-center justify-between text-xs rounded-lg px-2.5 py-2 bg-zinc-50 dark:bg-zinc-800/60"
                  >
                    <div className="min-w-0 pr-2">
                      <p className="font-medium text-zinc-900 dark:text-zinc-100 truncate">{item.goalName}</p>
                      <p className="text-zinc-500 dark:text-zinc-400">{item.daysRemaining} days remaining</p>
                    </div>
                    <div className="text-right">
                      <p className="text-zinc-500 dark:text-zinc-400">Cap {formatConverted(numberToMoney(item.capAmount, user?.baseCurrency || 'SGD'))}</p>
                      <p className="font-semibold text-zinc-900 dark:text-zinc-100">Allocate {formatConverted(numberToMoney(item.allocation, user?.baseCurrency || 'SGD'))}</p>
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-sm text-zinc-500 dark:text-zinc-400">Enter an available amount to preview allocation.</p>
              )}
            </div>
          </div>

          <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-1.5 bg-zinc-50/60 dark:bg-zinc-800/30">
            <div className="flex justify-between text-xs">
              <span className="text-zinc-500 dark:text-zinc-400">Total allocated</span>
              <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                {formatConverted(numberToMoney(autoAllocations.reduce((sum, item) => sum + item.allocation, 0), user?.baseCurrency || 'SGD'))}
              </span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-zinc-500 dark:text-zinc-400">Unallocated</span>
              <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                {formatConverted(numberToMoney(unallocatedAutoAmount, user?.baseCurrency || 'SGD'))}
              </span>
            </div>
          </div>
        </div>
      </Modal>

      {/* Create/Edit Goal Modal */}
      <Modal
        open={isModalOpen}
        onClose={closeModal}
        title={editingGoal ? 'Edit Goal' : 'Add Goal'}
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={closeModal}>Cancel</Button>
            <Button
              type="submit"
              form="goal-form"
              loading={createMutation.isPending || updateMutation.isPending || updateProgressMutation.isPending}
            >
              {editingGoal ? 'Update' : 'Add'}
            </Button>
          </div>
        }
      >
        <form id="goal-form" onSubmit={handleSubmit} className="space-y-4">
          <FormField label="Goal Name">
            <Input
              type="text"
              name="name"
              required
              defaultValue={editingGoal?.name || ''}
              placeholder="e.g., Emergency Fund"
            />
          </FormField>

          <FormField label="Target Amount">
            <Input
              type="number"
              name="targetAmount"
              step="0.01"
              min="0"
              required
              value={goalFormTargetAmount}
              onChange={(e) => setGoalFormTargetAmount(e.target.value)}
              placeholder="0.00"
            />
          </FormField>

          <FormField label={editingGoal ? 'Current Saved Amount' : 'Initial Amount'} hint={editingGoal ? 'Current progress amount' : 'Optional starting saved amount'}>
            <Input
              type="number"
              name="initialAmount"
              step="0.01"
              min="0"
              value={goalFormInitialAmount}
              onChange={(e) => setGoalFormInitialAmount(e.target.value)}
              disabled={!!editingGoal}
              placeholder="0.00"
            />
          </FormField>

          <FormField label="Target Date" hint="Optional">
            <Input
              type="date"
              name="deadline"
              value={goalFormDeadline}
              onChange={(e) => setGoalFormDeadline(e.target.value)}
            />
          </FormField>

          {(() => {
            const target = parseFloat(goalFormTargetAmount || '0')
            const current = parseFloat(goalFormInitialAmount || '0')
            const needs = calculateNeedFromInputs(target, current, goalFormDeadline)
            if (!needs) return null

            return (
              <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-1.5 bg-zinc-50/60 dark:bg-zinc-800/30">
                <div className="flex justify-between text-xs">
                  <span className="text-zinc-500 dark:text-zinc-400">Need/week</span>
                  <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">{formatConverted(needs.weekly)}</span>
                </div>
                <div className="flex justify-between text-xs">
                  <span className="text-zinc-500 dark:text-zinc-400">Need/month</span>
                  <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">{formatConverted(needs.monthly)}</span>
                </div>
              </div>
            )
          })()}

          <FormField label="Notes" hint="Optional">
            <Textarea
              name="notes"
              rows={2}
              defaultValue={editingGoal?.notes || ''}
              placeholder="Additional notes..."
            />
          </FormField>
        </form>
      </Modal>

      {/* Update Progress Modal */}
      <Modal
        open={isProgressModalOpen}
        onClose={closeProgressModal}
        title="Update Progress"
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={closeProgressModal}>Cancel</Button>
            <Button
              type="submit"
              form="progress-form"
              loading={updateProgressMutation.isPending}
            >
              Update
            </Button>
          </div>
        }
      >
        <form id="progress-form" onSubmit={handleProgressSubmit} className="space-y-4">
          {selectedGoal && (
            <p className="text-sm text-zinc-600 dark:text-zinc-400">
              Current:{' '}
              <strong className="text-zinc-900 dark:text-zinc-100">
                  {formatConverted(selectedGoal.currentAmount)}
                </strong>{' '}
              / {formatConverted(selectedGoal.targetAmount)}
            </p>
          )}
          <FormField label="Add / Remove Amount" hint="Use negative value to take money out">
            <Input
              type="number"
              name="amountDelta"
              step="0.01"
              required
              defaultValue=""
              placeholder="e.g. 250 or -100"
            />
          </FormField>
        </form>
      </Modal>
    </div>
  )
}

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { reportApi, budgetApi, goalApi } from '../api'
import { moneyToNumber } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import { useCurrency } from '../store/CurrencyContext'
import type { Money } from '../types'
import {
  TrendingUp,
  TrendingDown,
  Wallet,
  Target,
  PiggyBank,
  ArrowUpRight,
  ArrowDownRight,
  Pencil,
  Sparkles,
  CheckCircle2,
} from 'lucide-react'
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
} from 'recharts'
import { MetricCard, BentoCard, Modal, ProgressBar, Button, Input, FormField, PageSpinner } from '../components/ui'

const CHART_COLORS = [
  '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b',
  '#ef4444', '#ec4899', '#3b82f6', '#84cc16',
]

// Custom chart tooltip
function ChartTooltip({ active, payload, label, formatter }: {
  active?: boolean
  payload?: { name: string; value: number; color: string }[]
  label?: string
  formatter?: (value: number) => string
}) {
  if (!active || !payload?.length) return null
  return (
    <div className="bg-zinc-900 dark:bg-zinc-800 border border-zinc-700 rounded-xl px-3 py-2 shadow-xl text-xs">
      {label && <p className="text-zinc-400 mb-1">{label}</p>}
      {payload.map((p, i) => (
        <p key={i} className="font-semibold" style={{ color: p.color }}>
          {p.name}: {formatter ? formatter(p.value) : p.value}
        </p>
      ))}
    </div>
  )
}

export default function Dashboard() {
  const { user } = useAuth()
  const { formatConverted } = useCurrency()
  const queryClient = useQueryClient()
  const currentDate = new Date()
  const currentMonth = currentDate.getMonth() + 1
  const currentYear = currentDate.getFullYear()

  const [isGoalModalOpen, setIsGoalModalOpen] = useState(false)

  const { data: monthlyReport, isLoading: isLoadingReport } = useQuery({
    queryKey: ['monthlyReport', currentYear, currentMonth],
    queryFn: () => reportApi.getMonthlyReport(currentYear, currentMonth),
  })
  const { data: netWorthReport, isLoading: isLoadingNetWorth } = useQuery({
    queryKey: ['netWorthReport'],
    queryFn: () => reportApi.getNetWorthReport(),
  })
  const { data: budgetStatuses, isLoading: isLoadingBudgets } = useQuery({
    queryKey: ['budgetStatuses'],
    queryFn: () => budgetApi.getAllStatuses(),
  })
  const { data: goalProgress, isLoading: isLoadingGoals } = useQuery({
    queryKey: ['goalProgress'],
    queryFn: () => goalApi.getAllProgress(),
  })
  const { data: netWorthGoalProgress, isLoading: isLoadingNetWorthGoal } = useQuery({
    queryKey: ['netWorthGoalProgress'],
    queryFn: () => goalApi.getNetWorthGoalProgress(),
  })

  const setNetWorthGoalMutation = useMutation({
    mutationFn: (data: { name: string; targetAmount: Money; notes?: string }) =>
      goalApi.setNetWorthGoal(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['netWorthGoalProgress'] })
      setIsGoalModalOpen(false)
    },
  })
  const deleteNetWorthGoalMutation = useMutation({
    mutationFn: () => goalApi.deleteNetWorthGoal(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['netWorthGoalProgress'] }),
  })

  const isLoading = isLoadingReport || isLoadingNetWorth || isLoadingBudgets || isLoadingGoals || isLoadingNetWorthGoal

  const netWorthGoal = netWorthGoalProgress?.goal || null
  const currentNetWorth = moneyToNumber(
    netWorthGoalProgress?.currentNetWorth || netWorthReport?.netWorth || { amount: '0', currency: 'SGD' }
  )
  const monthlySavings = moneyToNumber(monthlyReport?.netSavings || { amount: '0', currency: 'SGD' })
  const goalProgress_pct = netWorthGoalProgress?.percentageComplete ?? 0
  const amountRemaining = moneyToNumber(
    netWorthGoalProgress?.amountRemaining || { amount: '0', currency: 'SGD' }
  )
  const monthsToGoal = netWorthGoalProgress?.estimatedMonthsToGoal ?? null
  const savingsRate = monthlyReport?.savingsRate ?? 0
  const currency = user?.baseCurrency || 'SGD'

  const spendingData = monthlyReport?.spendingByCategory?.map((cat, i) => ({
    name: cat.categoryName,
    value: moneyToNumber(cat.amount),
    color: CHART_COLORS[i % CHART_COLORS.length],
  })) || []

  const handleGoalSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    const targetAmount = parseFloat(fd.get('targetAmount') as string)
    const name = fd.get('name') as string
    if (targetAmount > 0) {
      setNetWorthGoalMutation.mutate({
        name: name || 'Net Worth Goal',
        targetAmount: { amount: targetAmount.toString(), currency },
      })
    }
  }

  const monthName = new Date(currentYear, currentMonth - 1).toLocaleString('default', { month: 'long' })

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-end justify-between">
        <div>
          <p className="text-xs font-medium tracking-wide text-zinc-400 dark:text-zinc-500 uppercase mb-1">
            {monthName} {currentYear}
          </p>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
            Good {currentDate.getHours() < 12 ? 'morning' : currentDate.getHours() < 18 ? 'afternoon' : 'evening'},{' '}
            {user?.name?.split(' ')[0]}
          </h1>
        </div>
      </div>

      {isLoading ? (
        <PageSpinner />
      ) : (
        <>
          {/* ── Bento Grid ────────────────────────────────── */}
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">

            {/* Row 1: 4 metric cards */}
            <MetricCard
              label="Monthly Income"
              value={formatConverted(monthlyReport?.totalIncome)}
              icon={TrendingUp}
              accent="emerald"
            />
            <MetricCard
              label="Monthly Expenses"
              value={formatConverted(monthlyReport?.totalExpenses)}
              icon={TrendingDown}
              accent="red"
            />
            <MetricCard
              label="Net Savings"
              value={formatConverted(monthlyReport?.netSavings)}
              icon={PiggyBank}
              accent="violet"
              subtext={`${savingsRate >= 0 ? '▲' : '▼'} ${Math.abs(savingsRate).toFixed(1)}% savings rate`}
              subtextPositive={savingsRate >= 0}
            />
            <MetricCard
              label="Net Worth"
              value={formatConverted(netWorthReport?.netWorth)}
              icon={Wallet}
              accent="blue"
            />
          </div>

          {/* Row 2: Net Worth Goal (wide) + Spending Donut */}
          <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">

            {/* Net Worth Goal — spans 3/5 */}
            <BentoCard colSpan={1} className="lg:col-span-3 p-0 overflow-hidden" hover={false}>
              <div className="relative bg-gradient-to-br from-violet-600 via-violet-700 to-indigo-800 dark:from-violet-700 dark:via-violet-800 dark:to-indigo-900 rounded-2xl p-6 h-full min-h-[220px]">
                {/* subtle noise overlay */}
                <div className="absolute inset-0 rounded-2xl opacity-[0.03] bg-[url('data:image/svg+xml,%3Csvg viewBox%3D%220 0 256 256%22 xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%3Cfilter id%3D%22noise%22%3E%3CfeTurbulence type%3D%22fractalNoise%22 baseFrequency%3D%220.9%22 numOctaves%3D%224%22 stitchTiles%3D%22stitch%22%2F%3E%3C%2Ffilter%3E%3Crect width%3D%22100%25%22 height%3D%22100%25%22 filter%3D%22url(%23noise)%22%2F%3E%3C%2Fsvg%3E')]" />

                <div className="relative flex items-start justify-between mb-5">
                  <div className="flex items-center gap-3">
                    <div className="h-9 w-9 rounded-xl bg-white/15 flex items-center justify-center">
                      <Sparkles className="h-4 w-4 text-white" />
                    </div>
                    <div>
                      <p className="text-xs font-medium text-white/60 uppercase tracking-wide">Net Worth Goal</p>
                      <h2 className="text-base font-semibold text-white leading-snug">
                        {netWorthGoal ? netWorthGoal.name : 'No goal set yet'}
                      </h2>
                    </div>
                  </div>
                  <button
                    onClick={() => setIsGoalModalOpen(true)}
                    className="h-8 w-8 flex items-center justify-center rounded-lg bg-white/10 text-white/70 hover:bg-white/20 hover:text-white transition-all duration-150"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                </div>

                {netWorthGoal ? (
                  <div className="relative space-y-4">
                    <div className="grid grid-cols-3 gap-3">
                      {[
                        { label: 'Current', value: formatConverted({ amount: currentNetWorth.toString(), currency }) },
                        { label: 'Target',  value: formatConverted(netWorthGoal.targetAmount) },
                        { label: 'Remaining', value: formatConverted({ amount: amountRemaining.toString(), currency }) },
                      ].map((s) => (
                        <div key={s.label} className="bg-white/10 rounded-xl px-3 py-2.5">
                          <p className="text-xs text-white/60 mb-0.5">{s.label}</p>
                          <p className="text-sm font-semibold text-white">{s.value}</p>
                        </div>
                      ))}
                    </div>

                    <div>
                      <div className="flex justify-between text-xs text-white/70 mb-1.5">
                        <span>Progress</span>
                        <span className="font-semibold text-white">{goalProgress_pct.toFixed(1)}%</span>
                      </div>
                      <div className="h-1.5 bg-white/20 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-white rounded-full transition-all duration-700"
                          style={{ width: `${Math.min(goalProgress_pct, 100)}%` }}
                        />
                      </div>
                    </div>

                    {goalProgress_pct >= 100 ? (
                      <p className="flex items-center gap-1.5 text-xs font-semibold text-white">
                        <CheckCircle2 className="h-3.5 w-3.5" /> Goal reached — congratulations!
                      </p>
                    ) : monthsToGoal !== null && monthsToGoal > 0 ? (
                      <p className="text-xs text-white/70">
                        At{' '}
                        <span className="text-white font-medium">
                          {formatConverted({ amount: monthlySavings.toString(), currency })}/mo
                        </span>{' '}
                        savings — roughly{' '}
                        <span className="text-white font-medium">{monthsToGoal} months</span> to go
                      </p>
                    ) : (
                      <p className="text-xs text-white/70">Start saving to see your estimated timeline</p>
                    )}
                  </div>
                ) : (
                  <div className="relative flex flex-col items-start gap-3">
                    <p className="text-sm text-white/70 max-w-xs">
                      Set a net worth target to track your progress towards major financial milestones.
                    </p>
                    <button
                      onClick={() => setIsGoalModalOpen(true)}
                      className="px-4 py-1.5 bg-white text-violet-700 text-sm font-semibold rounded-lg hover:bg-white/90 transition-colors"
                    >
                      Set goal
                    </button>
                  </div>
                )}
              </div>
            </BentoCard>

            {/* Spending Donut — spans 2/5 */}
            <BentoCard colSpan={1} className="lg:col-span-2 p-6">
              <p className="text-xs font-medium tracking-wide text-zinc-400 dark:text-zinc-500 uppercase mb-4">
                Spending by Category
              </p>
              {spendingData.length > 0 ? (
                <>
                  <div className="h-44">
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={spendingData}
                          cx="50%"
                          cy="50%"
                          innerRadius={52}
                          outerRadius={72}
                          paddingAngle={2}
                          dataKey="value"
                          strokeWidth={0}
                        >
                          {spendingData.map((entry, i) => (
                            <Cell key={i} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip
                          content={<ChartTooltip formatter={(v) => formatConverted({ amount: v.toString(), currency })} />}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>
                  <div className="mt-3 space-y-1.5">
                    {spendingData.slice(0, 4).map((item, i) => (
                      <div key={i} className="flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: item.color }} />
                          <span className="text-zinc-600 dark:text-zinc-400 truncate max-w-[100px]">{item.name}</span>
                        </div>
                         <span className="font-medium text-zinc-900 dark:text-zinc-100">
                           {formatConverted({ amount: item.value.toString(), currency })}
                         </span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="h-44 flex items-center justify-center">
                  <p className="text-sm text-zinc-400 dark:text-zinc-500">No spending data this month</p>
                </div>
              )}
            </BentoCard>
          </div>

          {/* Row 3: Budget Status + Saving Goals Bar Chart */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">

            {/* Budget Status */}
            <BentoCard className="p-6">
              <div className="flex items-center justify-between mb-5">
                <p className="text-xs font-medium tracking-wide text-zinc-400 dark:text-zinc-500 uppercase">
                  Budget Status
                </p>
                <Target className="h-4 w-4 text-zinc-300 dark:text-zinc-600" />
              </div>
              {budgetStatuses && budgetStatuses.length > 0 ? (
                <div className="space-y-4">
                  {budgetStatuses.slice(0, 5).map((status) => {
                    const pct = status.percentageUsed
                    const variant = pct > 100 ? 'danger' : pct > 80 ? 'warning' : 'success'
                    return (
                      <div key={status.budget.id}>
                        <div className="flex justify-between items-center mb-1.5">
                          <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                            {status.budget.categoryName}
                          </span>
                           <span className="text-xs text-zinc-500 dark:text-zinc-400">
                             {formatConverted(status.spent)}{' '}
                             <span className="text-zinc-400 dark:text-zinc-600">/</span>{' '}
                             {formatConverted(status.budget.amount)}
                           </span>
                        </div>
                        <ProgressBar value={pct} variant={variant} />
                      </div>
                    )
                  })}
                </div>
              ) : (
                <div className="py-10 text-center">
                  <p className="text-sm text-zinc-400 dark:text-zinc-500">No budgets set up yet</p>
                </div>
              )}
            </BentoCard>

            {/* Saving Goals */}
            <BentoCard className="p-6">
              <div className="flex items-center justify-between mb-5">
                <p className="text-xs font-medium tracking-wide text-zinc-400 dark:text-zinc-500 uppercase">
                  Saving Goals
                </p>
                <div className="flex items-center gap-1 text-xs text-zinc-400">
                  {savingsRate >= 0
                    ? <ArrowUpRight className="h-3.5 w-3.5 text-emerald-500" />
                    : <ArrowDownRight className="h-3.5 w-3.5 text-red-500" />
                  }
                </div>
              </div>
              {goalProgress && goalProgress.length > 0 ? (
                <div className="space-y-4">
                  {goalProgress.slice(0, 5).map((progress) => {
                    const pct = progress.percentageComplete
                    const variant = pct >= 100 ? 'success' : progress.isOnTrack ? 'default' : 'warning'
                    return (
                      <div key={progress.goal.id}>
                        <div className="flex justify-between items-center mb-1.5">
                          <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300 truncate pr-2">
                            {progress.goal.name}
                          </span>
                          <span className="text-xs text-zinc-500 dark:text-zinc-400 whitespace-nowrap">
                            {pct.toFixed(1)}%
                          </span>
                        </div>
                        <ProgressBar value={pct} variant={variant} />
                      </div>
                    )
                  })}
                </div>
              ) : (
                <div className="py-10 text-center">
                  <p className="text-sm text-zinc-400 dark:text-zinc-500">No saving goals set up yet</p>
                </div>
              )}
            </BentoCard>
          </div>
        </>
      )}

      {/* ── Net Worth Goal Modal ───────────────────────────── */}
      <Modal
        open={isGoalModalOpen}
        onClose={() => setIsGoalModalOpen(false)}
        title={netWorthGoal ? 'Edit Net Worth Goal' : 'Set Net Worth Goal'}
        footer={
          <div className="flex items-center gap-2">
            {netWorthGoal && (
              <Button
                variant="danger"
                size="sm"
                loading={deleteNetWorthGoalMutation.isPending}
                onClick={() => {
                  if (confirm('Delete this net worth goal?')) deleteNetWorthGoalMutation.mutate()
                }}
              >
                Delete
              </Button>
            )}
            <div className="flex-1" />
            <Button variant="secondary" size="sm" onClick={() => setIsGoalModalOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              size="sm"
              form="goal-form"
              type="submit"
              loading={setNetWorthGoalMutation.isPending}
            >
              {netWorthGoal ? 'Update' : 'Set Goal'}
            </Button>
          </div>
        }
      >
        <form id="goal-form" onSubmit={handleGoalSubmit} className="space-y-4">
          <FormField label="Goal Name" htmlFor="goal-name">
            <Input
              id="goal-name"
              name="name"
              defaultValue={netWorthGoal?.name || ''}
              placeholder="e.g., House Down Payment, Retirement"
            />
          </FormField>
          <FormField label={`Target Net Worth (${currency})`} htmlFor="goal-target">
            <Input
              id="goal-target"
              type="number"
              name="targetAmount"
              required
              min="1"
              step="1"
              defaultValue={netWorthGoal ? moneyToNumber(netWorthGoal.targetAmount) : ''}
              placeholder="500000"
            />
          </FormField>
          <div className="rounded-xl bg-zinc-50 dark:bg-zinc-800/60 border border-zinc-200 dark:border-zinc-700 px-4 py-3 space-y-1 text-sm">
            <div className="flex justify-between">
              <span className="text-zinc-500 dark:text-zinc-400">Current Net Worth</span>
              <span className="font-medium text-zinc-900 dark:text-zinc-100">
                  {formatConverted({ amount: currentNetWorth.toString(), currency })}
                </span>
              </div>
              {monthlySavings > 0 && (
                <div className="flex justify-between">
                  <span className="text-zinc-500 dark:text-zinc-400">Monthly Savings</span>
                  <span className="font-medium text-zinc-900 dark:text-zinc-100">
                    {formatConverted({ amount: monthlySavings.toString(), currency })}
                  </span>
              </div>
            )}
          </div>
        </form>
      </Modal>
    </div>
  )
}

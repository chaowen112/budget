import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { reportApi, budgetApi, goalApi } from '../api'
import { formatMoney, moneyToNumber } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import type { Money } from '../types'
import {
  TrendingUp,
  TrendingDown,
  Wallet,
  Target,
  PiggyBank,
  ArrowUpRight,
  ArrowDownRight,
  Home,
  Pencil,
  X,
} from 'lucide-react'
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
} from 'recharts'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16']

export default function Dashboard() {
  const { user } = useAuth()
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

  // Net Worth Goal API queries
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['netWorthGoalProgress'] })
    },
  })

  const isLoading = isLoadingReport || isLoadingNetWorth || isLoadingBudgets || isLoadingGoals || isLoadingNetWorthGoal

  // Net worth goal calculations from backend
  const netWorthGoal = netWorthGoalProgress?.goal || null
  const currentNetWorth = moneyToNumber(netWorthGoalProgress?.currentNetWorth || netWorthReport?.netWorth || { amount: '0', currency: 'SGD' })
  const monthlySavings = moneyToNumber(monthlyReport?.netSavings || { amount: '0', currency: 'SGD' })
  
  const goalProgress_pct = netWorthGoalProgress?.percentageComplete ?? 0
  const amountRemaining = moneyToNumber(netWorthGoalProgress?.amountRemaining || { amount: '0', currency: 'SGD' })
  const monthsToGoal = netWorthGoalProgress?.estimatedMonthsToGoal ?? null

  const handleGoalSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const targetAmount = parseFloat(formData.get('targetAmount') as string)
    const name = formData.get('name') as string

    if (targetAmount > 0) {
      setNetWorthGoalMutation.mutate({
        name: name || 'Net Worth Goal',
        targetAmount: {
          amount: targetAmount.toString(),
          currency: user?.baseCurrency || 'SGD',
        },
      })
    }
  }

  const handleDeleteGoal = () => {
    if (confirm('Delete this net worth goal?')) {
      deleteNetWorthGoalMutation.mutate()
    }
  }

  const spendingData = monthlyReport?.spendingByCategory?.map((cat, index) => ({
    name: cat.categoryName,
    value: moneyToNumber(cat.amount),
    color: COLORS[index % COLORS.length],
  })) || []

  const savingsRate = monthlyReport?.savingsRate ?? 0

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-500">Welcome back, {user?.name}</p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : (
        <>
          {/* Stats Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {/* Total Income */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Monthly Income</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {formatMoney(monthlyReport?.totalIncome)}
                  </p>
                </div>
                <div className="h-12 w-12 bg-green-100 rounded-full flex items-center justify-center">
                  <TrendingUp className="h-6 w-6 text-green-600" />
                </div>
              </div>
            </div>

            {/* Total Expenses */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Monthly Expenses</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {formatMoney(monthlyReport?.totalExpenses)}
                  </p>
                </div>
                <div className="h-12 w-12 bg-red-100 rounded-full flex items-center justify-center">
                  <TrendingDown className="h-6 w-6 text-red-600" />
                </div>
              </div>
            </div>

            {/* Net Savings */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Net Savings</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {formatMoney(monthlyReport?.netSavings)}
                  </p>
                  <p className={`text-sm ${savingsRate >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                    {savingsRate >= 0 ? <ArrowUpRight className="inline h-4 w-4" /> : <ArrowDownRight className="inline h-4 w-4" />}
                    {Math.abs(savingsRate).toFixed(1)}% savings rate
                  </p>
                </div>
                <div className="h-12 w-12 bg-blue-100 rounded-full flex items-center justify-center">
                  <PiggyBank className="h-6 w-6 text-blue-600" />
                </div>
              </div>
            </div>

            {/* Net Worth */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Net Worth</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {formatMoney(netWorthReport?.netWorth)}
                  </p>
                </div>
                <div className="h-12 w-12 bg-purple-100 rounded-full flex items-center justify-center">
                  <Wallet className="h-6 w-6 text-purple-600" />
                </div>
              </div>
            </div>
          </div>

          {/* Net Worth Goal Widget */}
          <div className="bg-gradient-to-r from-indigo-500 to-purple-600 rounded-xl p-6 shadow-sm text-white">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 bg-white/20 rounded-full flex items-center justify-center">
                  <Home className="h-5 w-5" />
                </div>
                <div>
                  <h2 className="text-lg font-semibold">
                    {netWorthGoal ? netWorthGoal.name : 'Net Worth Goal'}
                  </h2>
                  <p className="text-sm text-white/80">
                    {netWorthGoal ? 'Track your progress' : 'Set a target to track'}
                  </p>
                </div>
              </div>
              <button
                onClick={() => setIsGoalModalOpen(true)}
                className="p-2 hover:bg-white/20 rounded-lg transition-colors"
              >
                <Pencil className="h-5 w-5" />
              </button>
            </div>

            {netWorthGoal ? (
              <>
                <div className="grid grid-cols-3 gap-4 mb-4">
                  <div>
                    <p className="text-sm text-white/70">Current</p>
                    <p className="text-xl font-bold">
                      {formatMoney({ amount: currentNetWorth.toString(), currency: user?.baseCurrency || 'SGD' })}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-white/70">Target</p>
                    <p className="text-xl font-bold">
                      {formatMoney(netWorthGoal.targetAmount)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-white/70">Remaining</p>
                    <p className="text-xl font-bold">
                      {formatMoney({ amount: amountRemaining.toString(), currency: user?.baseCurrency || 'SGD' })}
                    </p>
                  </div>
                </div>

                <div className="mb-2">
                  <div className="flex justify-between text-sm mb-1">
                    <span>Progress</span>
                    <span className="font-semibold">{goalProgress_pct.toFixed(1)}%</span>
                  </div>
                  <div className="h-3 bg-white/20 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-white rounded-full transition-all duration-500"
                      style={{ width: `${goalProgress_pct}%` }}
                    />
                  </div>
                </div>

                {monthsToGoal !== null && monthsToGoal > 0 && (
                  <p className="text-sm text-white/80 mt-3">
                    At your current savings rate ({formatMoney({ amount: monthlySavings.toString(), currency: user?.baseCurrency || 'SGD' })}/month), 
                    you'll reach your goal in approximately <strong>{monthsToGoal} months</strong> ({(monthsToGoal / 12).toFixed(1)} years)
                  </p>
                )}
                {monthsToGoal === -1 && (
                  <p className="text-sm text-white/80 mt-3">
                    Start saving to see your estimated time to reach this goal
                  </p>
                )}
                {goalProgress_pct >= 100 && (
                  <p className="text-sm font-semibold mt-3">
                    Congratulations! You've reached your net worth goal!
                  </p>
                )}
              </>
            ) : (
              <div className="text-center py-4">
                <p className="text-white/80 mb-3">
                  Set a net worth goal to track your progress towards major financial milestones like buying a house
                </p>
                <button
                  onClick={() => setIsGoalModalOpen(true)}
                  className="px-4 py-2 bg-white text-indigo-600 rounded-lg font-medium hover:bg-white/90 transition-colors"
                >
                  Set Goal
                </button>
              </div>
            )}
          </div>

          {/* Charts Row */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Spending by Category */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Spending by Category</h2>
              {spendingData.length > 0 ? (
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={spendingData}
                        cx="50%"
                        cy="50%"
                        innerRadius={60}
                        outerRadius={80}
                        paddingAngle={2}
                        dataKey="value"
                      >
                        {spendingData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Pie>
                      <Tooltip
                        formatter={(value: number | undefined) => value !== undefined ? formatMoney({ amount: value.toString(), currency: user?.baseCurrency || 'SGD' }) : ''}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <div className="h-64 flex items-center justify-center text-gray-500">
                  No spending data for this month
                </div>
              )}
              {/* Legend */}
              <div className="mt-4 grid grid-cols-2 gap-2">
                {spendingData.slice(0, 6).map((item, index) => (
                  <div key={index} className="flex items-center text-sm">
                    <div className="w-3 h-3 rounded-full mr-2" style={{ backgroundColor: item.color }} />
                    <span className="truncate text-gray-600">{item.name}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Budget Status */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Budget Status</h2>
              {budgetStatuses && budgetStatuses.length > 0 ? (
                <div className="space-y-4">
                  {budgetStatuses.slice(0, 5).map((status) => (
                    <div key={status.budget.id}>
                      <div className="flex justify-between text-sm mb-1">
                        <span className="text-gray-600">{status.budget.categoryName}</span>
                        <span className="text-gray-900">
                          {formatMoney(status.spent)} / {formatMoney(status.budget.amount)}
                        </span>
                      </div>
                      <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                        <div
                          className={`h-full rounded-full ${
                            status.percentageUsed > 100
                              ? 'bg-red-500'
                              : status.percentageUsed > 80
                              ? 'bg-yellow-500'
                              : 'bg-green-500'
                          }`}
                          style={{ width: `${Math.min(status.percentageUsed, 100)}%` }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="h-64 flex items-center justify-center text-gray-500">
                  No budgets set up yet
                </div>
              )}
            </div>
          </div>

          {/* Goals Progress */}
          <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">Saving Goals</h2>
              <Target className="h-5 w-5 text-gray-400" />
            </div>
            {goalProgress && goalProgress.length > 0 ? (
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={goalProgress.slice(0, 5).map((g) => ({
                    name: g.goal.name,
                    progress: g.percentageComplete,
                    target: 100,
                  }))}>
                    <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                    <YAxis domain={[0, 100]} tick={{ fontSize: 12 }} />
                    <Tooltip formatter={(value: number | undefined) => value !== undefined ? `${value.toFixed(1)}%` : ''} />
                    <Bar dataKey="progress" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            ) : (
              <div className="h-64 flex items-center justify-center text-gray-500">
                No saving goals set up yet
              </div>
            )}
          </div>
        </>
      )}

      {/* Net Worth Goal Modal */}
      {isGoalModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-md">
            <div className="flex items-center justify-between p-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold">
                {netWorthGoal ? 'Edit Net Worth Goal' : 'Set Net Worth Goal'}
              </h2>
              <button
                onClick={() => setIsGoalModalOpen(false)}
                className="p-2 text-gray-400 hover:text-gray-600"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleGoalSubmit} className="p-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Goal Name
                </label>
                <input
                  type="text"
                  name="name"
                  defaultValue={netWorthGoal?.name || ''}
                  placeholder="e.g., House Down Payment, Retirement"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Target Net Worth ({user?.baseCurrency || 'SGD'})
                </label>
                <input
                  type="number"
                  name="targetAmount"
                  required
                  min="1"
                  step="1"
                  defaultValue={netWorthGoal ? moneyToNumber(netWorthGoal.targetAmount) : ''}
                  placeholder="e.g., 500000"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                />
              </div>
              <div className="bg-gray-50 rounded-lg p-3 text-sm text-gray-600">
                <p>
                  <strong>Current Net Worth:</strong>{' '}
                  {formatMoney({ amount: currentNetWorth.toString(), currency: user?.baseCurrency || 'SGD' })}
                </p>
                {monthlySavings > 0 && (
                  <p className="mt-1">
                    <strong>Monthly Savings:</strong>{' '}
                    {formatMoney({ amount: monthlySavings.toString(), currency: user?.baseCurrency || 'SGD' })}
                  </p>
                )}
              </div>
              <div className="flex gap-3 pt-2">
                {netWorthGoal && (
                  <button
                    type="button"
                    onClick={handleDeleteGoal}
                    disabled={deleteNetWorthGoalMutation.isPending}
                    className="px-4 py-2 text-red-600 hover:bg-red-50 rounded-lg disabled:opacity-50"
                  >
                    {deleteNetWorthGoalMutation.isPending ? 'Deleting...' : 'Delete'}
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setIsGoalModalOpen(false)}
                  className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={setNetWorthGoalMutation.isPending}
                  className="flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50"
                >
                  {setNetWorthGoalMutation.isPending ? 'Saving...' : netWorthGoal ? 'Update' : 'Set Goal'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

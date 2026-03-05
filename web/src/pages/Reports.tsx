import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { reportApi } from '../api'
import type { BudgetTrackingReport, SavingGoalReport } from '../api'
import { formatMoney, moneyToNumber, getMonthName } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
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
  Legend,
} from 'recharts'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16']

export default function Reports() {
  const { user } = useAuth()
  const currentDate = new Date()
  const [selectedMonth, setSelectedMonth] = useState(currentDate.getMonth() + 1)
  const [selectedYear, setSelectedYear] = useState(currentDate.getFullYear())

  const { data: monthlyReport, isLoading: isLoadingMonthly } = useQuery({
    queryKey: ['monthlyReport', selectedYear, selectedMonth],
    queryFn: () => reportApi.getMonthlyReport(selectedYear, selectedMonth),
  })

  const { data: netWorthReport, isLoading: isLoadingNetWorth } = useQuery({
    queryKey: ['netWorthReport'],
    queryFn: () => reportApi.getNetWorthReport(),
  })

  const { data: budgetReport, isLoading: isLoadingBudget } = useQuery<BudgetTrackingReport>({
    queryKey: ['budgetTrackingReport'],
    queryFn: () => reportApi.getBudgetTrackingReport(),
  })

  const { data: goalsReport, isLoading: isLoadingGoals } = useQuery<SavingGoalReport[]>({
    queryKey: ['goalsReport'],
    queryFn: () => reportApi.getGoalsReport(),
  })

  const isLoading = isLoadingMonthly || isLoadingNetWorth || isLoadingBudget || isLoadingGoals

  const spendingData = monthlyReport?.spendingByCategory?.map((cat, index) => ({
    name: cat.categoryName,
    value: moneyToNumber(cat.amount),
    color: COLORS[index % COLORS.length],
    count: cat.transactionCount,
  })) || []

  const budgetData = budgetReport?.categoryDetails?.slice(0, 8).map((item) => ({
    name: item.categoryName?.substring(0, 12) || 'Unknown',
    budget: moneyToNumber(item.budgeted),
    spent: moneyToNumber(item.spent),
  })) || []

  const goalsData = goalsReport?.map((goal) => ({
    name: goal.goalName,
    progress: goal.percentageComplete,
    remaining: 100 - goal.percentageComplete,
  })) || []

  const assetBreakdown = netWorthReport?.assetBreakdown?.map((item, index) => ({
    name: item.category.replace('ASSET_CATEGORY_', ''),
    value: moneyToNumber(item.total),
    color: COLORS[index % COLORS.length],
  })) || []

  const years = Array.from({ length: 5 }, (_, i) => currentDate.getFullYear() - i)
  const months = Array.from({ length: 12 }, (_, i) => ({ value: i + 1, label: getMonthName(i + 1) }))

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Reports</h1>
        <p className="text-gray-500">Financial insights and analytics</p>
      </div>

      {/* Period Selector */}
      <div className="flex gap-4 bg-white p-4 rounded-xl border border-gray-100">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Month</label>
          <select
            value={selectedMonth}
            onChange={(e) => setSelectedMonth(parseInt(e.target.value))}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            {months.map((m) => (
              <option key={m.value} value={m.value}>{m.label}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Year</label>
          <select
            value={selectedYear}
            onChange={(e) => setSelectedYear(parseInt(e.target.value))}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            {years.map((y) => (
              <option key={y} value={y}>{y}</option>
            ))}
          </select>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : (
        <>
          {/* Monthly Summary */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <p className="text-sm text-gray-500">Income</p>
              <p className="text-2xl font-bold text-green-600">{formatMoney(monthlyReport?.totalIncome)}</p>
            </div>
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <p className="text-sm text-gray-500">Expenses</p>
              <p className="text-2xl font-bold text-red-600">{formatMoney(monthlyReport?.totalExpenses)}</p>
            </div>
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <p className="text-sm text-gray-500">Net Savings</p>
              <p className={`text-2xl font-bold ${moneyToNumber(monthlyReport?.netSavings || { amount: '0', currency: 'SGD' }) >= 0 ? 'text-blue-600' : 'text-red-600'}`}>
                {formatMoney(monthlyReport?.netSavings)}
              </p>
            </div>
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <p className="text-sm text-gray-500">Savings Rate</p>
              <p className={`text-2xl font-bold ${(monthlyReport?.savingsRate || 0) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                {(monthlyReport?.savingsRate || 0).toFixed(1)}%
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Spending by Category */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Spending by Category</h2>
              {spendingData.length > 0 ? (
                <>
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={spendingData}
                          cx="50%"
                          cy="50%"
                          innerRadius={50}
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
                  <div className="mt-4 space-y-2">
                    {spendingData.map((item, index) => (
                      <div key={index} className="flex items-center justify-between text-sm">
                        <div className="flex items-center">
                          <div className="w-3 h-3 rounded-full mr-2" style={{ backgroundColor: item.color }} />
                          <span className="text-gray-600">{item.name}</span>
                        </div>
                        <span className="font-medium text-gray-900">
                          {formatMoney({ amount: item.value.toString(), currency: user?.baseCurrency || 'SGD' })}
                        </span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="h-64 flex items-center justify-center text-gray-500">
                  No spending data for this period
                </div>
              )}
            </div>

            {/* Budget vs Actual */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Budget vs Actual</h2>
              {budgetData.length > 0 ? (
                <div className="h-80">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={budgetData} layout="vertical">
                      <XAxis type="number" tick={{ fontSize: 12 }} />
                      <YAxis type="category" dataKey="name" tick={{ fontSize: 11 }} width={80} />
                      <Tooltip
                        formatter={(value: number | undefined) => value !== undefined ? formatMoney({ amount: value.toString(), currency: user?.baseCurrency || 'SGD' }) : ''}
                      />
                      <Legend />
                      <Bar dataKey="budget" fill="#e5e7eb" name="Budget" />
                      <Bar dataKey="spent" fill="#3b82f6" name="Spent" />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <div className="h-80 flex items-center justify-center text-gray-500">
                  No budget data available
                </div>
              )}
            </div>

            {/* Goals Progress */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Goals Progress</h2>
              {goalsData.length > 0 ? (
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={goalsData}>
                      <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                      <YAxis domain={[0, 100]} tick={{ fontSize: 12 }} />
                      <Tooltip formatter={(value: number | undefined) => value !== undefined ? `${value.toFixed(1)}%` : ''} />
                      <Bar dataKey="progress" stackId="a" fill="#10b981" name="Progress" />
                      <Bar dataKey="remaining" stackId="a" fill="#e5e7eb" name="Remaining" />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <div className="h-64 flex items-center justify-center text-gray-500">
                  No goals set up yet
                </div>
              )}
            </div>

            {/* Asset Allocation */}
            <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Asset Allocation</h2>
              <div className="grid grid-cols-2 gap-4 mb-4">
                <div className="text-center p-4 bg-gray-50 rounded-lg">
                  <p className="text-sm text-gray-500">Total Assets</p>
                  <p className="text-xl font-bold text-green-600">{formatMoney(netWorthReport?.totalAssets)}</p>
                </div>
                <div className="text-center p-4 bg-gray-50 rounded-lg">
                  <p className="text-sm text-gray-500">Net Worth</p>
                  <p className="text-xl font-bold text-blue-600">{formatMoney(netWorthReport?.netWorth)}</p>
                </div>
              </div>
              {assetBreakdown.length > 0 ? (
                <div className="h-48">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={assetBreakdown}
                        cx="50%"
                        cy="50%"
                        outerRadius={60}
                        dataKey="value"
                        label={({ name }) => name}
                      >
                        {assetBreakdown.map((entry, index) => (
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
                <div className="h-48 flex items-center justify-center text-gray-500">
                  No asset data available
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  )
}

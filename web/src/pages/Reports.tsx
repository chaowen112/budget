import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQueries, useQuery } from '@tanstack/react-query'
import { reportApi } from '../api'
import type { BudgetTrackingReport, SavingGoalReport } from '../api'
import { moneyToNumber, getMonthName } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import { useCurrency } from '../store/CurrencyContext'
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  LineChart,
  Line,
  CartesianGrid,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
} from 'recharts'
import { FormField, Input, ProgressBar, Select } from '../components/ui'

// Violet-first palette that fits the design system
const COLORS = [
  '#8b5cf6', // violet
  '#10b981', // emerald
  '#f59e0b', // amber
  '#3b82f6', // blue
  '#ef4444', // red
  '#ec4899', // pink
  '#06b6d4', // cyan
  '#84cc16', // lime
]

interface ChartTooltipProps {
  active?: boolean
  payload?: Array<{ name: string; value: number; color?: string; fill?: string }>
  label?: string
  formatter?: (value: number) => string
}

function ChartTooltip({ active, payload, label, formatter }: ChartTooltipProps) {
  if (!active || !payload?.length) return null
  return (
    <div className="bg-zinc-900 dark:bg-zinc-800 border border-zinc-700 rounded-xl px-3 py-2.5 shadow-xl text-xs">
      {label && <p className="text-zinc-400 mb-1.5 font-medium">{label}</p>}
      {payload.map((p, i) => (
        <div key={i} className="flex items-center gap-2">
          <span
            className="h-2 w-2 rounded-full flex-shrink-0"
            style={{ backgroundColor: p.color || p.fill }}
          />
          <span className="text-zinc-300">{p.name}:</span>
          <span className="text-white font-semibold">
            {formatter ? formatter(p.value) : p.value}
          </span>
        </div>
      ))}
    </div>
  )
}

export default function Reports() {
  const { user } = useAuth()
  const { formatConverted } = useCurrency()
  const currentDate = new Date()
  const [reportPeriod, setReportPeriod] = useState<'monthly' | 'yearly'>('monthly')
  const [selectedMonth, setSelectedMonth] = useState(currentDate.getMonth() + 1)
  const [selectedYear, setSelectedYear] = useState(currentDate.getFullYear())

  const { data: monthlyReport, isLoading: isLoadingMonthly } = useQuery({
    queryKey: ['monthlyReport', selectedYear, selectedMonth],
    queryFn: () => reportApi.getMonthlyReport(selectedYear, selectedMonth),
  })

  const selectedMonthKey = `${selectedYear}-${String(selectedMonth).padStart(2, '0')}`

  const { data: monthlyNetWorthTrend, isLoading: isLoadingMonthlyNetWorthTrend } = useQuery({
    queryKey: ['netWorthTrend', 'daily', selectedMonthKey],
    queryFn: () => reportApi.getNetWorthTrend({ interval: 'daily', month: selectedMonthKey }),
    enabled: reportPeriod === 'monthly',
  })

  const { data: yearlyNetWorthTrend, isLoading: isLoadingYearlyNetWorthTrend } = useQuery({
    queryKey: ['netWorthTrend', 'monthly', selectedYear],
    queryFn: () => reportApi.getNetWorthTrend({ interval: 'monthly', year: selectedYear, months: 12 }),
    enabled: reportPeriod === 'yearly',
  })

  const yearlyMonthlyQueries = useQueries({
    queries: Array.from({ length: 12 }, (_, i) => {
      const month = i + 1
      return {
        queryKey: ['monthlyReport', selectedYear, month],
        queryFn: () => reportApi.getMonthlyReport(selectedYear, month),
      }
    }),
  })

  const { data: netWorthReport, isLoading: isLoadingNetWorth } = useQuery({
    queryKey: ['netWorthReport'],
    queryFn: () => reportApi.getNetWorthReport(),
  })

  const { data: budgetReport, isLoading: isLoadingBudget } = useQuery<BudgetTrackingReport>({
    queryKey: ['budgetTrackingReport', reportPeriod, selectedYear, selectedMonth],
    queryFn: () =>
      reportApi.getBudgetTrackingReport({
        periodType: reportPeriod === 'yearly' ? 'PERIOD_TYPE_YEARLY' : 'PERIOD_TYPE_MONTHLY',
        year: selectedYear,
        month: selectedMonth,
      }),
  })

  const { data: goalsReport, isLoading: isLoadingGoals } = useQuery<SavingGoalReport[]>({
    queryKey: ['goalsReport'],
    queryFn: () => reportApi.getGoalsReport(),
  })

  const isLoadingYearly = yearlyMonthlyQueries.some((q) => q.isLoading)
  const isLoadingPeriod = reportPeriod === 'yearly' ? isLoadingYearly : isLoadingMonthly
  const isLoadingTrend = reportPeriod === 'yearly' ? isLoadingYearlyNetWorthTrend : isLoadingMonthlyNetWorthTrend
  const isLoading = isLoadingPeriod || isLoadingNetWorth || isLoadingBudget || isLoadingGoals || isLoadingTrend

  const currency = user?.baseCurrency || 'SGD'
  const moneyFmt = (v: number) => formatConverted({ amount: v.toString(), currency })

  const spendingData = monthlyReport?.spendingByCategory?.map((cat, index) => ({
    name: cat.categoryName,
    value: moneyToNumber(cat.amount),
    color: COLORS[index % COLORS.length],
    count: cat.transactionCount,
  })) || []

  const yearlySummary = useMemo(() => {
    const categoryMap = new Map<string, { name: string; value: number; count: number }>()
    const monthlyTrend: { month: string; income: number; expenses: number }[] = []

    let totalIncome = 0
    let totalExpenses = 0

    yearlyMonthlyQueries.forEach((query, index) => {
      const report = query.data
      const income = moneyToNumber(report?.totalIncome || { amount: '0', currency })
      const expenses = moneyToNumber(report?.totalExpenses || { amount: '0', currency })

      totalIncome += income
      totalExpenses += expenses

      monthlyTrend.push({
        month: getMonthName(index + 1).slice(0, 3),
        income,
        expenses,
      })

      report?.spendingByCategory?.forEach((cat) => {
        const existing = categoryMap.get(cat.categoryId)
        const value = moneyToNumber(cat.amount)
        if (existing) {
          existing.value += value
          existing.count += cat.transactionCount
          return
        }

        categoryMap.set(cat.categoryId, {
          name: cat.categoryName,
          value,
          count: cat.transactionCount,
        })
      })
    })

    const spendingByCategory = Array.from(categoryMap.values())
      .sort((a, b) => b.value - a.value)
      .map((item, index) => ({
        ...item,
        color: COLORS[index % COLORS.length],
      }))

    const netSavings = totalIncome - totalExpenses
    const savingsRate = totalIncome > 0 ? (netSavings / totalIncome) * 100 : 0

    return {
      totalIncome,
      totalExpenses,
      netSavings,
      savingsRate,
      spendingByCategory,
      monthlyTrend,
    }
  }, [currency, yearlyMonthlyQueries])

  const activeSpendingData = reportPeriod === 'yearly' ? yearlySummary.spendingByCategory : spendingData

  const monthlyTrendData = (monthlyNetWorthTrend?.trend || []).map((point) => ({
    label: point.month.slice(-2),
    netWorth: moneyToNumber(point.netWorth),
    assets: moneyToNumber(point.assets),
    liabilities: moneyToNumber(point.liabilities),
  }))

  const yearlyTrendData = (yearlyNetWorthTrend?.trend || []).map((point) => {
    const monthNumber = Number(point.month.split('-')[1] || '1')
    return {
      label: getMonthName(monthNumber).slice(0, 3),
      netWorth: moneyToNumber(point.netWorth),
      assets: moneyToNumber(point.assets),
      liabilities: moneyToNumber(point.liabilities),
    }
  })

  const activeNetWorthTrendData = reportPeriod === 'yearly' ? yearlyTrendData : monthlyTrendData

  const budgetRows = budgetReport?.categoryDetails?.slice(0, 8) || []
  const goalRows = goalsReport?.slice(0, 8) || []

  const assetBreakdown = netWorthReport?.assetBreakdown?.map((item, index) => ({
    name: item.category.replace('ASSET_CATEGORY_', ''),
    value: moneyToNumber(item.total),
    color: COLORS[index % COLORS.length],
  })) || []

  const years = Array.from({ length: 5 }, (_, i) => currentDate.getFullYear() - i)
  const months = Array.from({ length: 12 }, (_, i) => ({ value: i + 1, label: getMonthName(i + 1) }))

  const cardClass = 'bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm'
  const headingClass = 'text-sm font-semibold text-zinc-900 dark:text-zinc-50 mb-4'

  const axisStyle = { fill: '#71717a', fontSize: 11 }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">Reports</h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">Financial insights and analytics</p>
      </div>

      {/* Period Selector */}
      <div className={`${cardClass} flex flex-wrap gap-4`}>
        <FormField label="Report Type" className="min-w-[150px]">
          <Select
            value={reportPeriod}
            onChange={(e) => setReportPeriod(e.target.value as 'monthly' | 'yearly')}
          >
            <option value="monthly">Monthly</option>
            <option value="yearly">Yearly</option>
          </Select>
        </FormField>
        <FormField label="Month" className="min-w-[140px]">
          {reportPeriod === 'monthly' ? (
            <Select
              value={selectedMonth}
              onChange={(e) => setSelectedMonth(parseInt(e.target.value))}
            >
              {months.map((m) => (
                <option key={m.value} value={m.value}>{m.label}</option>
              ))}
            </Select>
          ) : (
            <Input value="All Months" disabled />
          )}
        </FormField>
        <FormField label="Year" className="min-w-[110px]">
          <Select
            value={selectedYear}
            onChange={(e) => setSelectedYear(parseInt(e.target.value))}
          >
            {years.map((y) => (
              <option key={y} value={y}>{y}</option>
            ))}
          </Select>
        </FormField>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <svg className="animate-spin h-8 w-8 text-violet-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
        </div>
      ) : (
        <>
          {/* Monthly Summary */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {[
              {
                label: reportPeriod === 'yearly' ? 'Yearly Income' : 'Income',
                value:
                  reportPeriod === 'yearly'
                    ? moneyFmt(yearlySummary.totalIncome)
                    : formatConverted(monthlyReport?.totalIncome),
                color: 'text-emerald-600 dark:text-emerald-400',
              },
              {
                label: reportPeriod === 'yearly' ? 'Yearly Expenses' : 'Expenses',
                value:
                  reportPeriod === 'yearly'
                    ? moneyFmt(yearlySummary.totalExpenses)
                    : formatConverted(monthlyReport?.totalExpenses),
                color: 'text-red-500 dark:text-red-400',
              },
              {
                label: reportPeriod === 'yearly' ? 'Yearly Net Savings' : 'Net Savings',
                value:
                  reportPeriod === 'yearly'
                    ? moneyFmt(yearlySummary.netSavings)
                    : formatConverted(monthlyReport?.netSavings),
                color:
                  (reportPeriod === 'yearly'
                    ? yearlySummary.netSavings
                    : moneyToNumber(monthlyReport?.netSavings || { amount: '0', currency })) >= 0
                    ? 'text-violet-600 dark:text-violet-400'
                    : 'text-red-500 dark:text-red-400',
              },
              {
                label: 'Savings Rate',
                value: `${(
                  reportPeriod === 'yearly' ? yearlySummary.savingsRate : monthlyReport?.savingsRate || 0
                ).toFixed(1)}%`,
                color:
                  (reportPeriod === 'yearly' ? yearlySummary.savingsRate : monthlyReport?.savingsRate || 0) >= 0
                    ? 'text-emerald-600 dark:text-emerald-400'
                    : 'text-red-500 dark:text-red-400',
              },
            ].map((item) => (
              <div key={item.label} className={cardClass}>
                <p className="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400 mb-1">
                  {item.label}
                </p>
                <p className={`text-xl font-bold tracking-tight tabular-nums ${item.color}`}>
                  {item.value}
                </p>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            <div className={`${cardClass} lg:col-span-2`}>
              <h2 className={headingClass}>
                {reportPeriod === 'yearly'
                  ? `Net Worth Trend (${selectedYear})`
                  : `Net Worth Trend (${getMonthName(selectedMonth)} ${selectedYear})`}
              </h2>
              {activeNetWorthTrendData.length > 0 ? (
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={activeNetWorthTrendData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="rgba(113,113,122,0.15)" />
                      <XAxis dataKey="label" tick={axisStyle} axisLine={false} tickLine={false} />
                      <YAxis tick={axisStyle} axisLine={false} tickLine={false} />
                      <Tooltip content={<ChartTooltip formatter={moneyFmt} />} />
                      <Legend wrapperStyle={{ fontSize: 11, color: '#71717a' }} />
                      <Line type="monotone" dataKey="netWorth" name="Net Worth" stroke="#8b5cf6" strokeWidth={2.5} dot={false} />
                      <Line type="monotone" dataKey="assets" name="Assets" stroke="#10b981" strokeWidth={1.8} dot={false} />
                      <Line type="monotone" dataKey="liabilities" name="Liabilities" stroke="#ef4444" strokeWidth={1.8} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <div className="h-56 flex items-center justify-center text-sm text-zinc-400 dark:text-zinc-500">
                  No net worth trend data for this report period
                </div>
              )}
            </div>

            {/* Spending by Category */}
            <div className={cardClass}>
              <h2 className={headingClass}>
                {reportPeriod === 'yearly' ? 'Yearly Spending by Category' : 'Spending by Category'}
              </h2>
              {activeSpendingData.length > 0 ? (
                <>
                  <div className="h-56">
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={activeSpendingData}
                          cx="50%"
                          cy="50%"
                          innerRadius={52}
                          outerRadius={80}
                          paddingAngle={2}
                          dataKey="value"
                        >
                          {activeSpendingData.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip
                          content={<ChartTooltip formatter={moneyFmt} />}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>
                  <div className="mt-3 space-y-1.5">
                    {activeSpendingData.map((item, index) => (
                      <div key={index} className="flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2">
                          <span className="h-2 w-2 rounded-full flex-shrink-0" style={{ backgroundColor: item.color }} />
                          <span className="text-zinc-600 dark:text-zinc-400 truncate max-w-[140px]">{item.name}</span>
                        </div>
                        <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                          {moneyFmt(item.value)}
                        </span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="h-56 flex items-center justify-center text-sm text-zinc-400 dark:text-zinc-500">
                  No spending data for this report period
                </div>
              )}
            </div>

            {reportPeriod === 'yearly' && (
              <div className={cardClass}>
                <h2 className={headingClass}>Income vs Expenses by Month</h2>
                <div className="h-72">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={yearlySummary.monthlyTrend} barCategoryGap="24%">
                      <XAxis dataKey="month" tick={axisStyle} axisLine={false} tickLine={false} />
                      <YAxis tick={axisStyle} axisLine={false} tickLine={false} />
                      <Tooltip content={<ChartTooltip formatter={moneyFmt} />} />
                      <Legend wrapperStyle={{ fontSize: 11, color: '#71717a' }} />
                      <Bar dataKey="income" fill="#10b981" name="Income" radius={[3, 3, 0, 0]} />
                      <Bar dataKey="expenses" fill="#8b5cf6" name="Expenses" radius={[3, 3, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>
            )}

            {/* Budget Status */}
            <div className={cardClass}>
              <h2 className={headingClass}>Budget Status</h2>
              {budgetRows.length > 0 ? (
                <div className="space-y-4">
                  {budgetRows.map((item) => {
                    const pct = item.percentageUsed
                    const variant = pct > 100 ? 'danger' : pct > 80 ? 'warning' : 'success'
                    return (
                      <div key={item.categoryId}>
                        <div className="flex justify-between items-center mb-1.5">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                              {item.categoryName}
                            </span>
                            <Link
                              to={`/transactions?categoryId=${item.categoryId}&startDate=${encodeURIComponent(
                                reportPeriod === 'yearly'
                                  ? new Date(Date.UTC(selectedYear, 0, 1)).toISOString()
                                  : new Date(Date.UTC(selectedYear, selectedMonth - 1, 1)).toISOString()
                              )}&endDate=${encodeURIComponent(
                                reportPeriod === 'yearly'
                                  ? new Date(Date.UTC(selectedYear, 11, 31, 23, 59, 59, 999)).toISOString()
                                  : new Date(Date.UTC(selectedYear, selectedMonth, 0, 23, 59, 59, 999)).toISOString()
                              )}`}
                              className="text-[10px] px-2 py-0.5 rounded-full bg-zinc-100 dark:bg-zinc-800 hover:bg-zinc-200 dark:hover:bg-zinc-700 text-zinc-600 dark:text-zinc-400 font-medium transition-colors"
                            >
                              View Transactions
                            </Link>
                          </div>
                          <span className="text-xs text-zinc-500 dark:text-zinc-400">
                            {formatConverted(item.spent)}{' '}
                            <span className="text-zinc-400 dark:text-zinc-600">/</span>{' '}
                            {formatConverted(item.budgeted)}
                          </span>
                        </div>
                        <ProgressBar value={pct} variant={variant} />
                      </div>
                    )
                  })}
                </div>
              ) : (
                <div className="h-56 flex items-center justify-center text-sm text-zinc-400 dark:text-zinc-500">
                  No budget data available
                </div>
              )}
            </div>

            {/* Saving Goals */}
            <div className={cardClass}>
              <h2 className={headingClass}>Saving Goals</h2>
              {goalRows.length > 0 ? (
                <div className="space-y-4">
                  {goalRows.map((goal) => {
                    const pct = goal.percentageComplete
                    const variant = pct >= 100 ? 'success' : goal.isOnTrack ? 'default' : 'warning'
                    return (
                      <div key={goal.goalId}>
                        <div className="flex justify-between items-center mb-1.5">
                          <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300 truncate pr-2">
                            {goal.goalName}
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
                <div className="h-56 flex items-center justify-center text-sm text-zinc-400 dark:text-zinc-500">
                  No goals set up yet
                </div>
              )}
            </div>

            {/* Asset Allocation */}
            <div className={cardClass}>
              <h2 className={headingClass}>Asset Allocation</h2>
              <div className="grid grid-cols-2 gap-3 mb-4">
                <div className="text-center p-3 bg-zinc-50 dark:bg-zinc-800/60 rounded-xl border border-zinc-100 dark:border-zinc-700">
                  <p className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">Total Assets</p>
                  <p className="text-base font-bold tabular-nums text-emerald-600 dark:text-emerald-400">
                    {formatConverted(netWorthReport?.totalAssets)}
                  </p>
                </div>
                <div className="text-center p-3 bg-zinc-50 dark:bg-zinc-800/60 rounded-xl border border-zinc-100 dark:border-zinc-700">
                  <p className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">Net Worth</p>
                  <p className="text-base font-bold tabular-nums text-violet-600 dark:text-violet-400">
                    {formatConverted(netWorthReport?.netWorth)}
                  </p>
                </div>
              </div>
              {assetBreakdown.length > 0 ? (
                <div className="h-40">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={assetBreakdown}
                        cx="50%"
                        cy="50%"
                        outerRadius={60}
                        dataKey="value"
                        label={({ name }) => name}
                        labelLine={false}
                      >
                        {assetBreakdown.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Pie>
                      <Tooltip content={<ChartTooltip formatter={moneyFmt} />} />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <div className="h-40 flex items-center justify-center text-sm text-zinc-400 dark:text-zinc-500">
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

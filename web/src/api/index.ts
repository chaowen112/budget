import api from './client'
import type {
  User,
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  Category,
  Transaction,
  CreateTransactionRequest,
  Budget,
  BudgetStatus,
  SavingGoal,
  GoalProgress,
  GoalHistory,
  NetWorthGoal,
  NetWorthGoalProgress,
  Asset,
  AssetSnapshot,
  MonthlyReport,
  NetWorthReport,
  NetWorthTrendReport,
  LedgerAccount,
  JournalEntry,
  Transfer,
  CreateTransferRequest,
  AssistantParseResponse,
  Currency,
  ExchangeRate,
  CategoryType,
  PeriodType,
  AssetCategory,
  Money,
  TransactionSourceLink,
} from '../types'

function dateInputToUTCISOString(value: string): string {
  const [yearStr, monthStr, dayStr] = value.split('-')
  const year = Number(yearStr)
  const month = Number(monthStr)
  const day = Number(dayStr)

  if (!Number.isFinite(year) || !Number.isFinite(month) || !Number.isFinite(day)) {
    return new Date(value).toISOString()
  }

  return new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0)).toISOString()
}

// Auth API
export const authApi = {
  login: async (data: LoginRequest): Promise<AuthResponse> => {
    const response = await api.post('/auth/login', data)
    return response.data
  },
  
  register: async (data: RegisterRequest): Promise<AuthResponse> => {
    const response = await api.post('/auth/register', data)
    return response.data
  },
  
  logout: async (refreshToken: string): Promise<void> => {
    await api.post('/auth/logout', { refreshToken })
  },
  
  getProfile: async (): Promise<User> => {
    const response = await api.get('/users/me')
    return response.data.user
  },
  
  updateProfile: async (data: { name?: string; baseCurrency?: string }): Promise<User> => {
    const response = await api.patch('/users/me', data)
    return response.data.user
  },
}

// Category API
export const categoryApi = {
  list: async (type?: CategoryType): Promise<Category[]> => {
    const params = type ? { type } : {}
    const response = await api.get('/categories', { params })
    return response.data.categories || []
  },
  
  create: async (data: { name: string; type: CategoryType; icon?: string; color?: string }): Promise<Category> => {
    const response = await api.post('/categories', data)
    return response.data.category
  },
  
  update: async (id: string, data: { name?: string; icon?: string; color?: string }): Promise<Category> => {
    const response = await api.patch(`/categories/${id}`, data)
    return response.data.category
  },
  
  delete: async (id: string): Promise<void> => {
    await api.delete(`/categories/${id}`)
  },
}

// Transaction API
export const transactionApi = {
  list: async (params?: {
    page?: number
    startDate?: string
    endDate?: string
    categoryId?: string
    keyword?: string
    type?: CategoryType
    currency?: string
    pageSize?: number
  }): Promise<{
    transactions: Transaction[]
    pagination?: { page: number; pageSize: number; totalCount: number; totalPages: number }
  }> => {
    const queryParams: Record<string, string | number> = {}

    if (params?.page) queryParams['pagination.page'] = params.page
    if (params?.pageSize) queryParams['pagination.pageSize'] = params.pageSize
    if (params?.startDate) queryParams['dateRange.startDate'] = params.startDate
    if (params?.endDate) queryParams['dateRange.endDate'] = params.endDate
    if (params?.categoryId) queryParams.categoryId = params.categoryId
    if (params?.type) queryParams.type = params.type
    if (params?.currency) queryParams.currency = params.currency

    const response = await api.get('/transactions', {
      params: queryParams,
      headers: params?.keyword
        ? { 'Grpc-Metadata-search-keyword': params.keyword }
        : undefined,
    })
    return {
      transactions: response.data.transactions || [],
      pagination: response.data.pagination,
    }
  },
  
  get: async (id: string): Promise<Transaction> => {
    const response = await api.get(`/transactions/${id}`)
    return response.data.transaction
  },
  
  create: async (data: CreateTransactionRequest): Promise<Transaction> => {
    const { sourceAssetId, ...payload } = data
    const response = await api.post('/transactions', payload, {
      headers: { 'Grpc-Metadata-source-asset-id': sourceAssetId },
    })
    return response.data.transaction
  },
  
  update: async (id: string, data: Partial<CreateTransactionRequest>): Promise<Transaction> => {
    const { sourceAssetId, ...payload } = data
    const response = await api.patch(`/transactions/${id}`, payload, {
      headers: sourceAssetId ? { 'Grpc-Metadata-source-asset-id': sourceAssetId } : undefined,
    })
    return response.data.transaction
  },
  
  delete: async (id: string): Promise<void> => {
    await api.delete(`/transactions/${id}`)
  },

  listSourceLinks: async (): Promise<TransactionSourceLink[]> => {
    const response = await api.get('/transactions/source-links')
    return response.data.links || []
  },
}

// Budget API
export const budgetApi = {
  list: async (): Promise<Budget[]> => {
    const response = await api.get('/budgets')
    return response.data.budgets || []
  },
  
  get: async (id: string): Promise<Budget> => {
    const response = await api.get(`/budgets/${id}`)
    return response.data.budget
  },
  
  create: async (data: {
    categoryId: string
    amount: Money
    periodType: PeriodType
    startDate: string  // Will be converted to ISO timestamp
  }): Promise<Budget> => {
    const response = await api.post('/budgets', {
      ...data,
      startDate: dateInputToUTCISOString(data.startDate),
    })
    return response.data.budget
  },
  
  update: async (id: string, data: { amount?: Money }): Promise<Budget> => {
    const response = await api.patch(`/budgets/${id}`, data)
    return response.data.budget
  },
  
  delete: async (id: string): Promise<void> => {
    await api.delete(`/budgets/${id}`)
  },
  
  getStatus: async (id: string): Promise<BudgetStatus> => {
    const response = await api.get(`/budgets/${id}/status`)
    return response.data
  },
  
  getAllStatuses: async (): Promise<BudgetStatus[]> => {
    const response = await api.get('/budgets/status')
    return response.data.statuses || []
  },
}

// Goals API
export const goalApi = {
  list: async (includeCompleted?: boolean): Promise<SavingGoal[]> => {
    const params = includeCompleted !== undefined ? { includeCompleted } : {}
    const response = await api.get('/goals', { params })
    return response.data.goals || []
  },
  
  get: async (id: string): Promise<SavingGoal> => {
    const response = await api.get(`/goals/${id}`)
    return response.data.goal
  },
  
  create: async (data: {
    name: string
    targetAmount: Money
    deadline?: string  // Date string, will be converted to ISO
    notes?: string
  }): Promise<SavingGoal> => {
    const payload: Record<string, unknown> = {
      name: data.name,
      targetAmount: data.targetAmount,
      notes: data.notes,
    }
    if (data.deadline) {
      payload.deadline = new Date(data.deadline).toISOString()
    }
    const response = await api.post('/goals', payload)
    return response.data.goal
  },
  
  update: async (id: string, data: {
    name?: string
    targetAmount?: Money
    deadline?: string
    notes?: string
  }): Promise<SavingGoal> => {
    const payload: Record<string, unknown> = { ...data }
    if (data.deadline) {
      payload.deadline = new Date(data.deadline).toISOString()
    }
    const response = await api.patch(`/goals/${id}`, payload)
    return response.data.goal
  },
  
  delete: async (id: string): Promise<void> => {
    await api.delete(`/goals/${id}`)
  },
  
  updateProgress: async (id: string, currentAmount: Money, source = 'manual'): Promise<SavingGoal> => {
    const response = await api.put(
      `/goals/${id}/progress`,
      { currentAmount },
      { headers: { 'x-goal-change-source': source } },
    )
    return response.data.goal
  },
  
  getProgress: async (id: string): Promise<GoalProgress> => {
    const response = await api.get(`/goals/${id}/progress`)
    return response.data
  },
  
  getAllProgress: async (): Promise<GoalProgress[]> => {
    const response = await api.get('/goals/progress')
    return response.data.progress || []
  },

  getHistory: async (id: string): Promise<GoalHistory> => {
    const response = await api.get(`/goals/${id}/history`, { params: { max_points: 365 } })
    return response.data
  },
  
  // Net Worth Goal API
  getNetWorthGoal: async (): Promise<NetWorthGoal | null> => {
    const response = await api.get('/net-worth-goal')
    return response.data.goal || null
  },
  
  setNetWorthGoal: async (data: {
    name: string
    targetAmount: Money
    notes?: string
  }): Promise<NetWorthGoal> => {
    const response = await api.put('/net-worth-goal', data)
    return response.data.goal
  },
  
  deleteNetWorthGoal: async (): Promise<void> => {
    await api.delete('/net-worth-goal')
  },
  
  getNetWorthGoalProgress: async (): Promise<NetWorthGoalProgress | null> => {
    const response = await api.get('/net-worth-goal/progress')
    return response.data.progress || null
  },
}

// Asset API
export const assetApi = {
  list: async (params?: {
    category?: AssetCategory
    includeLiabilities?: boolean
  }): Promise<Asset[]> => {
    const response = await api.get('/assets', { params: { 
      category: params?.category,
      include_liabilities: params?.includeLiabilities 
    }})
    return response.data.assets || []
  },
  
  get: async (id: string): Promise<Asset> => {
    const response = await api.get(`/assets/${id}`)
    return response.data.asset
  },
  
  listAssetTypes: async (category?: AssetCategory): Promise<{ id: string; name: string; category: AssetCategory }[]> => {
    const response = await api.get('/asset-types', { params: category ? { category } : {} })
    return response.data.assetTypes || []
  },
  
  create: async (data: {
    assetTypeId: string
    name: string
    currency: string
    currentValue: string  // Decimal string
    isLiability?: boolean
  }): Promise<Asset> => {
    const response = await api.post('/assets', {
      asset_type_id: data.assetTypeId,
      name: data.name,
      currency: data.currency,
      current_value: data.currentValue,
      is_liability: data.isLiability || false,
    })
    return response.data.asset
  },
  
  update: async (id: string, data: {
    assetTypeId?: string
    name?: string
    currency?: string
    currentValue?: string
    notes?: string
  }): Promise<Asset> => {
    const payload: Record<string, unknown> = {}
    if (data.name) payload.name = data.name
    if (data.currentValue) payload.current_value = data.currentValue
    if (data.notes) payload.notes = data.notes
    const headers: Record<string, string> = {}
    if (data.assetTypeId) headers['Grpc-Metadata-asset-type-id'] = data.assetTypeId
    if (data.currency) headers['Grpc-Metadata-asset-currency'] = data.currency

    const response = await api.patch(`/assets/${id}`, payload, {
      headers: Object.keys(headers).length > 0 ? headers : undefined,
    })
    return response.data.asset
  },
  
  delete: async (id: string): Promise<void> => {
    await api.delete(`/assets/${id}`)
  },

  getHistory: async (assetId: string): Promise<AssetSnapshot[]> => {
    const response = await api.get(`/assets/${assetId}/history`)
    return response.data.snapshots || []
  },

  recordSnapshot: async (assetId: string, value: string, recordedAt?: string): Promise<AssetSnapshot> => {
    const payload: Record<string, unknown> = { value }
    if (recordedAt) {
      payload.recorded_at = recordedAt
    }
    const response = await api.post(`/assets/${assetId}/snapshots`, payload)
    return response.data.snapshot
  },
}

// Report API
export const reportApi = {
  getMonthlyReport: async (year: number, month: number): Promise<MonthlyReport> => {
    const monthStr = `${year}-${month.toString().padStart(2, '0')}`
    const response = await api.get('/reports/monthly', { params: { month: monthStr } })
    return response.data.report
  },
  
  getWeeklyReport: async (year: number, week: number): Promise<MonthlyReport> => {
    const response = await api.get('/reports/weekly', { params: { year, week } })
    return response.data.report
  },
  
  getNetWorthReport: async (): Promise<NetWorthReport> => {
    const response = await api.get('/reports/net-worth')
    return response.data.report
  },

  getNetWorthTrend: async (params?: {
    months?: number
    interval?: 'monthly' | 'daily'
    year?: number
    month?: string
  }): Promise<NetWorthTrendReport> => {
    const queryParams = {
      months: params?.months ?? 12,
    }

    const metadataHeaders: Record<string, string> = {}
    if (params?.interval) metadataHeaders['Grpc-Metadata-trend-interval'] = params.interval
    if (typeof params?.year === 'number') metadataHeaders['Grpc-Metadata-trend-year'] = String(params.year)
    if (params?.month) metadataHeaders['Grpc-Metadata-trend-month'] = params.month

    const response = await api.get('/reports/net-worth-trend', {
      params: queryParams,
      headers: Object.keys(metadataHeaders).length > 0 ? metadataHeaders : undefined,
    })
    return response.data
  },
  
  getBudgetTrackingReport: async (params?: {
    periodType?: PeriodType
    year?: number
    month?: number
  }): Promise<BudgetTrackingReport> => {
    const metadataHeaders: Record<string, string> = {}
    if (typeof params?.year === 'number') metadataHeaders['Grpc-Metadata-report-year'] = String(params.year)
    if (typeof params?.month === 'number') metadataHeaders['Grpc-Metadata-report-month'] = String(params.month)

    const response = await api.get('/reports/budget-tracking', {
      params: params?.periodType ? { periodType: params.periodType } : undefined,
      headers: Object.keys(metadataHeaders).length > 0 ? metadataHeaders : undefined,
    })
    return response.data.report
  },
  
  getGoalsReport: async (): Promise<SavingGoalReport[]> => {
    const response = await api.get('/reports/goals')
    return response.data.goals || []
  },
}

export const accountingApi = {
  listAccounts: async (): Promise<LedgerAccount[]> => {
    const response = await api.get('/accounting/accounts')
    return response.data.accounts || []
  },

  listJournal: async (limit = 50): Promise<JournalEntry[]> => {
    const response = await api.get('/accounting/journal', { params: { limit } })
    return response.data.entries || []
  },
}

export const transferApi = {
  list: async (): Promise<Transfer[]> => {
    const response = await api.get('/transfers')
    return response.data.transfers || []
  },

  create: async (data: CreateTransferRequest): Promise<Transfer> => {
    const response = await api.post('/transfers', {
      ...data,
      transferDate: new Date(data.transferDate).toISOString(),
    })
    return response.data.transfer
  },

  update: async (id: string, data: CreateTransferRequest): Promise<Transfer> => {
    const response = await api.patch(`/transfers/${id}`, {
      ...data,
      transferDate: new Date(data.transferDate).toISOString(),
    })
    return response.data.transfer
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/transfers/${id}`)
  },
}

export const assistantApi = {
  parseTransactionInput: async (data: { message: string; imageDataUrl?: string }): Promise<AssistantParseResponse> => {
    const response = await api.post('/transactions/assistant/parse', data)
    return response.data
  },
}

// Types for report API
export interface BudgetTrackingReport {
  periodType: string
  periodStart: string
  periodEnd: string
  daysElapsed: number
  daysRemaining: number
  periodProgressPercentage: number
  totalBudgeted: Money
  totalSpent: Money
  expectedSpent: Money
  budgetUtilization: number
  isOnTrack: boolean
  statusMessage: string
  projectedEndOfPeriodSpending: Money
  categoryDetails: BudgetSummary[]
}

export interface BudgetSummary {
  categoryId: string
  categoryName: string
  budgeted: Money
  spent: Money
  remaining: Money
  percentageUsed: number
  isOverBudget: boolean
}

export interface SavingGoalReport {
  goalId: string
  goalName: string
  targetAmount: Money
  currentAmount: Money
  percentageComplete: number
  deadline?: string
  daysRemaining: number
  requiredMonthlySaving: Money
  currentMonthlySaving: Money
  isOnTrack: boolean
}

// Currency API (public endpoints)
export const currencyApi = {
  list: async (): Promise<Currency[]> => {
    const response = await api.get('/currencies')
    return response.data.currencies || []
  },
  
  getExchangeRate: async (from: string, to: string): Promise<ExchangeRate> => {
    const response = await api.get('/currencies/rate', { params: { fromCurrency: from, toCurrency: to } })
    return response.data.rate
  },
  
  convert: async (from: string, to: string, amount: Money): Promise<Money> => {
    const response = await api.post('/currencies/convert', {
      fromCurrency: from,
      toCurrency: to,
      amount,
    })
    return response.data.convertedAmount
  },
}

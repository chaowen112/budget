// User types
export interface User {
  id: string
  email: string
  name: string
  baseCurrency: string
  createdAt: string
}

// Auth types
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
  baseCurrency?: string
}

export interface AuthResponse {
  user: User
  accessToken: string
  refreshToken: string
}

// Money type (matches proto Money message)
export interface Money {
  amount: string    // Decimal string (e.g., "100.50")
  currency: string  // ISO 4217 currency code (e.g., "SGD")
}

// Category types (matches proto TransactionType enum)
export type CategoryType = 'TRANSACTION_TYPE_EXPENSE' | 'TRANSACTION_TYPE_INCOME'

export interface Category {
  id: string
  userId: string
  name: string
  type: CategoryType
  icon?: string
  color?: string
  isSystem: boolean
  createdAt: string
}

// Transaction types
export interface Transaction {
  id: string
  userId: string
  categoryId: string
  categoryName?: string
  amount: Money
  type?: string
  transactionDate: string  // ISO timestamp from API
  description?: string
  tags?: string[]
  createdAt: string
}

export interface CreateTransactionRequest {
  categoryId: string
  amount: Money
  transactionDate: string  // ISO timestamp
  description?: string
  tags?: string[]
}

// Budget types
export type PeriodType = 'PERIOD_TYPE_WEEKLY' | 'PERIOD_TYPE_MONTHLY' | 'PERIOD_TYPE_YEARLY'

export interface Budget {
  id: string
  categoryId: string
  categoryName?: string
  amount: Money
  periodType: PeriodType
  startDate: string
  createdAt: string
}

export interface BudgetStatus {
  budget: Budget
  spent: Money
  remaining: Money
  percentageUsed: number
  isOverBudget: boolean
}

// Saving Goal types
export interface SavingGoal {
  id: string
  name: string
  targetAmount: Money
  currentAmount: Money
  deadline?: string  // ISO timestamp
  linkedAssetIds?: string[]
  notes?: string
  createdAt: string
}

export interface GoalProgress {
  goal: SavingGoal
  percentageComplete: number
  amountRemaining: Money
  daysRemaining: number
  requiredMonthlySaving: Money
  isOnTrack: boolean
  statusMessage?: string
}

export interface GoalHistoryPoint {
  id: string
  goalId: string
  amount: string
  recordedAt: string
}

export interface GoalContribution {
  id: string
  goalId: string
  amountDelta: string
  balanceAfter: string
  source: string
  recordedAt: string
}

export interface GoalHistory {
  goal: {
    id: string
    name: string
    targetAmount: string
    currentAmount: string
    currency: string
    createdAt: string
    deadline?: string
  }
  history: GoalHistoryPoint[]
  contributions?: GoalContribution[]
}

// Net Worth Goal types
export interface NetWorthGoal {
  id: string
  name: string
  targetAmount: Money
  notes?: string
  createdAt: string
  updatedAt: string
}

export interface NetWorthGoalProgress {
  goal: NetWorthGoal
  currentNetWorth: Money
  percentageComplete: number
  amountRemaining: Money
  estimatedMonthsToGoal: number  // -1 if cannot calculate
}

// Asset types
export type AssetCategory = 
  | 'ASSET_CATEGORY_CASH'
  | 'ASSET_CATEGORY_BANK'
  | 'ASSET_CATEGORY_INVESTMENT'
  | 'ASSET_CATEGORY_PROPERTY'
  | 'ASSET_CATEGORY_VEHICLE'
  | 'ASSET_CATEGORY_CRYPTO'
  | 'ASSET_CATEGORY_OTHER'

export interface AssetType {
  id: string
  name: string
  category: AssetCategory
}

export interface Asset {
  id: string
  assetTypeId: string
  assetTypeName: string
  category: AssetCategory
  name: string
  currency: string
  currentValue: string  // Decimal string
  isLiability: boolean
  customFields?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface AssetSnapshot {
  id: string
  assetId: string
  value: string
  recordedAt: string
}

// Report types
export interface CategorySpending {
  categoryId: string
  categoryName: string
  amount: Money
  percentage: number
  transactionCount: number
}

export interface MonthlyReport {
  month: string
  year: number
  totalIncome: Money
  totalExpenses: Money
  netSavings: Money
  savingsRate: number
  spendingByCategory: CategorySpending[]
}

export interface NetWorthReport {
  totalAssets: Money
  totalLiabilities: Money
  netWorth: Money
  assetBreakdown: {
    category: AssetCategory
    total: Money
    percentage: number
  }[]
}

export interface NetWorthTrendPoint {
  month: string
  netWorth: Money
  assets: Money
  liabilities: Money
}

export interface NetWorthTrendReport {
  trend: NetWorthTrendPoint[]
  totalChange: Money
  totalChangePercentage: number
}

// Currency types
export interface Currency {
  code: string
  name: string
  symbol: string
}

export interface ExchangeRate {
  fromCurrency: string
  toCurrency: string
  rate: number
  updatedAt: string
}

// Pagination
export interface PaginatedResponse<T> {
  items: T[]
  totalCount: number
  pageSize: number
  pageToken?: string
  nextPageToken?: string
}

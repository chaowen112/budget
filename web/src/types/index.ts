// User types
export interface User {
  id: string
  email: string
  name: string
  baseCurrency: string
  createdAt: string
}

export interface ApiKey {
  id: string
  name: string
  keyValue?: string
  createdAt: string
  lastUsedAt?: string
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
  sourceAssetId?: string
  sourceAssetName?: string
  amount: Money
  type?: string
  transactionDate: string  // ISO timestamp from API
  description?: string
  tags?: string[]
  budgetAmount?: Money  // Portion that counts toward budget, in original currency (omitted = full amount)
  createdAt: string
}

export interface CreateTransactionRequest {
  categoryId: string
  sourceAssetId: string
  amount: Money
  transactionDate: string  // ISO timestamp
  description?: string
  tags?: string[]
  budgetAmount?: string  // Decimal string; omit or "" for full amount
}

export interface TransactionSourceLink {
  transactionId: string
  assetId: string
  assetName: string
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
  cost?: string
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

export interface CashflowTrendPoint {
  month: string
  income: Money
  expenses: Money
  net: Money
}

export interface CashflowTrendReport {
  trend: CashflowTrendPoint[]
  averageIncome: Money
  averageExpenses: Money
  averageNet: Money
}

export interface LedgerAccount {
  id: string
  name: string
  accountType: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  assetTypeName?: string
  currency: string
  openingBalance: string
  balance: string
  isSystem: boolean
  assetId?: string
  categoryId?: string
  createdAt: string
  updatedAt: string
}

export interface JournalLine {
  id: string
  accountId: string
  accountName: string
  accountType: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  debit: string
  credit: string
  baseDebit: string
  baseCredit: string
  description: string
}

export interface JournalEntry {
  id: string
  entryDate: string
  description: string
  source: string
  referenceType: string
  referenceId?: string
  baseCurrency: string
  createdAt: string
  lines: JournalLine[]
}

export interface Transfer {
  id: string
  fromAssetId: string
  toAssetId: string
  fromAmount: string
  toAmount: string
  fromCurrency: string
  toCurrency: string
  exchangeRate: string
  transferDate: string
  description?: string
  createdAt: string
  updatedAt: string
  fromAssetName?: string
  toAssetName?: string
}

export interface CreateTransferRequest {
  fromAssetId: string
  toAssetId: string
  fromAmount: string
  toAmount?: string
  fromCurrency: string
  toCurrency?: string
  exchangeRate?: string
  transferDate: string
  description?: string
}

export interface AssistantSuggestion {
  entryType: 'transaction' | 'transfer'
  description: string
  transactionDate: string
  amount?: string
  currency?: string
  categoryType?: CategoryType
  categoryName?: string
  sourceAsset?: string
  fromAsset?: string
  toAsset?: string
  fromAmount?: string
  toAmount?: string
  fromCurrency?: string
  toCurrency?: string
  confidence: number
  missingFields: string[]
}

export interface AssistantParseResponse {
  suggestion: AssistantSuggestion
  rawText: string
  provider: string
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

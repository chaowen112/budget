import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { transactionApi, categoryApi, assetApi } from '../api'
import { formatDate, numberToMoney, moneyToNumber } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import { useCurrency, DISPLAY_CURRENCIES } from '../store/CurrencyContext'
import type { Transaction, Category, CategoryType } from '../types'
import { Plus, Pencil, Trash2, Search, ArrowDownLeft, ArrowUpRight } from 'lucide-react'
import { Button, Modal, FormField, Input, Select } from '../components/ui'

const CREATE_CATEGORY_OPTION = '__create_new_category__'

export default function Transactions() {
  const { user } = useAuth()
  const { formatConverted } = useCurrency()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [filterCategory, setFilterCategory] = useState<string>('')
  const [transactionCategoryId, setTransactionCategoryId] = useState('')
  const [transactionSourceAssetId, setTransactionSourceAssetId] = useState('')
  const [showQuickCategoryForm, setShowQuickCategoryForm] = useState(false)
  const [quickCategoryName, setQuickCategoryName] = useState('')
  const [quickCategoryType, setQuickCategoryType] = useState<CategoryType>('TRANSACTION_TYPE_EXPENSE')

  const { data: transactionsData, isLoading } = useQuery({
    queryKey: ['transactions'],
    queryFn: () => transactionApi.list({ pageSize: 100 }),
  })

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoryApi.list(),
  })

  const { data: assets } = useQuery({
    queryKey: ['assets', 'transaction-source'],
    queryFn: () => assetApi.list({ includeLiabilities: true }),
  })

  const createMutation = useMutation({
    mutationFn: transactionApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries()
      queryClient.refetchQueries({ type: 'active' })
      setIsModalOpen(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof transactionApi.update>[1] }) =>
      transactionApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries()
      queryClient.refetchQueries({ type: 'active' })
      setIsModalOpen(false)
      setEditingTransaction(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: transactionApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries()
      queryClient.refetchQueries({ type: 'active' })
    },
  })

  const createCategoryMutation = useMutation({
    mutationFn: categoryApi.create,
    onSuccess: (createdCategory) => {
      queryClient.setQueryData<Category[]>(['categories'], (existing = []) => {
        if (existing.some((c) => c.id === createdCategory.id)) {
          return existing
        }
        return [...existing, createdCategory].sort((a, b) => a.name.localeCompare(b.name))
      })
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      setTransactionCategoryId(createdCategory.id)
      setQuickCategoryName('')
      setShowQuickCategoryForm(false)
    },
  })

  const transactions = transactionsData?.transactions || []
  const hasCategories = (categories?.length || 0) > 0
  const hasAssets = (assets?.length || 0) > 0
  const filteredTransactions = transactions.filter((t) => {
    const matchesSearch =
      !searchTerm ||
      t.description?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      t.categoryName?.toLowerCase().includes(searchTerm.toLowerCase())
    const matchesCategory = !filterCategory || t.categoryId === filterCategory
    return matchesSearch && matchesCategory
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const amount = parseFloat(formData.get('amount') as string)
    const currency = (formData.get('currency') as string) || user?.baseCurrency || 'SGD'
    const dateStr = formData.get('date') as string
    const transactionDate = new Date(dateStr).toISOString()
    const data = {
      categoryId: (formData.get('categoryId') as string) || transactionCategoryId,
      amount: numberToMoney(amount, currency),
      description: formData.get('description') as string,
      transactionDate,
    } as Parameters<typeof transactionApi.create>[0]

    const selectedSourceAssetId = (formData.get('sourceAssetId') as string) || transactionSourceAssetId
    if (!editingTransaction) {
      data.sourceAssetId = selectedSourceAssetId
    } else if (selectedSourceAssetId) {
      data.sourceAssetId = selectedSourceAssetId
    }

    if (editingTransaction) {
      updateMutation.mutate({ id: editingTransaction.id, data })
    } else {
      createMutation.mutate(data)
    }
  }

  const getCategoryType = (categoryId: string): CategoryType | undefined => {
    return categories?.find((c) => c.id === categoryId)?.type
  }

  const closeModal = () => {
    setIsModalOpen(false)
    setEditingTransaction(null)
    setTransactionCategoryId('')
    setTransactionSourceAssetId('')
    setShowQuickCategoryForm(false)
    setQuickCategoryName('')
    setQuickCategoryType('TRANSACTION_TYPE_EXPENSE')
  }

  const handleTransactionCategoryChange = (value: string) => {
    if (value === CREATE_CATEGORY_OPTION) {
      setShowQuickCategoryForm(true)
      setTransactionCategoryId('')
      setQuickCategoryType('TRANSACTION_TYPE_EXPENSE')
      return
    }

    setShowQuickCategoryForm(false)
    setTransactionCategoryId(value)
  }

  const handleQuickCategoryCreate = () => {
    const trimmed = quickCategoryName.trim()
    if (!trimmed) return
    createCategoryMutation.mutate({
      name: trimmed,
      type: quickCategoryType,
    })
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            Transactions
          </h1>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">
            Track your income and expenses
          </p>
        </div>
        <Button
          icon={<Plus className="h-4 w-4" />}
          disabled={!hasAssets}
          onClick={() => {
            setEditingTransaction(null)
            setTransactionCategoryId('')
            setTransactionSourceAssetId('')
            setIsModalOpen(true)
          }}
        >
          Add Transaction
        </Button>
      </div>

      {!hasCategories && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300">
          You have no categories yet. Create one in this form or in{' '}
          <Link to="/settings" className="font-medium underline underline-offset-2">
            Settings
          </Link>{' '}
          before adding transactions.
        </div>
      )}

      {!hasAssets && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300">
          You have no assets yet. Add an asset in{' '}
          <Link to="/assets" className="font-medium underline underline-offset-2">
            Assets
          </Link>{' '}
          before adding transactions.
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-zinc-400 dark:text-zinc-500 pointer-events-none" />
          <Input
            type="text"
            placeholder="Search transactions..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select
          value={filterCategory}
          onChange={(e) => setFilterCategory(e.target.value)}
          className="sm:w-48"
        >
          <option value="">All Categories</option>
          {categories?.map((cat) => (
            <option key={cat.id} value={cat.id}>{cat.name}</option>
          ))}
        </Select>
      </div>

      {/* Transactions List */}
      <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 overflow-hidden shadow-sm">
        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <svg className="animate-spin h-6 w-6 text-violet-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
        ) : filteredTransactions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-zinc-400 dark:text-zinc-500">
            <Search className="h-8 w-8 mb-3 opacity-40" />
            <p className="text-sm">No transactions found</p>
          </div>
        ) : (
          <div className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {filteredTransactions.map((transaction) => {
              const isExpense = getCategoryType(transaction.categoryId) === 'TRANSACTION_TYPE_EXPENSE'
              return (
                <div
                  key={transaction.id}
                  className="flex items-center gap-4 px-5 py-3.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors duration-150 group"
                >
                  {/* Icon */}
                  <div
                    className={`h-9 w-9 rounded-xl flex items-center justify-center flex-shrink-0 ${
                      isExpense
                        ? 'bg-red-50 dark:bg-red-500/10'
                        : 'bg-emerald-50 dark:bg-emerald-500/10'
                    }`}
                  >
                    {isExpense ? (
                      <ArrowDownLeft className="h-4 w-4 text-red-500 dark:text-red-400" />
                    ) : (
                      <ArrowUpRight className="h-4 w-4 text-emerald-500 dark:text-emerald-400" />
                    )}
                  </div>

                  {/* Details */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">
                      {transaction.description || transaction.categoryName}
                    </p>
                    <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                      {transaction.categoryName} · {formatDate(transaction.transactionDate)}
                      {transaction.amount.currency !== 'SGD' && (
                        <span className="ml-1.5 text-zinc-400 dark:text-zinc-500">
                          ({transaction.amount.currency})
                        </span>
                      )}
                    </p>
                  </div>

                  {/* Amount */}
                  <span
                    className={`text-sm font-semibold tabular-nums flex-shrink-0 ${
                      isExpense
                        ? 'text-red-500 dark:text-red-400'
                        : 'text-emerald-600 dark:text-emerald-400'
                    }`}
                  >
                    {isExpense ? '-' : '+'}{formatConverted(transaction.amount)}
                  </span>

                  {/* Actions */}
                  <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity duration-150">
                    <button
                      onClick={() => {
                        setEditingTransaction(transaction)
                        setTransactionCategoryId(transaction.categoryId)
                        setIsModalOpen(true)
                      }}
                      className="h-7 w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors duration-150"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Delete this transaction?')) {
                          deleteMutation.mutate(transaction.id)
                        }
                      }}
                      className="h-7 w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors duration-150"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Modal */}
      <Modal
        open={isModalOpen}
        onClose={closeModal}
        title={editingTransaction ? 'Edit Transaction' : 'Add Transaction'}
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={closeModal}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="transaction-form"
              loading={createMutation.isPending || updateMutation.isPending}
            >
              {editingTransaction ? 'Update' : 'Add'}
            </Button>
          </div>
        }
      >
        <form id="transaction-form" onSubmit={handleSubmit} className="space-y-4">
          <FormField label="Category">
            <Select
              name="categoryId"
              required
              value={transactionCategoryId}
              onChange={(e) => handleTransactionCategoryChange(e.target.value)}
            >
              <option value="">Select category</option>
              <option value={CREATE_CATEGORY_OPTION}>+ Create new category...</option>
              <optgroup label="Expenses">
                {categories
                  ?.filter((c) => c.type === 'TRANSACTION_TYPE_EXPENSE')
                  .map((cat) => (
                    <option key={cat.id} value={cat.id}>{cat.name}</option>
                  ))}
              </optgroup>
              <optgroup label="Income">
                {categories
                  ?.filter((c) => c.type === 'TRANSACTION_TYPE_INCOME')
                  .map((cat) => (
                    <option key={cat.id} value={cat.id}>{cat.name}</option>
                  ))}
              </optgroup>
            </Select>
          </FormField>

          <FormField label="Source Asset">
            <Select
              name="sourceAssetId"
              required={!editingTransaction}
              value={transactionSourceAssetId}
              onChange={(e) => setTransactionSourceAssetId(e.target.value)}
            >
              <option value="">{editingTransaction ? 'Keep existing asset link' : 'Select asset'}</option>
              {assets?.map((asset) => (
                <option key={asset.id} value={asset.id}>
                  {asset.name}
                </option>
              ))}
            </Select>
          </FormField>

          {showQuickCategoryForm && (
            <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-2">
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                <Input
                  type="text"
                  value={quickCategoryName}
                  onChange={(e) => setQuickCategoryName(e.target.value)}
                  placeholder="Category name"
                  className="sm:col-span-2"
                />
                <Select
                  value={quickCategoryType}
                  onChange={(e) => setQuickCategoryType(e.target.value as CategoryType)}
                >
                  <option value="TRANSACTION_TYPE_EXPENSE">Expense</option>
                  <option value="TRANSACTION_TYPE_INCOME">Income</option>
                </Select>
              </div>
              <div className="flex gap-2 justify-end">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    setShowQuickCategoryForm(false)
                    setQuickCategoryName('')
                  }}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  size="sm"
                  loading={createCategoryMutation.isPending}
                  onClick={handleQuickCategoryCreate}
                >
                  Add Category
                </Button>
              </div>
            </div>
          )}

          {/* Amount + Currency side by side */}
          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <FormField label="Amount">
                <Input
                  type="number"
                  name="amount"
                  step="0.01"
                  min="0"
                  required
                  defaultValue={editingTransaction ? moneyToNumber(editingTransaction.amount) : ''}
                  placeholder="0.00"
                />
              </FormField>
            </div>
            <div>
              <FormField label="Currency">
                <Select
                  name="currency"
                  defaultValue={editingTransaction?.amount.currency || 'SGD'}
                >
                  {DISPLAY_CURRENCIES.map((c) => (
                    <option key={c} value={c}>{c}</option>
                  ))}
                </Select>
              </FormField>
            </div>
          </div>

          <FormField label="Description">
            <Input
              type="text"
              name="description"
              defaultValue={editingTransaction?.description || ''}
              placeholder="What was this for?"
            />
          </FormField>

          <FormField label="Date">
            <Input
              type="date"
              name="date"
              required
              defaultValue={
                editingTransaction?.transactionDate?.split('T')[0] ||
                new Date().toISOString().split('T')[0]
              }
            />
          </FormField>
        </form>
      </Modal>
    </div>
  )
}

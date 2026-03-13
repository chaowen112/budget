import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { transactionApi, categoryApi, assetApi, transferApi, assistantApi } from '../api'
import { formatDate, formatMoney, numberToMoney, moneyToNumber } from '../lib/utils'
import { useAuth } from '../store/AuthContext'
import { DISPLAY_CURRENCIES } from '../store/CurrencyContext'
import type { Transaction, Category, CategoryType, Transfer, AssistantSuggestion } from '../types'
import { Plus, Pencil, Trash2, Search, ArrowDownLeft, ArrowUpRight, ArrowRightLeft } from 'lucide-react'
import { Button, Modal, FormField, Input, Select, useConfirm } from '../components/ui'

const CREATE_CATEGORY_OPTION = '__create_new_category__'
type ModalMode = 'transaction' | 'transfer'

type PaginationItem = number | 'ellipsis-left' | 'ellipsis-right'

function buildPaginationItems(currentPage: number, totalPages: number): PaginationItem[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1)
  }

  let start = Math.max(2, currentPage - 1)
  let end = Math.min(totalPages - 1, currentPage + 1)

  if (currentPage <= 3) {
    start = 2
    end = 4
  }

  if (currentPage >= totalPages - 2) {
    start = totalPages - 3
    end = totalPages - 1
  }

  const items: PaginationItem[] = [1]
  if (start > 2) {
    items.push('ellipsis-left')
  }

  for (let page = start; page <= end; page += 1) {
    items.push(page)
  }

  if (end < totalPages - 1) {
    items.push('ellipsis-right')
  }

  items.push(totalPages)
  return items
}

export default function Transactions() {
  const { user } = useAuth()
  const { confirm } = useConfirm()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [modalMode, setModalMode] = useState<ModalMode>('transaction')
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(null)
  const [editingTransfer, setEditingTransfer] = useState<Transfer | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [searchParams] = useSearchParams()
  const [filterCategory, setFilterCategory] = useState<string>(searchParams.get('categoryId') || '')
  const [filterStartDate] = useState<string>(searchParams.get('startDate') || '')
  const [filterEndDate] = useState<string>(searchParams.get('endDate') || '')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(100)
  const [transactionCategoryId, setTransactionCategoryId] = useState('')
  const [transactionSourceAssetId, setTransactionSourceAssetId] = useState('')
  const [transactionAmountInput, setTransactionAmountInput] = useState('')
  const [transactionCurrencyInput, setTransactionCurrencyInput] = useState(user?.baseCurrency || 'SGD')
  const [transactionDescriptionInput, setTransactionDescriptionInput] = useState('')
  const [transactionDateInput, setTransactionDateInput] = useState(new Date().toISOString().split('T')[0])
  const [showQuickCategoryForm, setShowQuickCategoryForm] = useState(false)
  const [quickCategoryName, setQuickCategoryName] = useState('')
  const [quickCategoryType, setQuickCategoryType] = useState<CategoryType>('TRANSACTION_TYPE_EXPENSE')
  const [transferFromAssetId, setTransferFromAssetId] = useState('')
  const [transferToAssetId, setTransferToAssetId] = useState('')
  const [transferFromAmount, setTransferFromAmount] = useState('')
  const [transferToAmount, setTransferToAmount] = useState('')
  const [transferDescriptionInput, setTransferDescriptionInput] = useState('')
  const [transferDateInput, setTransferDateInput] = useState(new Date().toISOString().split('T')[0])
  const [transactionTagsInput, setTransactionTagsInput] = useState('')
  const [transactionBudgetAmountInput, setTransactionBudgetAmountInput] = useState('')
  const [assistantMessage, setAssistantMessage] = useState('')
  const [assistantImageDataUrl, setAssistantImageDataUrl] = useState('')
  const [assistantImageName, setAssistantImageName] = useState('')
  const [assistantSuggestion, setAssistantSuggestion] = useState<AssistantSuggestion | null>(null)
  const [assistantCompletionNote, setAssistantCompletionNote] = useState('')

  const { data: transactionsData, isLoading } = useQuery({
    queryKey: ['transactions', currentPage, pageSize, filterCategory, searchTerm, filterStartDate, filterEndDate],
    queryFn: () =>
      transactionApi.list({
        page: currentPage,
        pageSize,
        categoryId: filterCategory || undefined,
        keyword: searchTerm.trim() || undefined,
        startDate: filterStartDate || undefined,
        endDate: filterEndDate || undefined,
      }),
  })

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoryApi.list(),
  })

  const { data: assets } = useQuery({
    queryKey: ['assets', 'transaction-source'],
    queryFn: () => assetApi.list(),
  })

  const { data: transfers } = useQuery({
    queryKey: ['transfers'],
    queryFn: transferApi.list,
  })

  const { data: transactionSourceLinks } = useQuery({
    queryKey: ['transaction-source-links'],
    queryFn: transactionApi.listSourceLinks,
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

  const createTransferMutation = useMutation({
    mutationFn: transferApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries()
      queryClient.refetchQueries({ type: 'active' })
      setIsModalOpen(false)
    },
  })

  const updateTransferMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof transferApi.update>[1] }) => transferApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries()
      queryClient.refetchQueries({ type: 'active' })
      setIsModalOpen(false)
      setEditingTransfer(null)
    },
  })

  const deleteTransferMutation = useMutation({
    mutationFn: transferApi.delete,
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

  const parseAssistantMutation = useMutation({
    mutationFn: assistantApi.parseTransactionInput,
    onSuccess: (res) => {
      setAssistantSuggestion(res.suggestion)
    },
  })

  const transactions = transactionsData?.transactions || []
  const transactionPagination = transactionsData?.pagination
  const totalPages = Math.max(transactionPagination?.totalPages || 1, 1)
  const totalCount = transactionPagination?.totalCount || transactions.length
  const paginationItems = buildPaginationItems(currentPage, totalPages)

  const sourceLinkMap = new Map((transactionSourceLinks || []).map((item) => [item.transactionId, item]))
  const transactionsWithSource = transactions.map((t) => {
    const link = sourceLinkMap.get(t.id)
    return {
      ...t,
      sourceAssetId: link?.assetId,
      sourceAssetName: link?.assetName,
    }
  })
  const hasCategories = (categories?.length || 0) > 0
  const hasAssets = (assets?.length || 0) > 0
  const filteredTransactions = transactionsWithSource

  const filteredTransfers = (transfers || []).filter((t) => {
    const q = searchTerm.toLowerCase()
    const matchesSearch =
      !searchTerm ||
      (t.description || '').toLowerCase().includes(q) ||
      (t.fromAssetName || '').toLowerCase().includes(q) ||
      (t.toAssetName || '').toLowerCase().includes(q)
    const matchesCategory = !filterCategory
    return matchesSearch && matchesCategory
  })

  const timelineItems = [
    ...filteredTransactions.map((transaction) => ({ kind: 'transaction' as const, date: transaction.transactionDate, transaction })),
    ...filteredTransfers.map((transfer) => ({ kind: 'transfer' as const, date: transfer.transferDate, transfer })),
  ].sort((a, b) => {
    const timeA = new Date(a.date).getTime()
    const timeB = new Date(b.date).getTime()
    if (timeA !== timeB) return timeB - timeA
    const createdAtA = a.kind === 'transaction' ? a.transaction.createdAt : a.transfer.createdAt
    const createdAtB = b.kind === 'transaction' ? b.transaction.createdAt : b.transfer.createdAt
    return new Date(createdAtB).getTime() - new Date(createdAtA).getTime()
  })

  const selectedFromAsset = assets?.find((a) => a.id === transferFromAssetId)
  const selectedToAsset = assets?.find((a) => a.id === transferToAssetId)
  const transferFromCurrency = selectedFromAsset?.currency || editingTransfer?.fromCurrency || user?.baseCurrency || 'SGD'
  const transferToCurrency = selectedToAsset?.currency || editingTransfer?.toCurrency || user?.baseCurrency || 'SGD'
  const transferRequiresFx = !!selectedFromAsset && !!selectedToAsset && transferFromCurrency !== transferToCurrency

  const computedFxRate = (() => {
    const from = parseFloat(transferFromAmount || '0')
    const to = parseFloat(transferToAmount || '0')
    if (from > 0 && to > 0) {
      return to / from
    }
    return null
  })()

  const getCategoryType = (categoryId: string): CategoryType | undefined => {
    return categories?.find((c) => c.id === categoryId)?.type
  }

  const selectedSourceAsset = assets?.find((a) => a.id === transactionSourceAssetId)
  const sourceAssetBalance = selectedSourceAsset ? parseFloat(selectedSourceAsset.currentValue) : null
  const transactionAmount = parseFloat(transactionAmountInput || '0')
  const isExpenseCategory = getCategoryType(transactionCategoryId) === 'TRANSACTION_TYPE_EXPENSE'
  const isSourceLiability = selectedSourceAsset?.isLiability ?? false
  const sourceAssetBalanceAfter =
    sourceAssetBalance !== null && transactionAmount > 0
      ? isExpenseCategory !== isSourceLiability
        ? sourceAssetBalance - transactionAmount
        : sourceAssetBalance + transactionAmount
      : null

  const fromAssetBalance = selectedFromAsset ? parseFloat(selectedFromAsset.currentValue) : null
  const toAssetBalance = selectedToAsset ? parseFloat(selectedToAsset.currentValue) : null
  const parsedTransferFromAmount = parseFloat(transferFromAmount || '0')
  const parsedTransferToAmount = parseFloat(transferToAmount || '0')
  const fromAssetBalanceAfter =
    fromAssetBalance !== null && parsedTransferFromAmount > 0
      ? fromAssetBalance - parsedTransferFromAmount
      : null
  const toAssetBalanceAfter =
    toAssetBalance !== null
      ? toAssetBalance + (parsedTransferToAmount > 0 ? parsedTransferToAmount : parsedTransferFromAmount > 0 ? parsedTransferFromAmount : 0)
      : null

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    if (modalMode === 'transfer') {
      const fromAmount = parseFloat(transferFromAmount || '0')
      const toAmountRaw = transferToAmount.trim()
      const dateStr = transferDateInput
      const transferDate = new Date(dateStr).toISOString()
      const fromCurrency = transferFromCurrency
      const toCurrency = transferToCurrency

      if (!(fromAmount > 0)) return
      if (transferRequiresFx && !toAmountRaw) return

      const normalizedToAmount = toAmountRaw
        ? parseFloat(toAmountRaw)
        : fromAmount

      if (!(normalizedToAmount > 0)) return

      const exchangeRate = normalizedToAmount / fromAmount

      const transferData = {
        fromAssetId: (formData.get('fromAssetId') as string) || transferFromAssetId,
        toAssetId: (formData.get('toAssetId') as string) || transferToAssetId,
        fromAmount: fromAmount.toFixed(2),
        toAmount: normalizedToAmount.toFixed(2),
        fromCurrency,
        toCurrency,
        exchangeRate: exchangeRate.toFixed(10),
        transferDate,
        description: transferDescriptionInput || '',
      }

      if (editingTransfer) {
        updateTransferMutation.mutate({ id: editingTransfer.id, data: transferData })
      } else {
        createTransferMutation.mutate(transferData)
      }
      return
    }

    const amount = parseFloat(transactionAmountInput || '0')
    if (!(amount > 0)) return
    const currency = transactionCurrencyInput || user?.baseCurrency || 'SGD'
    const dateStr = transactionDateInput
    const transactionDate = new Date(dateStr).toISOString()
    const parsedTags = transactionTagsInput
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    const budgetAmt = transactionBudgetAmountInput.trim()
    const data = {
      categoryId: (formData.get('categoryId') as string) || transactionCategoryId,
      amount: numberToMoney(amount, currency),
      description: transactionDescriptionInput,
      transactionDate,
      ...(parsedTags.length > 0 ? { tags: parsedTags } : {}),
      ...(budgetAmt !== '' ? { budgetAmount: budgetAmt } : {}),
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

  const closeModal = () => {
    setIsModalOpen(false)
    setModalMode('transaction')
    setEditingTransaction(null)
    setEditingTransfer(null)
    setTransactionCategoryId('')
    setTransactionSourceAssetId('')
    setTransactionAmountInput('')
    setTransactionCurrencyInput(user?.baseCurrency || 'SGD')
    setTransactionDescriptionInput('')
    setTransactionTagsInput('')
    setTransactionBudgetAmountInput('')
    setTransactionDateInput(new Date().toISOString().split('T')[0])
    setTransferFromAssetId('')
    setTransferToAssetId('')
    setTransferFromAmount('')
    setTransferToAmount('')
    setTransferDescriptionInput('')
    setTransferDateInput(new Date().toISOString().split('T')[0])
    setAssistantCompletionNote('')
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

  const normalizeHint = (value: string) => value.toLowerCase().replace(/[^a-z0-9]/g, '')

  const findAssetByHint = (hint?: string) => {
    if (!hint) return undefined
    const h = hint.toLowerCase().trim()
    const hNorm = normalizeHint(h)
    return (
      assets?.find((a) => a.name.toLowerCase() === h) ||
      assets?.find((a) => a.name.toLowerCase().includes(h)) ||
      assets?.find((a) => normalizeHint(a.name) === hNorm) ||
      assets?.find((a) => normalizeHint(a.name).includes(hNorm))
    )
  }

  const findCategoryByHint = (hint?: string, type?: CategoryType) => {
    if (!hint) return undefined
    const h = hint.toLowerCase()
    const candidates = categories?.filter((c) => (type ? c.type === type : true)) || []
    return candidates.find((c) => c.name.toLowerCase() === h) || candidates.find((c) => c.name.toLowerCase().includes(h))
  }

  const openAssistantTransactionModal = (params: {
    categoryId?: string
    sourceAssetId?: string
    amount?: string
    currency?: string
    description?: string
    transactionDate?: string
  }) => {
    setAssistantCompletionNote('AI suggestion needs a few details. Please complete the form and submit.')
    setEditingTransaction(null)
    setEditingTransfer(null)
    setModalMode('transaction')
    setShowQuickCategoryForm(false)
    setQuickCategoryName('')
    setQuickCategoryType('TRANSACTION_TYPE_EXPENSE')
    setTransactionCategoryId(params.categoryId || '')
    setTransactionSourceAssetId(params.sourceAssetId || '')
    setTransactionAmountInput(params.amount || '')
    setTransactionCurrencyInput(params.currency || user?.baseCurrency || 'SGD')
    setTransactionDescriptionInput(params.description || '')
    setTransactionDateInput((params.transactionDate || new Date().toISOString()).split('T')[0])
    setIsModalOpen(true)
  }

  const openAssistantTransferModal = (params: {
    fromAssetId?: string
    toAssetId?: string
    fromAmount?: string
    toAmount?: string
    description?: string
    transferDate?: string
  }) => {
    setAssistantCompletionNote('AI suggestion needs a few details. Please complete the transfer form and submit.')
    setEditingTransaction(null)
    setEditingTransfer(null)
    setModalMode('transfer')
    setShowQuickCategoryForm(false)
    setQuickCategoryName('')
    setQuickCategoryType('TRANSACTION_TYPE_EXPENSE')
    setTransferFromAssetId(params.fromAssetId || '')
    setTransferToAssetId(params.toAssetId || '')
    setTransferFromAmount(params.fromAmount || '')
    setTransferToAmount(params.toAmount || '')
    setTransferDescriptionInput(params.description || '')
    setTransferDateInput((params.transferDate || new Date().toISOString()).split('T')[0])
    setIsModalOpen(true)
  }

  const applyAssistantSuggestion = async () => {
    if (!assistantSuggestion) return

    const desc = assistantSuggestion.description || ''
    const looksLikeDebtPayment = /paid off|repay|settle|credit card bill|loan repayment|pay.+from/i.test(desc)
    const looksLikeCardChargePurchase = /paid by credit card|charged to|using credit card|via credit card|on credit card/i.test(desc)

    if (assistantSuggestion.entryType === 'transfer' || looksLikeDebtPayment) {
      const fromAsset = findAssetByHint(assistantSuggestion.fromAsset || assistantSuggestion.sourceAsset)
      let toAsset = findAssetByHint(assistantSuggestion.toAsset)
      if (!toAsset && looksLikeDebtPayment) {
        toAsset = assets?.find((a) => a.isLiability) || undefined
      }
      const fromAmount = parseFloat(assistantSuggestion.fromAmount || assistantSuggestion.amount || '0')
      if (!fromAsset || !toAsset || !(fromAmount > 0)) {
        openAssistantTransferModal({
          fromAssetId: fromAsset?.id,
          toAssetId: toAsset?.id,
          fromAmount: assistantSuggestion.fromAmount || assistantSuggestion.amount || '',
          toAmount: assistantSuggestion.toAmount || '',
          description: assistantSuggestion.description || '',
          transferDate: assistantSuggestion.transactionDate || new Date().toISOString(),
        })
        return
      }
      const toAmount = assistantSuggestion.toAmount || undefined
      const date = assistantSuggestion.transactionDate || new Date().toISOString()
      createTransferMutation.mutate({
        fromAssetId: fromAsset.id,
        toAssetId: toAsset.id,
        fromAmount: fromAmount.toFixed(2),
        toAmount: toAmount ? parseFloat(toAmount).toFixed(2) : undefined,
        fromCurrency: fromAsset.currency,
        toCurrency: toAsset.currency,
        exchangeRate: undefined,
        transferDate: date,
        description: assistantSuggestion.description || `Transfer ${fromAsset.name} to ${toAsset.name}`,
      })
      return
    }

    const amount = parseFloat(assistantSuggestion.amount || assistantSuggestion.fromAmount || '0')
    const categoryType = assistantSuggestion.categoryType || 'TRANSACTION_TYPE_EXPENSE'
    let category = findCategoryByHint(assistantSuggestion.categoryName, categoryType)
    let sourceAsset = findAssetByHint(assistantSuggestion.sourceAsset || assistantSuggestion.fromAsset)
    if (!sourceAsset && looksLikeCardChargePurchase) {
      sourceAsset = assets?.find((a) => a.isLiability && /credit card|card/i.test(a.name)) || assets?.find((a) => a.isLiability)
    }
    if (!category) {
      const fallbackName = assistantSuggestion.categoryName || (categoryType === 'TRANSACTION_TYPE_INCOME' ? 'Other Income' : 'Other Expense')
      try {
        category = await createCategoryMutation.mutateAsync({ name: fallbackName, type: categoryType })
      } catch {
        // fall through to validation alert
      }
    }

    if (!(amount > 0) || !category || !sourceAsset) {
      openAssistantTransactionModal({
        categoryId: category?.id,
        sourceAssetId: sourceAsset?.id,
        amount: assistantSuggestion.amount || assistantSuggestion.fromAmount || '',
        currency: assistantSuggestion.currency || sourceAsset?.currency || user?.baseCurrency || 'SGD',
        description: assistantSuggestion.description || '',
        transactionDate: assistantSuggestion.transactionDate || new Date().toISOString(),
      })
      return
    }

    createMutation.mutate({
      categoryId: category.id,
      sourceAssetId: sourceAsset.id,
      amount: numberToMoney(amount, assistantSuggestion.currency || sourceAsset.currency),
      transactionDate: assistantSuggestion.transactionDate || new Date().toISOString(),
      description: assistantSuggestion.description || category.name,
    })
  }

  const handleAssistantImageChange = async (file: File | null) => {
    if (!file) {
      setAssistantImageDataUrl('')
      setAssistantImageName('')
      return
    }

    const reader = new FileReader()
    const dataUrl = await new Promise<string>((resolve, reject) => {
      reader.onload = () => resolve(String(reader.result || ''))
      reader.onerror = reject
      reader.readAsDataURL(file)
    })
    setAssistantImageDataUrl(dataUrl)
    setAssistantImageName(file.name)
  }

  const handleAssistantParse = () => {
    parseAssistantMutation.mutate({
      message: assistantMessage,
      imageDataUrl: assistantImageDataUrl || undefined,
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
            setEditingTransfer(null)
            setModalMode('transaction')
            setShowQuickCategoryForm(false)
            setQuickCategoryName('')
            setQuickCategoryType('TRANSACTION_TYPE_EXPENSE')
            setTransactionCategoryId('')
            setTransactionSourceAssetId('')
            setTransactionAmountInput('')
            setTransactionCurrencyInput(user?.baseCurrency || 'SGD')
            setTransactionDescriptionInput('')
            setTransactionTagsInput('')
            setTransactionBudgetAmountInput('')
            setTransactionDateInput(new Date().toISOString().split('T')[0])
            setTransferFromAssetId('')
            setTransferToAssetId('')
            setTransferFromAmount('')
            setTransferToAmount('')
            setTransferDescriptionInput('')
            setTransferDateInput(new Date().toISOString().split('T')[0])
            setAssistantCompletionNote('')
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
            onChange={(e) => {
              setSearchTerm(e.target.value)
              setCurrentPage(1)
            }}
            className="pl-9"
          />
        </div>
        <Select
          value={filterCategory}
          onChange={(e) => {
            setFilterCategory(e.target.value)
            setCurrentPage(1)
          }}
          className="sm:w-48"
        >
          <option value="">All Categories</option>
          {categories?.map((cat) => (
            <option key={cat.id} value={cat.id}>{cat.name}</option>
          ))}
        </Select>
      </div>

      {/* AI Assistant */}
      <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-4 sm:p-5 shadow-sm space-y-3">
        <div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Transactions Assistant</h2>
          <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">Describe a transaction or upload a receipt photo, then create entry from suggestion.</p>
        </div>

        <textarea
          value={assistantMessage}
          onChange={(e) => setAssistantMessage(e.target.value)}
          placeholder="e.g. paid 12.80 SGD for lunch at Toast Box from OCBC yesterday"
          className="w-full min-h-[84px] rounded-xl border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-3 py-2 text-sm text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 dark:placeholder:text-zinc-500 focus:outline-none focus:ring-2 focus:ring-violet-500/50"
        />

        <div className="flex flex-col sm:flex-row sm:items-center gap-3">
          <Input type="file" accept="image/*" onChange={(e) => handleAssistantImageChange(e.target.files?.[0] || null)} />
          <Button type="button" onClick={handleAssistantParse} loading={parseAssistantMutation.isPending}>
            Parse with AI
          </Button>
        </div>
        {assistantImageName && (
          <p className="text-xs text-zinc-500 dark:text-zinc-400">Attached: {assistantImageName}</p>
        )}

        {assistantSuggestion && (
          <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-xs px-2 py-1 rounded bg-violet-100 dark:bg-violet-500/20 text-violet-700 dark:text-violet-300">
                {assistantSuggestion.entryType === 'transfer' ? 'Transfer Suggestion' : 'Transaction Suggestion'}
              </span>
              <span className="text-xs text-zinc-500 dark:text-zinc-400">Confidence {(assistantSuggestion.confidence * 100).toFixed(0)}%</span>
            </div>
            <p className="text-sm text-zinc-700 dark:text-zinc-300">{assistantSuggestion.description || 'No description'}</p>
            <div className="text-xs text-zinc-500 dark:text-zinc-400 space-y-1">
              {assistantSuggestion.entryType === 'transfer' ? (
                <>
                  <p>From: {assistantSuggestion.fromAsset || assistantSuggestion.sourceAsset || '-'} · Amount: {assistantSuggestion.fromAmount || assistantSuggestion.amount || '-'}</p>
                  <p>To: {assistantSuggestion.toAsset || '-'} · To Amount: {assistantSuggestion.toAmount || '-'}</p>
                </>
              ) : (
                <>
                  <p>Category: {assistantSuggestion.categoryName || '-'} ({assistantSuggestion.categoryType || '-'})</p>
                  <p>Source Asset: {assistantSuggestion.sourceAsset || '-'}</p>
                  <p>Amount: {assistantSuggestion.amount || '-'} {assistantSuggestion.currency || ''}</p>
                </>
              )}
              {assistantSuggestion.missingFields.length > 0 && (
                <p>Missing: {assistantSuggestion.missingFields.join(', ')}</p>
              )}
            </div>
            <div className="flex justify-end">
              <Button type="button" size="sm" onClick={applyAssistantSuggestion}>
                Create Entry from Suggestion
              </Button>
            </div>
          </div>
        )}
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
        ) : timelineItems.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-zinc-400 dark:text-zinc-500">
            <Search className="h-8 w-8 mb-3 opacity-40" />
            <p className="text-sm">No transactions found</p>
          </div>
        ) : (
          <div className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {timelineItems.map((item) => {
              if (item.kind === 'transfer') {
                const transfer = item.transfer
                return (
                  <div
                    key={`transfer-${transfer.id}`}
                    className="flex items-start sm:items-center gap-3 sm:gap-4 px-4 sm:px-5 py-3.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors duration-150"
                  >
                    <div className="h-9 w-9 rounded-xl flex items-center justify-center flex-shrink-0 bg-sky-50 dark:bg-sky-500/10">
                      <ArrowRightLeft className="h-4 w-4 text-sky-600 dark:text-sky-400" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">
                        {transfer.description || `Transfer ${transfer.fromAssetName} -> ${transfer.toAssetName}`}
                      </p>
                      <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                        <span className="inline-flex items-center px-1.5 py-0.5 rounded bg-sky-100 dark:bg-sky-500/20 text-sky-700 dark:text-sky-300 mr-1.5">Transfer</span>
                        {transfer.fromAssetName} {'->'} {transfer.toAssetName} {'·'} {formatDate(transfer.transferDate)}
                      </p>
                    </div>
                    <div className="flex flex-col items-end gap-1.5 sm:gap-2 flex-shrink-0">
                      <span className="text-sm font-semibold tabular-nums text-sky-600 dark:text-sky-400">
                        {formatMoney({ amount: transfer.fromAmount, currency: transfer.fromCurrency })}
                      </span>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => {
                            setEditingTransfer(transfer)
                            setEditingTransaction(null)
                            setModalMode('transfer')
                            setShowQuickCategoryForm(false)
                            setQuickCategoryName('')
                            setQuickCategoryType('TRANSACTION_TYPE_EXPENSE')
                            setTransferFromAssetId(transfer.fromAssetId)
                            setTransferToAssetId(transfer.toAssetId)
                            setTransferFromAmount(transfer.fromAmount)
                            setTransferToAmount(transfer.toAmount)
                            setTransferDescriptionInput(transfer.description || '')
                            setTransferDateInput(transfer.transferDate?.split('T')[0] || new Date().toISOString().split('T')[0])
                            setAssistantCompletionNote('')
                            setIsModalOpen(true)
                          }}
                          className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors duration-150"
                          title="Edit transfer"
                        >
                          <Pencil className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                        </button>
                        <button
                          onClick={async () => {
                            const ok = await confirm({ message: 'Delete this transfer?', variant: 'danger', confirmLabel: 'Delete' }); if (ok) {
                              deleteTransferMutation.mutate(transfer.id)
                            }
                          }}
                          className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors duration-150"
                          title="Delete transfer"
                        >
                          <Trash2 className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                        </button>
                      </div>
                    </div>
                  </div>
                )
              }
              const transaction = item.transaction
              const isExpense = getCategoryType(transaction.categoryId) === 'TRANSACTION_TYPE_EXPENSE'
              return (
                <div
                  key={`tx-${transaction.id}`}
                  className="flex items-start sm:items-center gap-3 sm:gap-4 px-4 sm:px-5 py-3.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors duration-150"
                >
                  {/* Icon */}
                  <div
                    className={`h-9 w-9 rounded-xl flex items-center justify-center flex-shrink-0 ${isExpense
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
                      {transaction.categoryName}
                      {transaction.sourceAssetName ? ` · ${transaction.sourceAssetName}` : ''}
                      {' · '}
                      {formatDate(transaction.transactionDate)}
                    </p>
                    {(transaction.budgetAmount || (transaction.tags && transaction.tags.length > 0)) && (
                      <div className="flex flex-wrap items-center gap-1 mt-1">
                        {transaction.budgetAmount && (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300">
                            Budget: {formatMoney(transaction.budgetAmount)}
                          </span>
                        )}
                        {transaction.tags?.map((tag) => (
                          <span
                            key={tag}
                            className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>

                  {/* Amount + Actions */}
                  <div className="flex flex-col items-end gap-1.5 sm:gap-2 flex-shrink-0">
                    <span
                      className={`text-sm font-semibold tabular-nums ${isExpense
                        ? 'text-red-500 dark:text-red-400'
                        : 'text-emerald-600 dark:text-emerald-400'
                        }`}
                    >
                      {isExpense ? '-' : '+'}{formatMoney(transaction.amount)}
                    </span>

                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => {
                          setEditingTransaction(transaction)
                          setEditingTransfer(null)
                          setModalMode('transaction')
                          setShowQuickCategoryForm(false)
                          setQuickCategoryName('')
                          setQuickCategoryType('TRANSACTION_TYPE_EXPENSE')
                          setTransactionCategoryId(transaction.categoryId)
                          setTransactionSourceAssetId(transaction.sourceAssetId || '')
                          setTransactionAmountInput(String(moneyToNumber(transaction.amount)))
                          setTransactionCurrencyInput(transaction.amount.currency)
                          setTransactionDescriptionInput(transaction.description || '')
                          setTransactionTagsInput((transaction.tags || []).join(', '))
                          setTransactionBudgetAmountInput(transaction.budgetAmount?.amount || '')
                          setTransactionDateInput(transaction.transactionDate?.split('T')[0] || new Date().toISOString().split('T')[0])
                          setAssistantCompletionNote('')
                          setIsModalOpen(true)
                        }}
                        className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors duration-150"
                        title="Edit transaction"
                      >
                        <Pencil className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                      </button>
                      <button
                        onClick={async () => {
                          const ok = await confirm({ message: 'Delete this transaction?', variant: 'danger', confirmLabel: 'Delete' }); if (ok) {
                            deleteMutation.mutate(transaction.id)
                          }
                        }}
                        className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors duration-150"
                        title="Delete transaction"
                      >
                        <Trash2 className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}

        <div className="border-t border-zinc-100 dark:border-zinc-800 px-4 sm:px-5 py-3 flex justify-end">
          <div className="flex flex-wrap items-center justify-end gap-3">
            <span className="text-xs text-zinc-500 dark:text-zinc-400">Showing {transactions.length} of {totalCount}</span>
            <div className="flex items-center gap-2 whitespace-nowrap">
              <span className="text-xs text-zinc-500 dark:text-zinc-400">Page size</span>
              <Select
                value={String(pageSize)}
                onChange={(e) => {
                  setPageSize(Number(e.target.value))
                  setCurrentPage(1)
                }}
                className="w-24"
              >
                <option value="20">20</option>
                <option value="50">50</option>
                <option value="100">100</option>
                <option value="200">200</option>
              </Select>
              <nav aria-label="Transactions pagination">
                <ul className="pagination pagination-sm mb-0">
                  <li className={`page-item ${currentPage <= 1 ? 'disabled' : ''}`}>
                    <button
                      type="button"
                      className="page-link"
                      onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
                      disabled={currentPage <= 1}
                    >
                      Previous
                    </button>
                  </li>
                  {paginationItems.map((item, index) => {
                    if (typeof item !== 'number') {
                      return (
                        <li key={`${item}-${index}`} className="page-item disabled" aria-hidden="true">
                          <span className="page-link">...</span>
                        </li>
                      )
                    }

                    return (
                      <li key={item} className={`page-item ${item === currentPage ? 'active' : ''}`}>
                        <button
                          type="button"
                          className="page-link"
                          onClick={() => setCurrentPage(item)}
                          aria-current={item === currentPage ? 'page' : undefined}
                        >
                          {item}
                        </button>
                      </li>
                    )
                  })}
                  <li className={`page-item ${currentPage >= totalPages ? 'disabled' : ''}`}>
                    <button
                      type="button"
                      className="page-link"
                      onClick={() => setCurrentPage((prev) => Math.min(totalPages, prev + 1))}
                      disabled={currentPage >= totalPages}
                    >
                      Next
                    </button>
                  </li>
                </ul>
              </nav>
            </div>
          </div>
        </div>
      </div>

      {/* Modal */}
      <Modal
        open={isModalOpen}
        onClose={closeModal}
        title={modalMode === 'transfer' ? (editingTransfer ? 'Edit Transfer' : 'Add Transfer') : (editingTransaction ? 'Edit Transaction' : 'Add Transaction')}
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={closeModal}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="transaction-form"
              loading={createMutation.isPending || updateMutation.isPending || createTransferMutation.isPending || updateTransferMutation.isPending}
            >
              {modalMode === 'transfer' ? (editingTransfer ? 'Update' : 'Add') : (editingTransaction ? 'Update' : 'Add')}
            </Button>
          </div>
        }
      >
        <form id="transaction-form" onSubmit={handleSubmit} className="space-y-4">
          {assistantCompletionNote && (
            <div className="rounded-xl border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-300">
              {assistantCompletionNote}
            </div>
          )}

          <FormField label="Entry Type">
            <Select
              value={modalMode}
              onChange={(e) => setModalMode(e.target.value as ModalMode)}
              disabled={!!editingTransaction || !!editingTransfer}
            >
              <option value="transaction">Income / Expense</option>
              <option value="transfer">Transfer</option>
            </Select>
          </FormField>

          {modalMode === 'transfer' ? (
            <>
              <FormField label="From Asset">
                <Select
                  name="fromAssetId"
                  required
                  value={transferFromAssetId}
                  onChange={(e) => setTransferFromAssetId(e.target.value)}
                >
                  <option value="">Select source asset</option>
                  {assets?.map((asset) => (
                    <option key={asset.id} value={asset.id}>{asset.name}</option>
                  ))}
                </Select>
              </FormField>
              <FormField label="To Asset">
                <Select
                  name="toAssetId"
                  required
                  value={transferToAssetId}
                  onChange={(e) => setTransferToAssetId(e.target.value)}
                >
                  <option value="">Select destination asset</option>
                  {assets?.map((asset) => (
                    <option key={asset.id} value={asset.id}>{asset.name}</option>
                  ))}
                </Select>
              </FormField>

              {(selectedFromAsset || selectedToAsset) && (
                <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-1.5 bg-zinc-50/60 dark:bg-zinc-800/30">
                  {selectedFromAsset && fromAssetBalance !== null && (
                    <>
                      <div className="flex justify-between text-xs">
                        <span className="text-zinc-500 dark:text-zinc-400">{selectedFromAsset.name} balance</span>
                        <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                          {formatMoney({ amount: String(fromAssetBalance), currency: selectedFromAsset.currency })}
                        </span>
                      </div>
                      {fromAssetBalanceAfter !== null && (
                        <div className="flex justify-between text-xs">
                          <span className="text-zinc-500 dark:text-zinc-400">{selectedFromAsset.name} after</span>
                          <span className={`font-medium tabular-nums ${fromAssetBalanceAfter < 0 ? 'text-red-500 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'}`}>
                            {formatMoney({ amount: fromAssetBalanceAfter.toFixed(2), currency: selectedFromAsset.currency })}
                          </span>
                        </div>
                      )}
                    </>
                  )}
                  {selectedToAsset && toAssetBalance !== null && (
                    <>
                      <div className="flex justify-between text-xs">
                        <span className="text-zinc-500 dark:text-zinc-400">{selectedToAsset.name} balance</span>
                        <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                          {formatMoney({ amount: String(toAssetBalance), currency: selectedToAsset.currency })}
                        </span>
                      </div>
                      {toAssetBalanceAfter !== null && toAssetBalanceAfter !== toAssetBalance && (
                        <div className="flex justify-between text-xs">
                          <span className="text-zinc-500 dark:text-zinc-400">{selectedToAsset.name} after</span>
                          <span className="font-medium tabular-nums text-emerald-600 dark:text-emerald-400">
                            {formatMoney({ amount: toAssetBalanceAfter.toFixed(2), currency: selectedToAsset.currency })}
                          </span>
                        </div>
                      )}
                    </>
                  )}
                </div>
              )}

              <div className="grid grid-cols-2 gap-3">
                <FormField label="From Amount">
                  <Input
                    type="number"
                    name="fromAmount"
                    step="0.01"
                    min="0"
                    required
                    value={transferFromAmount}
                    onChange={(e) => setTransferFromAmount(e.target.value)}
                  />
                </FormField>
                <FormField label="From Currency">
                  <Input value={transferFromCurrency} disabled />
                </FormField>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <FormField label={transferRequiresFx ? 'To Amount (Required)' : 'To Amount (Optional)'}>
                  <Input
                    type="number"
                    name="toAmount"
                    step="0.01"
                    min="0"
                    required={transferRequiresFx}
                    value={transferToAmount}
                    onChange={(e) => setTransferToAmount(e.target.value)}
                    placeholder={transferRequiresFx ? 'Required for FX transfer' : 'Leave blank for same amount'}
                  />
                </FormField>
                <FormField label="To Currency">
                  <Input name="toCurrency" value={transferToCurrency} disabled />
                </FormField>
              </div>

              <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 text-xs text-zinc-500 dark:text-zinc-400">
                {transferRequiresFx ? (
                  <>
                    FX transfer detected ({transferFromCurrency} {'->'} {transferToCurrency}).
                    {computedFxRate ? (
                      <span className="block mt-1 text-zinc-700 dark:text-zinc-300">
                        Auto exchange rate: {computedFxRate.toFixed(6)}
                      </span>
                    ) : (
                      <span className="block mt-1">Enter To Amount to auto-calculate exchange rate.</span>
                    )}
                  </>
                ) : (
                  <>
                    Same currency transfer. If To Amount is blank, system uses From Amount automatically (no FX).
                  </>
                )}
              </div>

              <FormField label="Description">
                <Input
                  type="text"
                  name="description"
                  value={transferDescriptionInput}
                  onChange={(e) => setTransferDescriptionInput(e.target.value)}
                  placeholder="e.g. Pay credit card bill"
                />
              </FormField>

              <FormField label="Date">
                <Input
                  type="date"
                  name="date"
                  required
                  value={transferDateInput}
                  onChange={(e) => setTransferDateInput(e.target.value)}
                />
              </FormField>
              <p className="text-xs text-zinc-500 dark:text-zinc-400">Transfers are excluded from income/expense reports.</p>
            </>
          ) : (
            <>
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
                  onChange={(e) => {
                    const assetId = e.target.value
                    setTransactionSourceAssetId(assetId)
                    const asset = assets?.find((a) => a.id === assetId)
                    if (asset) setTransactionCurrencyInput(asset.currency)
                  }}
                >
                  <option value="">{editingTransaction ? 'Keep existing asset link' : 'Select asset'}</option>
                  {assets?.map((asset) => (
                    <option key={asset.id} value={asset.id}>
                      {asset.name} ({asset.currency})
                    </option>
                  ))}
                </Select>
              </FormField>

              {selectedSourceAsset && sourceAssetBalance !== null && (
                <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 p-3 space-y-1.5 bg-zinc-50/60 dark:bg-zinc-800/30">
                  <div className="flex justify-between text-xs">
                    <span className="text-zinc-500 dark:text-zinc-400">Current balance</span>
                    <span className="font-medium tabular-nums text-zinc-900 dark:text-zinc-100">
                      {formatMoney({ amount: String(sourceAssetBalance), currency: selectedSourceAsset.currency })}
                    </span>
                  </div>
                  {sourceAssetBalanceAfter !== null && (
                    <div className="flex justify-between text-xs">
                      <span className="text-zinc-500 dark:text-zinc-400">Balance after</span>
                      <span className={`font-medium tabular-nums ${sourceAssetBalanceAfter < 0 ? 'text-red-500 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'}`}>
                        {formatMoney({ amount: sourceAssetBalanceAfter.toFixed(2), currency: selectedSourceAsset.currency })}
                      </span>
                    </div>
                  )}
                </div>
              )}

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
                      value={transactionAmountInput}
                      onChange={(e) => setTransactionAmountInput(e.target.value)}
                      placeholder="0.00"
                    />
                  </FormField>
                </div>
                <div>
                  <FormField label="Currency">
                    <Select
                      name="currency"
                      value={transactionCurrencyInput}
                      onChange={(e) => setTransactionCurrencyInput(e.target.value)}
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
                  value={transactionDescriptionInput}
                  onChange={(e) => setTransactionDescriptionInput(e.target.value)}
                  placeholder="What was this for?"
                />
              </FormField>

              <FormField label="Tags">
                <Input
                  type="text"
                  name="tags"
                  value={transactionTagsInput}
                  onChange={(e) => setTransactionTagsInput(e.target.value)}
                  placeholder="e.g. food, work, travel (comma-separated)"
                />
              </FormField>

              <FormField label="Budget Amount (optional)">
                <Input
                  type="number"
                  name="budgetAmount"
                  step="0.01"
                  min="0"
                  value={transactionBudgetAmountInput}
                  onChange={(e) => setTransactionBudgetAmountInput(e.target.value)}
                  placeholder={`Leave blank to use full amount toward budget`}
                />
                <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                  How much of this expense counts toward your budget. Leave blank for 100%.
                </p>
              </FormField>

              <FormField label="Date">
                <Input
                  type="date"
                  name="date"
                  required
                  value={transactionDateInput}
                  onChange={(e) => setTransactionDateInput(e.target.value)}
                />
              </FormField>
            </>
          )}
        </form>
      </Modal>
    </div>
  )
}

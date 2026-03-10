import { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { assetApi, reportApi } from '../api'
import { useAuth } from '../store/AuthContext'
import { useCurrency, DISPLAY_CURRENCIES } from '../store/CurrencyContext'
import { formatDate, formatMoney } from '../lib/utils'
import type { Asset, AssetCategory, AssetSnapshot, AssetType } from '../types'
import { Plus, Pencil, Trash2, Building, Car, Coins, CreditCard, Landmark, Wallet, Bitcoin, TrendingDown, TrendingUp, Scale, LineChart as LineChartIcon } from 'lucide-react'
import { Button, Modal, FormField, Input, Select } from '../components/ui'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from 'recharts'

const ASSET_CATEGORY_ICONS: Record<AssetCategory, typeof Wallet> = {
  'ASSET_CATEGORY_CASH': Wallet,
  'ASSET_CATEGORY_BANK': Landmark,
  'ASSET_CATEGORY_INVESTMENT': Coins,
  'ASSET_CATEGORY_PROPERTY': Building,
  'ASSET_CATEGORY_VEHICLE': Car,
  'ASSET_CATEGORY_CRYPTO': Bitcoin,
  'ASSET_CATEGORY_OTHER': CreditCard,
}

const ASSET_CATEGORY_LABELS: Record<AssetCategory, string> = {
  'ASSET_CATEGORY_CASH': 'Cash',
  'ASSET_CATEGORY_BANK': 'Bank Account',
  'ASSET_CATEGORY_INVESTMENT': 'Investment',
  'ASSET_CATEGORY_PROPERTY': 'Property',
  'ASSET_CATEGORY_VEHICLE': 'Vehicle',
  'ASSET_CATEGORY_CRYPTO': 'Cryptocurrency',
  'ASSET_CATEGORY_OTHER': 'Other',
}

function parseAssetValue(value: string): number {
  return parseFloat(value) || 0
}

export default function Assets() {
  const { user } = useAuth()
  const { formatConverted, displayCurrency, convertToDisplayAmount } = useCurrency()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingAsset, setEditingAsset] = useState<Asset | null>(null)
  const [historyAsset, setHistoryAsset] = useState<Asset | null>(null)
  const [showLiabilities, setShowLiabilities] = useState(true)
  const [assetCurrencyInput, setAssetCurrencyInput] = useState(user?.baseCurrency || 'SGD')
  const [assetTypeInput, setAssetTypeInput] = useState('')
  const [selectedCategories, setSelectedCategories] = useState<AssetCategory[]>([])
  const [sortBy, setSortBy] = useState<'name' | 'amount' | 'currency' | 'category'>('name')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')

  const { data: assets, isLoading } = useQuery({
    queryKey: ['assets', showLiabilities],
    queryFn: () => assetApi.list({ includeLiabilities: showLiabilities }),
  })

  const { data: assetTypes } = useQuery({
    queryKey: ['assetTypes'],
    queryFn: () => assetApi.listAssetTypes(),
  })

  const { data: netWorthTrend } = useQuery({
    queryKey: ['netWorthTrend', 12],
    queryFn: () => reportApi.getNetWorthTrend({ months: 12, interval: 'monthly' }),
  })

  const { data: selectedAssetHistory } = useQuery({
    queryKey: ['assetHistory', historyAsset?.id],
    queryFn: () => assetApi.getHistory(historyAsset!.id),
    enabled: !!historyAsset,
  })

  const createMutation = useMutation({
    mutationFn: assetApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['assets'] })
      setIsModalOpen(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof assetApi.update>[1] }) =>
      assetApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['assets'] })
      setIsModalOpen(false)
      setEditingAsset(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: assetApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['assets'] })
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const currentValue = formData.get('currentValue') as string
    const cost = formData.get('cost') as string | undefined

    if (editingAsset) {
      updateMutation.mutate({
        id: editingAsset.id,
        data: {
          assetTypeId: formData.get('assetTypeId') as string,
          name: formData.get('name') as string,
          currency: (formData.get('currency') as string) || assetCurrencyInput,
          currentValue,
          cost: cost || undefined,
        },
      })
    } else {
      createMutation.mutate({
        assetTypeId: formData.get('assetTypeId') as string,
        name: formData.get('name') as string,
        currency: (formData.get('currency') as string) || user?.baseCurrency || 'SGD',
        currentValue,
        cost: cost || undefined,
        isLiability: formData.get('isLiability') === 'true',
      })
    }
  }

  const getCategoryIcon = (category: AssetCategory) => ASSET_CATEGORY_ICONS[category] || CreditCard
  const getCategoryLabel = (category: AssetCategory) => ASSET_CATEGORY_LABELS[category] || 'Other'

  const allAssets = assets || []
  const availableCategories = useMemo(() => {
    return Array.from(new Set(allAssets.map((a) => a.category))) as AssetCategory[]
  }, [allAssets])

  const filteredAndSortedAssets = useMemo(() => {
    const filtered = allAssets.filter((asset) => {
      if (selectedCategories.length === 0) return true
      return selectedCategories.includes(asset.category)
    })

    const sorted = [...filtered].sort((a, b) => {
      let comparison = 0

      if (sortBy === 'name') {
        comparison = a.name.localeCompare(b.name)
      } else if (sortBy === 'currency') {
        comparison = a.currency.localeCompare(b.currency)
      } else if (sortBy === 'category') {
        comparison = getCategoryLabel(a.category).localeCompare(getCategoryLabel(b.category))
      } else if (sortBy === 'amount') {
        comparison = parseAssetValue(a.currentValue) - parseAssetValue(b.currentValue)
      }

      return sortDirection === 'asc' ? comparison : -comparison
    })

    return sorted
  }, [allAssets, selectedCategories, sortBy, sortDirection])

  const assetsList = filteredAndSortedAssets.filter((a) => !a.isLiability)
  const liabilitiesList = filteredAndSortedAssets.filter((a) => a.isLiability)
  const baseCurrency = user?.baseCurrency || 'SGD'
  const assetDisplayCurrency = editingAsset?.currency || assetCurrencyInput
  const totalAssets = assetsList.reduce(
    (sum, a) => sum + convertToDisplayAmount({ amount: a.currentValue, currency: a.currency }),
    0
  )
  const totalLiabilities = liabilitiesList.reduce(
    (sum, a) => sum + convertToDisplayAmount({ amount: a.currentValue, currency: a.currency }),
    0
  )
  const netWorth = totalAssets - totalLiabilities

  const toggleCategoryFilter = (category: AssetCategory) => {
    setSelectedCategories((prev) =>
      prev.includes(category) ? prev.filter((c) => c !== category) : [...prev, category]
    )
  }

  const clearFilters = () => {
    setSelectedCategories([])
    setSortBy('name')
    setSortDirection('asc')
  }

  const groupedAssetTypes =
    assetTypes?.reduce((acc, type) => {
      if (!acc[type.category]) acc[type.category] = []
      acc[type.category].push(type)
      return acc
    }, {} as Record<AssetCategory, AssetType[]>) || {}

  const netWorthTrendData = (netWorthTrend?.trend || []).map((point) => ({
    month: point.month,
    netWorth: parseAssetValue(point.netWorth.amount),
    assets: parseAssetValue(point.assets.amount),
    liabilities: parseAssetValue(point.liabilities.amount),
  }))

  const selectedAssetHistoryData = (selectedAssetHistory || []).map((snapshot: AssetSnapshot) => ({
    date: formatDate(snapshot.recordedAt),
    value: parseAssetValue(snapshot.value),
  }))

  const closeModal = () => {
    setIsModalOpen(false)
    setEditingAsset(null)
    setAssetCurrencyInput(user?.baseCurrency || 'SGD')
    setAssetTypeInput('')
  }

  const closeHistoryModal = () => {
    setHistoryAsset(null)
  }

  const AssetRow = ({ asset, isLiability }: { asset: Asset; isLiability: boolean }) => {
    const Icon = getCategoryIcon(asset.category)
    const costValue = asset.cost ? Number(asset.cost) : 0
    const currentValue = Number(asset.currentValue)
    const diff = currentValue - costValue
    const diffPct = costValue > 0 ? (diff / costValue) * 100 : 0

    return (
      <div className="flex items-center gap-4 px-5 py-3.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors duration-150 group">
        <div
          className={`h-9 w-9 rounded-xl flex items-center justify-center flex-shrink-0 ${isLiability
            ? 'bg-red-50 dark:bg-red-500/10'
            : 'bg-emerald-50 dark:bg-emerald-500/10'
            }`}
        >
          <Icon
            className={`h-4 w-4 ${isLiability ? 'text-red-500 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'
              }`}
          />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">{asset.name}</p>
            {costValue > 0 && !isLiability && (
              <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${diff >= 0 ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400' : 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-400'}`}>
                {diff >= 0 ? '+' : ''}{diffPct.toFixed(2)}%
              </span>
            )}
          </div>
          <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
            {asset.assetTypeName || getCategoryLabel(asset.category)}
          </p>
        </div>
        <div className="flex flex-col items-end">
          <span
            className={`text-sm font-semibold tabular-nums flex-shrink-0 ${isLiability ? 'text-red-500 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'
              }`}
          >
            {isLiability ? '-' : ''}{formatMoney({ amount: String(asset.currentValue), currency: asset.currency })}
          </span>
          {costValue > 0 && !isLiability && (
            <span className="text-[10px] text-zinc-400 dark:text-zinc-500 font-medium mt-0.5">
              Cost: {formatMoney({ amount: String(asset.cost), currency: asset.currency })}
            </span>
          )}
        </div>
        <div className="flex gap-1 ml-2">
          <button
            onClick={() => setHistoryAsset(asset)}
            className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-violet-600 dark:hover:text-violet-400 hover:bg-violet-50 dark:hover:bg-violet-500/10 transition-colors duration-150"
            title="View History"
          >
            <LineChartIcon className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
          </button>
          <button
            onClick={() => {
              setEditingAsset(asset)
              setAssetCurrencyInput(asset.currency)
              setAssetTypeInput(asset.assetTypeId)
              setIsModalOpen(true)
            }}
            className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors duration-150"
          >
            <Pencil className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
          </button>
          <button
            onClick={() => { if (confirm('Delete this asset?')) deleteMutation.mutate(asset.id) }}
            className="h-8 w-8 sm:h-7 sm:w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors duration-150"
          >
            <Trash2 className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            Assets & Liabilities
          </h1>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">Track your net worth</p>
        </div>
        <Button
          icon={<Plus className="h-4 w-4" />}
          onClick={() => {
            setEditingAsset(null)
            setAssetCurrencyInput(user?.baseCurrency || 'SGD')
            setAssetTypeInput('')
            setIsModalOpen(true)
          }}
        >
          Add Asset
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Total Assets */}
        <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm">
          <div className="flex items-center gap-3 mb-3">
            <div className="h-8 w-8 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 flex items-center justify-center">
              <TrendingUp className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
            </div>
            <span className="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
              Total Assets
            </span>
          </div>
          <p className="text-2xl font-bold tracking-tight text-emerald-600 dark:text-emerald-400 tabular-nums">
            {formatConverted({ amount: totalAssets.toString(), currency: displayCurrency })}
          </p>
        </div>

        {/* Total Liabilities */}
        <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm">
          <div className="flex items-center gap-3 mb-3">
            <div className="h-8 w-8 rounded-lg bg-red-50 dark:bg-red-500/10 flex items-center justify-center">
              <TrendingDown className="h-4 w-4 text-red-500 dark:text-red-400" />
            </div>
            <span className="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
              Total Liabilities
            </span>
          </div>
          <p className="text-2xl font-bold tracking-tight text-red-500 dark:text-red-400 tabular-nums">
            {formatConverted({ amount: totalLiabilities.toString(), currency: displayCurrency })}
          </p>
        </div>

        {/* Net Worth */}
        <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm">
          <div className="flex items-center gap-3 mb-3">
            <div className="h-8 w-8 rounded-lg bg-violet-50 dark:bg-violet-500/10 flex items-center justify-center">
              <Scale className="h-4 w-4 text-violet-600 dark:text-violet-400" />
            </div>
            <span className="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
              Net Worth
            </span>
          </div>
          <p
            className={`text-2xl font-bold tracking-tight tabular-nums ${netWorth >= 0
              ? 'text-violet-600 dark:text-violet-400'
              : 'text-red-500 dark:text-red-400'
              }`}
          >
            {formatConverted({ amount: netWorth.toString(), currency: displayCurrency })}
          </p>
        </div>
      </div>

      {/* Net Worth Trend */}
      <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm">
        <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50 mb-4">Net Asset Trend (Last 12 Months)</h2>
        {netWorthTrendData.length > 0 ? (
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={netWorthTrendData}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(113,113,122,0.15)" />
                <XAxis dataKey="month" tick={{ fontSize: 11, fill: '#71717a' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: '#71717a' }} axisLine={false} tickLine={false} />
                <Tooltip
                  formatter={(value: number | undefined, name: string | undefined) => [formatConverted({ amount: String(value ?? 0), currency: baseCurrency }), name || 'Value']}
                  contentStyle={{
                    background: '#18181b',
                    border: '1px solid #3f3f46',
                    borderRadius: 12,
                    fontSize: 12,
                    color: '#fff',
                  }}
                />
                <Legend wrapperStyle={{ fontSize: 11, color: '#71717a' }} />
                <Line type="monotone" dataKey="netWorth" stroke="#8b5cf6" strokeWidth={2.5} dot={false} name="Net Asset" />
                <Line type="monotone" dataKey="assets" stroke="#10b981" strokeWidth={1.8} dot={false} name="Assets" />
                <Line type="monotone" dataKey="liabilities" stroke="#ef4444" strokeWidth={1.8} dot={false} name="Liabilities" />
              </LineChart>
            </ResponsiveContainer>
          </div>
        ) : (
          <div className="h-48 flex items-center justify-center text-sm text-zinc-400 dark:text-zinc-500">
            No trend data yet. Update assets over time to build your chart.
          </div>
        )}
      </div>

      {/* Toggle */}
      <label className="inline-flex items-center gap-2 cursor-pointer select-none">
        <div
          onClick={() => setShowLiabilities((v) => !v)}
          className={`relative w-9 h-5 rounded-full transition-colors duration-200 ${showLiabilities ? 'bg-violet-500' : 'bg-zinc-200 dark:bg-zinc-700'
            }`}
        >
          <div
            className={`absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white shadow-sm transition-transform duration-200 ${showLiabilities ? 'translate-x-4' : 'translate-x-0'
              }`}
          />
        </div>
        <span className="text-sm text-zinc-700 dark:text-zinc-300">Show liabilities</span>
      </label>

      {/* Filters and Sorting */}
      <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-4 shadow-sm space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-end gap-3">
          <div className="flex-1">
            <FormField label="Sort By">
              <Select value={sortBy} onChange={(e) => setSortBy(e.target.value as 'name' | 'amount' | 'currency' | 'category')}>
                <option value="name">Name</option>
                <option value="amount">Amount</option>
                <option value="currency">Currency</option>
                <option value="category">Category</option>
              </Select>
            </FormField>
          </div>
          <div className="flex-1">
            <FormField label="Direction">
              <Select value={sortDirection} onChange={(e) => setSortDirection(e.target.value as 'asc' | 'desc')}>
                <option value="asc">Ascending</option>
                <option value="desc">Descending</option>
              </Select>
            </FormField>
          </div>
          <Button variant="secondary" size="sm" onClick={clearFilters}>Reset</Button>
        </div>

        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400 mb-2">
            Filter by Category (multi-select)
          </p>
          {availableCategories.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {availableCategories.map((category) => {
                const active = selectedCategories.includes(category)
                return (
                  <button
                    key={category}
                    type="button"
                    onClick={() => toggleCategoryFilter(category)}
                    className={`px-3 py-1.5 rounded-full text-xs font-medium border transition-colors ${active
                      ? 'bg-violet-600 border-violet-600 text-white'
                      : 'bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 text-zinc-600 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-700'
                      }`}
                  >
                    {getCategoryLabel(category)}
                  </button>
                )
              })}
            </div>
          ) : (
            <p className="text-sm text-zinc-400 dark:text-zinc-500">No categories available yet.</p>
          )}
        </div>
      </div>

      {/* Assets List */}
      <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 overflow-hidden shadow-sm">
        <div className="px-5 py-3.5 border-b border-zinc-100 dark:border-zinc-800">
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">Assets</h2>
        </div>
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <svg className="animate-spin h-6 w-6 text-violet-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
        ) : assetsList.length === 0 ? (
          <div className="text-center py-10 text-sm text-zinc-400 dark:text-zinc-500">No assets yet</div>
        ) : (
          <div className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {assetsList.map((asset) => (
              <AssetRow key={asset.id} asset={asset} isLiability={false} />
            ))}
          </div>
        )}
      </div>

      {/* Liabilities List */}
      {showLiabilities && (
        <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 overflow-hidden shadow-sm">
          <div className="px-5 py-3.5 border-b border-zinc-100 dark:border-zinc-800">
            <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">Liabilities</h2>
          </div>
          {liabilitiesList.length === 0 ? (
            <div className="text-center py-10 text-sm text-zinc-400 dark:text-zinc-500">No liabilities</div>
          ) : (
            <div className="divide-y divide-zinc-100 dark:divide-zinc-800">
              {liabilitiesList.map((asset) => (
                <AssetRow key={asset.id} asset={asset} isLiability={true} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Modal */}
      <Modal
        open={isModalOpen}
        onClose={closeModal}
        title={editingAsset ? 'Edit Asset' : 'Add Asset'}
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={closeModal}>Cancel</Button>
            <Button
              type="submit"
              form="asset-form"
              loading={createMutation.isPending || updateMutation.isPending}
            >
              {editingAsset ? 'Update' : 'Add'}
            </Button>
          </div>
        }
      >
        <form id="asset-form" onSubmit={handleSubmit} className="space-y-4">
          <FormField label="Name">
            <Input
              type="text"
              name="name"
              required
              defaultValue={editingAsset?.name || ''}
              placeholder="e.g., OCBC Savings"
            />
          </FormField>

          <FormField label="Category / Type">
            <Select
              name="assetTypeId"
              required
              value={assetTypeInput}
              onChange={(e) => setAssetTypeInput(e.target.value)}
            >
              <option value="">Select an asset type...</option>
              {editingAsset && assetTypeInput && !assetTypes?.some((t) => t.id === assetTypeInput) && (
                <option value={assetTypeInput}>{editingAsset.assetTypeName || 'Current type'}</option>
              )}
              {Object.entries(groupedAssetTypes).map(([category, types]) => (
                <optgroup key={category} label={getCategoryLabel(category as AssetCategory)}>
                  {(types as AssetType[]).map((type) => (
                    <option key={type.id} value={type.id}>{type.name}</option>
                  ))}
                </optgroup>
              ))}
            </Select>
          </FormField>

          {!editingAsset && (
            <FormField label="Type">
              <Select name="isLiability">
                <option value="false">Asset</option>
                <option value="true">Liability</option>
              </Select>
            </FormField>
          )}

          <FormField label="Current Value">
            <Input
              type="number"
              name="currentValue"
              step="0.01"
              min="0"
              required
              defaultValue={editingAsset?.currentValue || ''}
              placeholder="0.00"
            />
          </FormField>

          <FormField label="Cost (Optional)">
            <Input
              type="number"
              name="cost"
              step="0.01"
              min="0"
              defaultValue={editingAsset?.cost || ''}
              placeholder="0.00"
            />
          </FormField>

          <FormField label="Asset Currency">
            <Select
              name="currency"
              value={assetCurrencyInput}
              onChange={(e) => setAssetCurrencyInput(e.target.value)}
            >
              {[...new Set([...DISPLAY_CURRENCIES, assetDisplayCurrency].filter(Boolean))].map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </Select>
          </FormField>
        </form>
      </Modal>

      <Modal
        open={!!historyAsset}
        onClose={closeHistoryModal}
        title={historyAsset ? `${historyAsset.name} History` : 'Asset History'}
        footer={
          <div className="flex justify-end">
            <Button variant="secondary" onClick={closeHistoryModal}>Close</Button>
          </div>
        }
      >
        {selectedAssetHistoryData.length > 0 ? (
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={selectedAssetHistoryData}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(113,113,122,0.15)" />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: '#71717a' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: '#71717a' }} axisLine={false} tickLine={false} />
                <Tooltip
                  formatter={(value: number | undefined) => [formatMoney({ amount: String(value ?? 0), currency: historyAsset?.currency || baseCurrency }), 'Value']}
                  contentStyle={{
                    background: '#18181b',
                    border: '1px solid #3f3f46',
                    borderRadius: 12,
                    fontSize: 12,
                    color: '#fff',
                  }}
                />
                <Line type="monotone" dataKey="value" stroke="#3b82f6" strokeWidth={2.5} dot={false} name="Value" />
              </LineChart>
            </ResponsiveContainer>
          </div>
        ) : (
          <div className="py-10 text-center text-sm text-zinc-500 dark:text-zinc-400">
            No history yet for this asset.
          </div>
        )}
      </Modal>
    </div>
  )
}

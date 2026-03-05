import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { assetApi } from '../api'
import { useAuth } from '../store/AuthContext'
import type { Asset, AssetCategory, AssetType } from '../types'
import { Plus, Pencil, Trash2, X, Building, Car, Coins, CreditCard, Landmark, Wallet, Bitcoin } from 'lucide-react'

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

function formatAssetValue(value: string, currency: string): string {
  const num = parseFloat(value)
  return new Intl.NumberFormat('en-SG', {
    style: 'currency',
    currency: currency || 'SGD',
  }).format(num)
}

function parseAssetValue(value: string): number {
  return parseFloat(value) || 0
}

export default function Assets() {
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingAsset, setEditingAsset] = useState<Asset | null>(null)
  const [showLiabilities, setShowLiabilities] = useState(true)

  const { data: assets, isLoading } = useQuery({
    queryKey: ['assets', showLiabilities],
    queryFn: () => assetApi.list({ includeLiabilities: showLiabilities }),
  })

  const { data: assetTypes } = useQuery({
    queryKey: ['assetTypes'],
    queryFn: () => assetApi.listAssetTypes(),
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

    if (editingAsset) {
      updateMutation.mutate({
        id: editingAsset.id,
        data: {
          name: formData.get('name') as string,
          currentValue: currentValue,
        },
      })
    } else {
      createMutation.mutate({
        assetTypeId: formData.get('assetTypeId') as string,
        name: formData.get('name') as string,
        currency: user?.baseCurrency || 'SGD',
        currentValue: currentValue,
        isLiability: formData.get('isLiability') === 'true',
      })
    }
  }

  const getCategoryIcon = (category: AssetCategory) => {
    return ASSET_CATEGORY_ICONS[category] || CreditCard
  }

  const getCategoryLabel = (category: AssetCategory) => {
    return ASSET_CATEGORY_LABELS[category] || 'Other'
  }

  const assetsList = assets?.filter((a) => !a.isLiability) || []
  const liabilitiesList = assets?.filter((a) => a.isLiability) || []
  const currency = user?.baseCurrency || 'SGD'
  const totalAssets = assetsList.reduce((sum, a) => sum + parseAssetValue(a.currentValue), 0)
  const totalLiabilities = liabilitiesList.reduce((sum, a) => sum + parseAssetValue(a.currentValue), 0)
  const netWorth = totalAssets - totalLiabilities

  // Group asset types by category for the select dropdown
  const groupedAssetTypes = assetTypes?.reduce((acc, type) => {
    if (!acc[type.category]) {
      acc[type.category] = []
    }
    acc[type.category].push(type)
    return acc
  }, {} as Record<AssetCategory, AssetType[]>) || {}

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Assets & Liabilities</h1>
          <p className="text-gray-500">Track your net worth</p>
        </div>
        <button
          onClick={() => {
            setEditingAsset(null)
            setIsModalOpen(true)
          }}
          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="h-5 w-5 mr-2" />
          Add Asset
        </button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
          <p className="text-sm text-gray-500">Total Assets</p>
          <p className="text-2xl font-bold text-green-600">
            {formatAssetValue(totalAssets.toString(), currency)}
          </p>
        </div>
        <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
          <p className="text-sm text-gray-500">Total Liabilities</p>
          <p className="text-2xl font-bold text-red-600">
            {formatAssetValue(totalLiabilities.toString(), currency)}
          </p>
        </div>
        <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
          <p className="text-sm text-gray-500">Net Worth</p>
          <p className={`text-2xl font-bold ${netWorth >= 0 ? 'text-blue-600' : 'text-red-600'}`}>
            {formatAssetValue(netWorth.toString(), currency)}
          </p>
        </div>
      </div>

      {/* Toggle */}
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id="showLiabilities"
          checked={showLiabilities}
          onChange={(e) => setShowLiabilities(e.target.checked)}
          className="h-4 w-4 text-blue-600 rounded border-gray-300"
        />
        <label htmlFor="showLiabilities" className="text-sm text-gray-700">Show liabilities</label>
      </div>

      {/* Assets List */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        <div className="p-4 border-b border-gray-100">
          <h2 className="font-semibold text-gray-900">Assets</h2>
        </div>
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : assetsList.length === 0 ? (
          <div className="text-center py-8 text-gray-500">No assets yet</div>
        ) : (
          <div className="divide-y divide-gray-100">
            {assetsList.map((asset) => {
              const Icon = getCategoryIcon(asset.category)
              return (
                <div key={asset.id} className="p-4 hover:bg-gray-50">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="h-10 w-10 rounded-full bg-green-100 flex items-center justify-center">
                        <Icon className="h-5 w-5 text-green-600" />
                      </div>
                      <div>
                        <p className="font-medium text-gray-900">{asset.name}</p>
                        <p className="text-sm text-gray-500">{asset.assetTypeName || getCategoryLabel(asset.category)}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className="font-semibold text-green-600">{formatAssetValue(asset.currentValue, asset.currency)}</span>
                      <div className="flex gap-1">
                        <button onClick={() => { setEditingAsset(asset); setIsModalOpen(true) }} className="p-2 text-gray-400 hover:text-blue-600">
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button onClick={() => { if (confirm('Delete this asset?')) deleteMutation.mutate(asset.id) }} className="p-2 text-gray-400 hover:text-red-600">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Liabilities List */}
      {showLiabilities && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="p-4 border-b border-gray-100">
            <h2 className="font-semibold text-gray-900">Liabilities</h2>
          </div>
          {liabilitiesList.length === 0 ? (
            <div className="text-center py-8 text-gray-500">No liabilities</div>
          ) : (
            <div className="divide-y divide-gray-100">
              {liabilitiesList.map((asset) => {
                const Icon = getCategoryIcon(asset.category)
                return (
                  <div key={asset.id} className="p-4 hover:bg-gray-50">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="h-10 w-10 rounded-full bg-red-100 flex items-center justify-center">
                          <Icon className="h-5 w-5 text-red-600" />
                        </div>
                        <div>
                          <p className="font-medium text-gray-900">{asset.name}</p>
                          <p className="text-sm text-gray-500">{asset.assetTypeName || getCategoryLabel(asset.category)}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <span className="font-semibold text-red-600">-{formatAssetValue(asset.currentValue, asset.currency)}</span>
                        <div className="flex gap-1">
                          <button onClick={() => { setEditingAsset(asset); setIsModalOpen(true) }} className="p-2 text-gray-400 hover:text-blue-600">
                            <Pencil className="h-4 w-4" />
                          </button>
                          <button onClick={() => { if (confirm('Delete?')) deleteMutation.mutate(asset.id) }} className="p-2 text-gray-400 hover:text-red-600">
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl w-full max-w-md max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between p-4 border-b border-gray-200 sticky top-0 bg-white">
              <h2 className="text-lg font-semibold">{editingAsset ? 'Edit Asset' : 'Add Asset'}</h2>
              <button onClick={() => { setIsModalOpen(false); setEditingAsset(null) }} className="p-2 text-gray-400 hover:text-gray-600">
                <X className="h-5 w-5" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="p-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input type="text" name="name" required defaultValue={editingAsset?.name || ''} className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500" placeholder="e.g., OCBC Savings" />
              </div>
              {!editingAsset && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Asset Type</label>
                    <select name="assetTypeId" required className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                      <option value="">Select an asset type...</option>
                      {Object.entries(groupedAssetTypes).map(([category, types]) => (
                        <optgroup key={category} label={getCategoryLabel(category as AssetCategory)}>
                          {(types as AssetType[]).map((type) => (
                            <option key={type.id} value={type.id}>{type.name}</option>
                          ))}
                        </optgroup>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
                    <select name="isLiability" className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                      <option value="false">Asset</option>
                      <option value="true">Liability</option>
                    </select>
                  </div>
                </>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Current Value</label>
                <input type="number" name="currentValue" step="0.01" min="0" required defaultValue={editingAsset?.currentValue || ''} className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500" placeholder="0.00" />
              </div>
              <div className="flex gap-3 pt-2">
                <button type="button" onClick={() => { setIsModalOpen(false); setEditingAsset(null) }} className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50">Cancel</button>
                <button type="submit" disabled={createMutation.isPending || updateMutation.isPending} className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{editingAsset ? 'Update' : 'Add'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

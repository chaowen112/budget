import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { useAuth } from '../store/AuthContext'
import { authApi, categoryApi } from '../api'
import { User, Globe, Key, Info, CheckCircle2, AlertCircle, ExternalLink, FolderTree, Plus, Pencil, Trash2, X } from 'lucide-react'
import { Button, FormField, Input, Select } from '../components/ui'

export default function Settings() {
  const { user, refreshUser } = useAuth()
  const queryClient = useQueryClient()
  const [name, setName] = useState(user?.name || '')
  const [baseCurrency, setBaseCurrency] = useState(user?.baseCurrency || 'SGD')
  const [successMessage, setSuccessMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [newCategoryName, setNewCategoryName] = useState('')
  const [newCategoryType, setNewCategoryType] = useState<'TRANSACTION_TYPE_EXPENSE' | 'TRANSACTION_TYPE_INCOME'>('TRANSACTION_TYPE_EXPENSE')
  const [editingCategoryId, setEditingCategoryId] = useState<string | null>(null)
  const [editingCategoryName, setEditingCategoryName] = useState('')

  const { data: categories = [] } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoryApi.list(),
  })

  const updateProfileMutation = useMutation({
    mutationFn: authApi.updateProfile,
    onSuccess: () => {
      refreshUser()
      queryClient.invalidateQueries()
      setSuccessMessage('Profile updated successfully')
      setTimeout(() => setSuccessMessage(''), 3000)
    },
    onError: () => {
      setErrorMessage('Failed to update profile')
      setTimeout(() => setErrorMessage(''), 3000)
    },
  })

  const createCategoryMutation = useMutation({
    mutationFn: categoryApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      queryClient.invalidateQueries({ queryKey: ['budgetStatuses'] })
      setNewCategoryName('')
      setSuccessMessage('Category created')
      setTimeout(() => setSuccessMessage(''), 3000)
    },
    onError: () => {
      setErrorMessage('Failed to create category')
      setTimeout(() => setErrorMessage(''), 3000)
    },
  })

  const updateCategoryMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => categoryApi.update(id, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      queryClient.invalidateQueries({ queryKey: ['budgetStatuses'] })
      setEditingCategoryId(null)
      setEditingCategoryName('')
      setSuccessMessage('Category updated')
      setTimeout(() => setSuccessMessage(''), 3000)
    },
    onError: () => {
      setErrorMessage('Failed to update category')
      setTimeout(() => setErrorMessage(''), 3000)
    },
  })

  const deleteCategoryMutation = useMutation({
    mutationFn: categoryApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      queryClient.invalidateQueries({ queryKey: ['budgetStatuses'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      setSuccessMessage('Category deleted')
      setTimeout(() => setSuccessMessage(''), 3000)
    },
    onError: () => {
      setErrorMessage('Failed to delete category. Remove linked transactions/budgets first.')
      setTimeout(() => setErrorMessage(''), 3500)
    },
  })

  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateProfileMutation.mutate({ name, baseCurrency })
  }

  const handleCreateCategory = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = newCategoryName.trim()
    if (!trimmed) return
    createCategoryMutation.mutate({
      name: trimmed,
      type: newCategoryType,
    })
  }

  const beginEditCategory = (id: string, currentName: string) => {
    setEditingCategoryId(id)
    setEditingCategoryName(currentName)
  }

  const saveEditCategory = (id: string) => {
    const trimmed = editingCategoryName.trim()
    if (!trimmed) return
    updateCategoryMutation.mutate({ id, name: trimmed })
  }

  const sectionClass =
    'bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 shadow-sm overflow-hidden'
  const sectionHeaderClass =
    'flex items-center gap-2.5 px-5 py-3.5 border-b border-zinc-100 dark:border-zinc-800'
  const expenseCategories = categories.filter((c) => c.type === 'TRANSACTION_TYPE_EXPENSE')
  const incomeCategories = categories.filter((c) => c.type === 'TRANSACTION_TYPE_INCOME')

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">Settings</h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">Manage your account preferences</p>
      </div>

      {/* Toast Messages */}
      {successMessage && (
        <div className="flex items-center gap-2.5 px-4 py-3 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 text-sm text-emerald-700 dark:text-emerald-400">
          <CheckCircle2 className="h-4 w-4 flex-shrink-0" />
          {successMessage}
        </div>
      )}
      {errorMessage && (
        <div className="flex items-center gap-2.5 px-4 py-3 rounded-xl bg-red-50 dark:bg-red-500/10 border border-red-200 dark:border-red-500/20 text-sm text-red-700 dark:text-red-400">
          <AlertCircle className="h-4 w-4 flex-shrink-0" />
          {errorMessage}
        </div>
      )}

      {/* Profile */}
      <div className={sectionClass}>
        <div className={sectionHeaderClass}>
          <div className="h-7 w-7 rounded-lg bg-violet-50 dark:bg-violet-500/10 flex items-center justify-center">
            <User className="h-3.5 w-3.5 text-violet-600 dark:text-violet-400" />
          </div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">Profile</h2>
        </div>
        <form onSubmit={handleProfileSubmit} className="p-5 space-y-4">
          <FormField label="Email" hint="Cannot be changed">
            <Input
              type="email"
              value={user?.email || ''}
              disabled
            />
          </FormField>
          <FormField label="Name">
            <Input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Your name"
            />
          </FormField>
          <Button
            type="submit"
            loading={updateProfileMutation.isPending}
            size="sm"
          >
            Save Changes
          </Button>
        </form>
      </div>

      {/* Currency */}
      <div className={sectionClass}>
        <div className={sectionHeaderClass}>
          <div className="h-7 w-7 rounded-lg bg-amber-50 dark:bg-amber-500/10 flex items-center justify-center">
            <Globe className="h-3.5 w-3.5 text-amber-600 dark:text-amber-400" />
          </div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">Currency</h2>
        </div>
        <div className="p-5 space-y-4">
          <FormField label="Base Currency" hint="All amounts display in this currency">
            <Select
              value={baseCurrency}
              onChange={(e) => setBaseCurrency(e.target.value)}
            >
              <option value="SGD">SGD — Singapore Dollar</option>
              <option value="USD">USD — US Dollar</option>
              <option value="EUR">EUR — Euro</option>
              <option value="GBP">GBP — British Pound</option>
              <option value="JPY">JPY — Japanese Yen</option>
              <option value="CNY">CNY — Chinese Yuan</option>
              <option value="MYR">MYR — Malaysian Ringgit</option>
              <option value="IDR">IDR — Indonesian Rupiah</option>
              <option value="AUD">AUD — Australian Dollar</option>
              <option value="HKD">HKD — Hong Kong Dollar</option>
              <option value="THB">THB — Thai Baht</option>
            </Select>
          </FormField>
          <Button
            size="sm"
            onClick={() => updateProfileMutation.mutate({ baseCurrency })}
            disabled={updateProfileMutation.isPending || baseCurrency === user?.baseCurrency}
            loading={updateProfileMutation.isPending}
          >
            Update Currency
          </Button>
        </div>
      </div>

      {/* Categories */}
      <div className={sectionClass}>
        <div className={sectionHeaderClass}>
          <div className="h-7 w-7 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 flex items-center justify-center">
            <FolderTree className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
          </div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">Categories</h2>
        </div>
        <div className="p-5 space-y-5">
          <form onSubmit={handleCreateCategory} className="grid grid-cols-1 sm:grid-cols-4 gap-2">
            <Input
              value={newCategoryName}
              onChange={(e) => setNewCategoryName(e.target.value)}
              placeholder="New category name"
              className="sm:col-span-2"
            />
            <Select
              value={newCategoryType}
              onChange={(e) => setNewCategoryType(e.target.value as 'TRANSACTION_TYPE_EXPENSE' | 'TRANSACTION_TYPE_INCOME')}
            >
              <option value="TRANSACTION_TYPE_EXPENSE">Expense</option>
              <option value="TRANSACTION_TYPE_INCOME">Income</option>
            </Select>
            <Button
              type="submit"
              icon={<Plus className="h-4 w-4" />}
              loading={createCategoryMutation.isPending}
            >
              Add
            </Button>
          </form>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[
              { label: 'Expense Categories', items: expenseCategories },
              { label: 'Income Categories', items: incomeCategories },
            ].map((group) => (
              <div key={group.label} className="rounded-xl border border-zinc-200 dark:border-zinc-800 overflow-hidden">
                <div className="px-3 py-2 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/50">
                  <p className="text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">{group.label}</p>
                </div>
                {group.items.length === 0 ? (
                  <div className="px-3 py-4 text-xs text-zinc-500 dark:text-zinc-400">No categories yet.</div>
                ) : (
                  <div className="divide-y divide-zinc-100 dark:divide-zinc-800">
                    {group.items.map((cat) => (
                      <div key={cat.id} className="px-3 py-2.5 flex items-center gap-2">
                        {editingCategoryId === cat.id ? (
                          <>
                            <Input
                              value={editingCategoryName}
                              onChange={(e) => setEditingCategoryName(e.target.value)}
                              className="h-8"
                            />
                            <Button
                              size="sm"
                              onClick={() => saveEditCategory(cat.id)}
                              loading={updateCategoryMutation.isPending}
                            >
                              Save
                            </Button>
                            <button
                              type="button"
                              onClick={() => {
                                setEditingCategoryId(null)
                                setEditingCategoryName('')
                              }}
                              className="h-8 w-8 rounded-lg border border-zinc-200 dark:border-zinc-700 text-zinc-500 dark:text-zinc-400 flex items-center justify-center hover:bg-zinc-100 dark:hover:bg-zinc-800"
                            >
                              <X className="h-3.5 w-3.5" />
                            </button>
                          </>
                        ) : (
                          <>
                            <span className="text-sm text-zinc-800 dark:text-zinc-200 flex-1">{cat.name}</span>
                            <button
                              type="button"
                              onClick={() => beginEditCategory(cat.id, cat.name)}
                              className="h-7 w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                            >
                              <Pencil className="h-3.5 w-3.5" />
                            </button>
                            <button
                              type="button"
                              onClick={() => {
                                if (confirm('Delete this category? This fails if linked data exists.')) {
                                  deleteCategoryMutation.mutate(cat.id)
                                }
                              }}
                              className="h-7 w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          </>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>

          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            Categories are now fully user-defined. Create what you need here, then use them in
            <Link to="/transactions" className="text-violet-600 dark:text-violet-400 hover:underline"> transactions</Link>
            {' '}and
            <Link to="/budgets" className="text-violet-600 dark:text-violet-400 hover:underline"> budgets</Link>.
          </p>
        </div>
      </div>

      {/* API Documentation */}
      <div className={sectionClass}>
        <div className={sectionHeaderClass}>
          <div className="h-7 w-7 rounded-lg bg-blue-50 dark:bg-blue-500/10 flex items-center justify-center">
            <Key className="h-3.5 w-3.5 text-blue-600 dark:text-blue-400" />
          </div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">API Documentation</h2>
        </div>
        <div className="p-5">
          <p className="text-sm text-zinc-600 dark:text-zinc-400 mb-4">
            Access the API documentation to integrate with external tools or build your own applications.
          </p>
          <a
            href="/swagger/"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 h-9 px-4 text-sm font-medium rounded-xl
              bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-200
              border border-zinc-200 dark:border-zinc-700
              hover:bg-zinc-50 dark:hover:bg-zinc-800
              transition-colors duration-150"
          >
            Open Swagger UI
            <ExternalLink className="h-3.5 w-3.5 text-zinc-400" />
          </a>
        </div>
      </div>

      {/* Account Info */}
      <div className={sectionClass}>
        <div className={sectionHeaderClass}>
          <div className="h-7 w-7 rounded-lg bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center">
            <Info className="h-3.5 w-3.5 text-zinc-500 dark:text-zinc-400" />
          </div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50">Account Information</h2>
        </div>
        <div className="p-5 space-y-3">
          <div className="flex items-center justify-between text-sm">
            <span className="text-zinc-500 dark:text-zinc-400">User ID</span>
            <span className="font-mono text-xs text-zinc-700 dark:text-zinc-300 bg-zinc-100 dark:bg-zinc-800 px-2 py-0.5 rounded-md">
              {user?.id}
            </span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-zinc-500 dark:text-zinc-400">Member since</span>
            <span className="text-zinc-700 dark:text-zinc-300">
              {user?.createdAt ? new Date(user.createdAt).toLocaleDateString() : '—'}
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}

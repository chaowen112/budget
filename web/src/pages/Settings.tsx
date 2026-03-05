import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '../store/AuthContext'
import { authApi } from '../api'
import { User, Key, Globe } from 'lucide-react'

export default function Settings() {
  const { user, refreshUser } = useAuth()
  const queryClient = useQueryClient()
  const [name, setName] = useState(user?.name || '')
  const [baseCurrency, setBaseCurrency] = useState(user?.baseCurrency || 'SGD')
  const [successMessage, setSuccessMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateProfileMutation.mutate({ name, baseCurrency })
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
        <p className="text-gray-500">Manage your account preferences</p>
      </div>

      {successMessage && (
        <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg">
          {successMessage}
        </div>
      )}

      {errorMessage && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
          {errorMessage}
        </div>
      )}

      {/* Profile Settings */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100">
        <div className="p-4 border-b border-gray-100 flex items-center gap-2">
          <User className="h-5 w-5 text-gray-400" />
          <h2 className="font-semibold text-gray-900">Profile</h2>
        </div>
        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input
              type="email"
              value={user?.email || ''}
              disabled
              className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-gray-50 text-gray-500"
            />
            <p className="mt-1 text-xs text-gray-500">Email cannot be changed</p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
          <button
            type="submit"
            disabled={updateProfileMutation.isPending}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {updateProfileMutation.isPending ? 'Saving...' : 'Save Changes'}
          </button>
        </form>
      </div>

      {/* Currency Settings */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100">
        <div className="p-4 border-b border-gray-100 flex items-center gap-2">
          <Globe className="h-5 w-5 text-gray-400" />
          <h2 className="font-semibold text-gray-900">Currency</h2>
        </div>
        <div className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Base Currency</label>
            <select
              value={baseCurrency}
              onChange={(e) => setBaseCurrency(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="SGD">SGD - Singapore Dollar</option>
              <option value="USD">USD - US Dollar</option>
              <option value="EUR">EUR - Euro</option>
              <option value="GBP">GBP - British Pound</option>
              <option value="JPY">JPY - Japanese Yen</option>
              <option value="CNY">CNY - Chinese Yuan</option>
              <option value="MYR">MYR - Malaysian Ringgit</option>
              <option value="AUD">AUD - Australian Dollar</option>
              <option value="HKD">HKD - Hong Kong Dollar</option>
              <option value="THB">THB - Thai Baht</option>
            </select>
            <p className="mt-1 text-xs text-gray-500">
              All amounts will be displayed in this currency
            </p>
          </div>
          <button
            onClick={() => updateProfileMutation.mutate({ baseCurrency })}
            disabled={updateProfileMutation.isPending || baseCurrency === user?.baseCurrency}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Update Currency
          </button>
        </div>
      </div>

      {/* API Documentation Link */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100">
        <div className="p-4 border-b border-gray-100 flex items-center gap-2">
          <Key className="h-5 w-5 text-gray-400" />
          <h2 className="font-semibold text-gray-900">API Documentation</h2>
        </div>
        <div className="p-4">
          <p className="text-sm text-gray-600 mb-4">
            Access the API documentation to integrate with external tools or build your own applications.
          </p>
          <a
            href="/swagger/"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
          >
            Open Swagger UI
          </a>
        </div>
      </div>

      {/* Account Info */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100">
        <div className="p-4 border-b border-gray-100">
          <h2 className="font-semibold text-gray-900">Account Information</h2>
        </div>
        <div className="p-4 space-y-3 text-sm">
          <div className="flex justify-between">
            <span className="text-gray-500">User ID</span>
            <span className="text-gray-900 font-mono text-xs">{user?.id}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Member since</span>
            <span className="text-gray-900">
              {user?.createdAt ? new Date(user.createdAt).toLocaleDateString() : '-'}
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}

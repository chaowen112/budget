import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '../store/AuthContext'
import { useCurrency, DISPLAY_CURRENCIES, type DisplayCurrency } from '../store/CurrencyContext'
import {
  LayoutDashboard,
  Receipt,
  PiggyBank,
  Target,
  Wallet,
  BarChart3,
  Settings,
  LogOut,
  Menu,
  X,
  TrendingUp,
  RefreshCw,
} from 'lucide-react'
import { useState } from 'react'

const navigation = [
  { name: 'Dashboard',    href: '/',             icon: LayoutDashboard },
  { name: 'Transactions', href: '/transactions', icon: Receipt },
  { name: 'Budgets',      href: '/budgets',      icon: PiggyBank },
  { name: 'Goals',        href: '/goals',        icon: Target },
  { name: 'Assets',       href: '/assets',       icon: Wallet },
  { name: 'Reports',      href: '/reports',      icon: BarChart3 },
  { name: 'Settings',     href: '/settings',     icon: Settings },
]

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const { displayCurrency, setDisplayCurrency, isLoadingRates } = useCurrency()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const initials = user?.name
    ? user.name.split(' ').map((n) => n[0]).join('').slice(0, 2).toUpperCase()
    : 'U'

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-950">

      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/40 backdrop-blur-sm z-40 lg:hidden transition-opacity duration-200"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* ── Sidebar ────────────────────────────────────────── */}
      <aside
        className={`
          fixed top-0 left-0 z-50 h-full w-60
          bg-white dark:bg-zinc-900
          border-r border-zinc-200 dark:border-zinc-800
          flex flex-col
          transition-transform duration-200 ease-in-out
          lg:translate-x-0
          ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
      >
        {/* Logo */}
        <div className="flex items-center justify-between h-14 px-5 border-b border-zinc-200 dark:border-zinc-800 flex-shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="h-7 w-7 rounded-lg bg-violet-600 flex items-center justify-center flex-shrink-0">
              <TrendingUp className="h-4 w-4 text-white" />
            </div>
            <span className="text-sm font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
              Wealthly
            </span>
          </div>
          <button
            className="lg:hidden h-7 w-7 flex items-center justify-center rounded-lg text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
            onClick={() => setSidebarOpen(false)}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
          {navigation.map((item) => (
            <NavLink
              key={item.name}
              to={item.href}
              end={item.href === '/'}
              onClick={() => setSidebarOpen(false)}
              className={({ isActive }) =>
                `group flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium transition-all duration-150 ${
                  isActive
                    ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-50'
                    : 'text-zinc-500 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800/60 hover:text-zinc-900 dark:hover:text-zinc-100'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <item.icon
                    className={`h-[18px] w-[18px] flex-shrink-0 transition-colors ${
                      isActive ? 'text-violet-600 dark:text-violet-400' : 'text-zinc-400 dark:text-zinc-500 group-hover:text-zinc-600 dark:group-hover:text-zinc-300'
                    }`}
                  />
                  {item.name}
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* User footer */}
        <div className="p-3 border-t border-zinc-200 dark:border-zinc-800 flex-shrink-0 space-y-1">
          <div className="flex items-center gap-3 px-3 py-2 rounded-xl">
            <div className="h-7 w-7 rounded-full bg-violet-100 dark:bg-violet-500/20 flex items-center justify-center flex-shrink-0">
              <span className="text-xs font-semibold text-violet-700 dark:text-violet-300">
                {initials}
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-xs font-medium text-zinc-900 dark:text-zinc-100 truncate">{user?.name}</p>
              <p className="text-xs text-zinc-400 dark:text-zinc-500 truncate">{user?.email}</p>
            </div>
          </div>
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 w-full px-3 py-2 rounded-xl text-sm text-zinc-500 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-red-600 dark:hover:text-red-400 transition-all duration-150"
          >
            <LogOut className="h-[18px] w-[18px] flex-shrink-0" />
            Sign out
          </button>
        </div>
      </aside>

      {/* ── Main area ──────────────────────────────────────── */}
      <div className="lg:pl-60 flex flex-col min-h-screen">
        {/* Top bar */}
        <header className="sticky top-0 z-30 h-14 flex items-center justify-between px-4 lg:px-6 bg-zinc-50/80 dark:bg-zinc-950/80 backdrop-blur-md border-b border-zinc-200 dark:border-zinc-800">
          <button
            className="lg:hidden h-8 w-8 flex items-center justify-center rounded-lg text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </button>
          <div className="flex-1 lg:flex-none" />
          <div className="flex items-center gap-3">
            {/* Currency toggle */}
            <div className="flex items-center gap-px p-0.5 rounded-xl bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700">
              {DISPLAY_CURRENCIES.map((c: DisplayCurrency) => (
                <button
                  key={c}
                  onClick={() => setDisplayCurrency(c)}
                  className={`
                    h-7 px-2.5 rounded-lg text-xs font-semibold transition-all duration-150
                    ${displayCurrency === c
                      ? 'bg-violet-600 text-white shadow-sm'
                      : 'text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200'
                    }
                  `}
                >
                  {c}
                </button>
              ))}
              {isLoadingRates && (
                <span className="ml-1 mr-0.5">
                  <RefreshCw className="h-3 w-3 text-zinc-400 dark:text-zinc-500 animate-spin" />
                </span>
              )}
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 p-4 lg:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

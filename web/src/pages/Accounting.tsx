import { useQuery } from '@tanstack/react-query'
import { accountingApi } from '../api'
import { useCurrency } from '../store/CurrencyContext'
import type { LedgerAccount, JournalEntry } from '../types'
import { formatDate } from '../lib/utils'

const accountTypeOrder: Array<LedgerAccount['accountType']> = ['asset', 'liability', 'equity', 'income', 'expense']

export default function Accounting() {
  const { formatConverted } = useCurrency()

  const { data: accounts, isLoading: isLoadingAccounts } = useQuery({
    queryKey: ['ledgerAccounts'],
    queryFn: accountingApi.listAccounts,
  })

  const { data: entries, isLoading: isLoadingEntries } = useQuery({
    queryKey: ['journalEntries'],
    queryFn: () => accountingApi.listJournal(100),
  })

  const isLoading = isLoadingAccounts || isLoadingEntries

  const grouped = (accounts || []).reduce<Record<string, LedgerAccount[]>>((acc, item) => {
    if (!acc[item.accountType]) {
      acc[item.accountType] = []
    }
    acc[item.accountType].push(item)
    return acc
  }, {})

  const typeLabel: Record<LedgerAccount['accountType'], string> = {
    asset: 'Assets',
    liability: 'Liabilities',
    equity: 'Equity',
    income: 'Income',
    expense: 'Expenses',
  }

  const moneyText = (amount: string, currency: string) => {
    return formatConverted({ amount, currency })
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">Accounting Ledger</h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">
          Double-entry accounts and journal entries
        </p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <svg className="animate-spin h-8 w-8 text-violet-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            {accountTypeOrder.map((type) => (
              <div key={type} className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm">
                <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50 mb-3">{typeLabel[type]}</h2>
                <div className="space-y-2 max-h-64 overflow-y-auto pr-1">
                  {(grouped[type] || []).length > 0 ? (
                    grouped[type].map((account) => (
                      <div
                        key={account.id}
                        className="flex items-center justify-between rounded-xl border border-zinc-100 dark:border-zinc-800 px-3 py-2 text-sm"
                      >
                        <span className="text-zinc-700 dark:text-zinc-300 truncate pr-2">{account.name}</span>
                        <span className="font-semibold tabular-nums text-zinc-900 dark:text-zinc-100 whitespace-nowrap">
                          {moneyText(account.balance, account.currency)}
                        </span>
                      </div>
                    ))
                  ) : (
                    <p className="text-sm text-zinc-400 dark:text-zinc-500">No {typeLabel[type].toLowerCase()} accounts</p>
                  )}
                </div>
              </div>
            ))}
          </div>

          <div className="bg-white dark:bg-zinc-900 rounded-2xl border border-zinc-200 dark:border-zinc-800 p-5 shadow-sm">
            <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50 mb-4">Recent Journal Entries</h2>
            <div className="space-y-3 max-h-[32rem] overflow-y-auto pr-1">
              {(entries || []).length > 0 ? (
                (entries || []).map((entry: JournalEntry) => (
                  <div key={entry.id} className="rounded-xl border border-zinc-100 dark:border-zinc-800 p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2 mb-2">
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">{entry.description || 'Journal Entry'}</p>
                        <p className="text-xs text-zinc-500 dark:text-zinc-400">
                          {formatDate(entry.entryDate)} · {entry.source}
                        </p>
                      </div>
                      <span className="text-[11px] px-2 py-1 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400">
                        {entry.referenceType || 'manual'}
                      </span>
                    </div>
                    <div className="space-y-1.5">
                      {entry.lines.map((line) => (
                        <div key={line.id} className="grid grid-cols-3 gap-2 text-xs">
                          <span className="text-zinc-600 dark:text-zinc-400 truncate">{line.accountName}</span>
                          <span className="text-right text-emerald-600 dark:text-emerald-400 tabular-nums">
                            {line.debit !== '0' ? line.debit : '-'}
                          </span>
                          <span className="text-right text-red-500 dark:text-red-400 tabular-nums">
                            {line.credit !== '0' ? line.credit : '-'}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-sm text-zinc-400 dark:text-zinc-500">No journal entries yet</p>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  )
}

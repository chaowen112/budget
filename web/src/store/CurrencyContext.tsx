import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useMemo,
  type ReactNode,
} from 'react'
import { currencyApi } from '../api'
import type { Money } from '../types'

// The four currencies supported by the display toggle
export const DISPLAY_CURRENCIES = ['SGD', 'TWD', 'CNY', 'USD'] as const
export type DisplayCurrency = (typeof DISPLAY_CURRENCIES)[number]

interface CurrencyContextType {
  /** Currently selected display currency */
  displayCurrency: DisplayCurrency
  /** Change the display currency — fetches rates if not cached */
  setDisplayCurrency: (currency: DisplayCurrency) => void
  /**
   * Convert a Money value to the current display currency and format it.
   * Falls back to the raw formatMoney if rate is unavailable.
   */
  formatConverted: (money: Money | undefined | null) => string
  /**
   * Whether exchange rates are currently being fetched.
   */
  isLoadingRates: boolean
}

const CurrencyContext = createContext<CurrencyContextType | undefined>(undefined)

// ----- helpers ---------------------------------------------------------------

/** Map: "SGD→USD" → rate number */
type RateCache = Map<string, number>

function rateKey(from: string, to: string) {
  return `${from}→${to}`
}

function formatWithCurrency(amount: number, currency: string): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

// ----- provider --------------------------------------------------------------

export function CurrencyProvider({ children }: { children: ReactNode }) {
  const [displayCurrency, setDisplayCurrencyState] = useState<DisplayCurrency>(() => {
    const stored = localStorage.getItem('displayCurrency')
    return (DISPLAY_CURRENCIES.includes(stored as DisplayCurrency)
      ? (stored as DisplayCurrency)
      : 'SGD')
  })

  // Cached rates keyed by "FROM→TO"
  const [rateCache, setRateCache] = useState<RateCache>(new Map())
  const [isLoadingRates, setIsLoadingRates] = useState(false)

  /**
   * Fetch all rates needed to convert from any of the 4 supported currencies
   * to `targetCurrency`. We fetch each pair individually from the backend
   * (which does DB-first with API fallback caching).
   */
  const fetchRatesForCurrency = useCallback(
    async (target: DisplayCurrency) => {
      if (target === 'SGD') {
        // SGD→SGD is always 1; SGD is our base so we also need the inverse pairs
        // Just ensure 1:1 is cached and skip fetching.
        setRateCache((prev) => {
          const next = new Map(prev)
          for (const c of DISPLAY_CURRENCIES) {
            next.set(rateKey(c, c), 1)
          }
          next.set(rateKey('SGD', 'SGD'), 1)
          return next
        })
        return
      }

      setIsLoadingRates(true)
      try {
        // Fetch rates from every other display currency into `target`
        const sourceCurrencies = DISPLAY_CURRENCIES.filter((c) => c !== target)
        const pairs: [string, string][] = [
          ...sourceCurrencies.map((c): [string, string] => [c, target]),
          [target, target], // identity
        ]

        const results = await Promise.allSettled(
          pairs.map(([from, to]) =>
            from === to
              ? Promise.resolve({ fromCurrency: from, toCurrency: to, rate: 1, updatedAt: '' })
              : currencyApi.getExchangeRate(from, to)
          )
        )

        setRateCache((prev) => {
          const next = new Map(prev)
          results.forEach((result, i) => {
            const [from, to] = pairs[i]
            if (result.status === 'fulfilled') {
              next.set(rateKey(from, to), result.value.rate)
            }
            // If fetch failed, leave existing cached rate (if any)
          })
          return next
        })
      } catch {
        // Silently continue with whatever cached rates exist
      } finally {
        setIsLoadingRates(false)
      }
    },
    []
  )

  // On mount and whenever displayCurrency changes, ensure rates are loaded
  useEffect(() => {
    fetchRatesForCurrency(displayCurrency)
  }, [displayCurrency, fetchRatesForCurrency])

  const setDisplayCurrency = useCallback((currency: DisplayCurrency) => {
    localStorage.setItem('displayCurrency', currency)
    setDisplayCurrencyState(currency)
  }, [])

  const formatConverted = useCallback(
    (money: Money | undefined | null): string => {
      if (!money) return formatWithCurrency(0, displayCurrency)

      const amount = parseFloat(money.amount || '0')
      const fromCurrency = money.currency || 'SGD'

      if (fromCurrency === displayCurrency) {
        return formatWithCurrency(amount, displayCurrency)
      }

      const key = rateKey(fromCurrency, displayCurrency)
      const rate = rateCache.get(key)

      if (rate !== undefined) {
        return formatWithCurrency(amount * rate, displayCurrency)
      }

      // Rate not yet cached — show original value with original currency
      return formatWithCurrency(amount, fromCurrency)
    },
    [displayCurrency, rateCache]
  )

  const value = useMemo(
    () => ({ displayCurrency, setDisplayCurrency, formatConverted, isLoadingRates }),
    [displayCurrency, setDisplayCurrency, formatConverted, isLoadingRates]
  )

  return <CurrencyContext.Provider value={value}>{children}</CurrencyContext.Provider>
}

export function useCurrency() {
  const ctx = useContext(CurrencyContext)
  if (!ctx) throw new Error('useCurrency must be used inside CurrencyProvider')
  return ctx
}

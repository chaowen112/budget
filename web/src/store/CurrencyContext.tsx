import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useRef,
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
type RateCacheEntry = { rate: number; fetchedAt: number }
type RateCache = Map<string, RateCacheEntry>

const RATE_CACHE_KEY = 'exchangeRateCache'
const RATE_CACHE_TTL_MS = 1000 * 60 * 60 * 12 // 12 hours

function rateKey(from: string, to: string) {
  return `${from}→${to}`
}

function loadRateCache(): RateCache {
  try {
    const raw = localStorage.getItem(RATE_CACHE_KEY)
    if (!raw) return new Map()

    const parsed = JSON.parse(raw) as Record<string, RateCacheEntry>
    const now = Date.now()
    const entries = Object.entries(parsed).filter(([, value]) => {
      if (!value || typeof value.rate !== 'number' || typeof value.fetchedAt !== 'number') {
        return false
      }
      return now - value.fetchedAt <= RATE_CACHE_TTL_MS
    })
    return new Map(entries)
  } catch {
    return new Map()
  }
}

function persistRateCache(cache: RateCache) {
  const payload: Record<string, RateCacheEntry> = {}
  cache.forEach((value, key) => {
    payload[key] = value
  })
  localStorage.setItem(RATE_CACHE_KEY, JSON.stringify(payload))
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
  const [rateCache, setRateCache] = useState<RateCache>(() => loadRateCache())
  const [isLoadingRates, setIsLoadingRates] = useState(false)
  const rateCacheRef = useRef<RateCache>(rateCache)

  useEffect(() => {
    rateCacheRef.current = rateCache
    persistRateCache(rateCache)
  }, [rateCache])

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
          const now = Date.now()
          let changed = false
          for (const c of DISPLAY_CURRENCIES) {
            const key = rateKey(c, c)
            const existing = next.get(key)
            if (!existing || existing.rate !== 1) {
              next.set(key, { rate: 1, fetchedAt: now })
              changed = true
            }
          }
          return changed ? next : prev
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

        const now = Date.now()
        const freshPairs = pairs.filter(([from, to]) => {
          const cached = rateCacheRef.current.get(rateKey(from, to))
          if (!cached) return true
          return now - cached.fetchedAt > RATE_CACHE_TTL_MS
        })

        if (freshPairs.length === 0) {
          return
        }

        const results = await Promise.allSettled(
          freshPairs.map(([from, to]) =>
            from === to
              ? Promise.resolve({ fromCurrency: from, toCurrency: to, rate: 1, updatedAt: '' })
              : currencyApi.getExchangeRate(from, to)
          )
        )

        setRateCache((prev) => {
          const next = new Map(prev)
          const fetchedAt = Date.now()
          results.forEach((result, i) => {
            const [from, to] = freshPairs[i]
            if (result.status === 'fulfilled') {
              next.set(rateKey(from, to), { rate: result.value.rate, fetchedAt })
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
      const rate = rateCache.get(key)?.rate

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

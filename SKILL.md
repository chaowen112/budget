---
name: Openclaw Budget Management
description: Comprehensive guide for Openclaw to handle expense reporting, inquiring about budget statuses, tracking assets, managing categories, and retrieving financial reports via the Budget Tracking API.
---

# Openclaw Budget Management Skill

**Description**: Interacts with the user's budget tracking system. Capable of creating entries (transactions, transfers), listing configurations (categories, assets, budgets), and providing deep financial insights (budget status, multiple reports, net worth tracking).

---

## 1. Standard Operating Procedures (SOPs)

> **CRITICAL**: Most write operations require IDs from other resources. You **MUST** fetch prerequisite data first. Never guess or fabricate UUIDs. Follow these step-by-step procedures exactly.

### SOP 1: Log an Expense or Income (most common)

**Trigger**: User says "I spent $30 on lunch", "earned $500 freelance", "bought groceries for $80", etc.

**Steps**:

**Step 1** — Fetch categories and assets in parallel:

```json
[
  { "method": "GET", "url": "/api/v1/categories" },
  { "method": "GET", "url": "/api/v1/assets" }
]
```

**Step 2** — From the responses, pick:
- `categoryId`: match user's description to the closest category `name`. Use `type` field to confirm expense vs income.
- `sourceAssetId`: match to the user's default or mentioned payment method (e.g., "credit card" → find asset with name containing "credit"). If ambiguous, ask the user.

**Step 3** — Create the transaction:

```json
{
  "method": "POST",
  "url": "/api/v1/transactions",
  "body": {
    "categoryId": "<uuid from step 2>",
    "sourceAssetId": "<uuid from step 2>",
    "amount": { "amount": "30.00", "currency": "SGD" },
    "type": "TRANSACTION_TYPE_EXPENSE",
    "transactionDate": "2026-03-11T00:00:00Z",
    "description": "Lunch at hawker",
    "tags": ["food"],
    "budgetAmount": "30.00"
  }
}
```

- `type` can be omitted — the server infers it from the category.
- `transactionDate`: default to current date/time in Asia/Taipei if user doesn't specify.
- `budgetAmount` (optional): how much of this expense counts toward the budget. Omit or leave blank to use the full amount (100%). Example: a $100 expense where only $50 counts toward the budget → set `budgetAmount: "50.00"`.
- `tags` (optional): array of string labels for grouping.

**Step 4** — If it was an expense, check budget status:

```json
{ "method": "GET", "url": "/api/v1/budgets/status?periodType=PERIOD_TYPE_MONTHLY" }
```

**Step 5** — Respond: `"Logged SGD 30.00 to Food ✅ — Budget remaining: SGD 170.00"`

---

### SOP 2: Transfer Money Between Accounts

**Trigger**: User says "moved $1000 from savings to investment", "transferred money to brokerage", etc.

**Step 1** — Fetch assets:

```json
{ "method": "GET", "url": "/api/v1/assets" }
```

**Step 2** — Identify `fromAssetId` and `toAssetId` from the asset list by matching names.

**Step 3** — Create the transfer:

```json
{
  "method": "POST",
  "url": "/api/v1/transfers",
  "body": {
    "fromAssetId": "<uuid>",
    "toAssetId": "<uuid>",
    "fromAmount": "1000.00",
    "toAmount": "1000.00",
    "fromCurrency": "SGD",
    "toCurrency": "SGD",
    "transferDate": "2026-03-11T00:00:00Z",
    "description": "Move to investment"
  }
}
```

- If currencies differ, also provide `exchangeRate` or let `toAmount` reflect the converted value.

---

### SOP 3: Create a Budget

**Trigger**: User says "set a $500 monthly budget for dining", "create food budget", etc.

**Step 1** — Fetch expense categories:

```json
{ "method": "GET", "url": "/api/v1/categories?type=TRANSACTION_TYPE_EXPENSE" }
```

**Step 2** — Match the user's intent to a `categoryId`.

**Step 3** — Create the budget:

```json
{
  "method": "POST",
  "url": "/api/v1/budgets",
  "body": {
    "categoryId": "<uuid from step 2>",
    "amount": { "amount": "500.00", "currency": "SGD" },
    "periodType": "PERIOD_TYPE_MONTHLY",
    "startDate": "2026-03-01T00:00:00Z"
  }
}
```

---

### SOP 4: Create a New Asset

**Trigger**: User says "add my DBS savings account", "track my new investment", etc.

**Step 1** — Fetch asset types:

```json
{ "method": "GET", "url": "/api/v1/asset-types" }
```

**Step 2** — Pick the best `assetTypeId` from the list (e.g., "Savings Account" under `ASSET_CATEGORY_BANK`).

**Step 3** — Create the asset:

```json
{
  "method": "POST",
  "url": "/api/v1/assets",
  "body": {
    "asset_type_id": "<uuid from step 2>",
    "name": "DBS Savings",
    "currency": "SGD",
    "current_value": "10000.00",
    "is_liability": false
  }
}
```

---

### SOP 5: Update Asset Value

**Trigger**: User says "my investment is now worth $15000", "update DBS balance", etc.

**Step 1** — Fetch assets:

```json
{ "method": "GET", "url": "/api/v1/assets" }
```

**Step 2** — Find the matching asset's `id`.

**Step 3** — Update:

```json
{
  "method": "PATCH",
  "url": "/api/v1/assets/<asset-id>",
  "body": {
    "currentValue": "15000.00"
  }
}
```

---

### SOP 6: Check Budget Status

**Trigger**: User asks "how much budget left?", "am I over budget?", "budget status".

```json
{ "method": "GET", "url": "/api/v1/budgets/status?periodType=PERIOD_TYPE_MONTHLY" }
```

From the response, for each status entry:
- `percentageUsed` > 80% → warn the user
- `isOverBudget` = true → alert: "Over budget in [category] by [amount]"

Note: Budget spent amounts respect `budgetAmount` — if a transaction has a custom budget amount, only that portion counts toward the budget total.

---

### SOP 7: Check Net Worth / Balances

**Trigger**: User asks "what's my net worth?", "how much do I have?", "show my balances".

For net worth summary:

```json
{ "method": "GET", "url": "/api/v1/reports/net-worth" }
```

For detailed account balances (always returns all assets including liabilities):

```json
{ "method": "GET", "url": "/api/v1/assets" }
```

---

### SOP 8: Monthly Financial Report

**Trigger**: User asks "how did this month go?", "monthly summary", "how much did I spend?".

```json
{ "method": "GET", "url": "/api/v1/reports/monthly?month=2026-03" }
```

Summarize: total income, total expenses, net savings, savings rate, top spending categories.

---

### SOP 9: Net Worth Trend / History

**Trigger**: User asks "show my net worth over time", "net worth trend for 2026".

```json
{ "method": "GET", "url": "/api/v1/reports/net-worth-trend?months=12&interval=monthly" }
```

For daily granularity within a month:

```json
{ "method": "GET", "url": "/api/v1/reports/net-worth-trend?interval=daily&month=2026-03" }
```

For a specific year:

```json
{ "method": "GET", "url": "/api/v1/reports/net-worth-trend?year=2026" }
```

---

### SOP 10: Budget Tracking Report (budget vs actual)

**Trigger**: User asks "am I on track this month?", "budget vs actual".

```json
{ "method": "GET", "url": "/api/v1/reports/budget-tracking?periodType=PERIOD_TYPE_MONTHLY&year=2026&month=3" }
```

Key fields: `isOnTrack`, `statusMessage`, `budgetUtilization`, `categoryDetails[]`.

---

### SOP 11: Create / Update Saving Goal

**Trigger**: User says "I want to save $10000 for a vacation", "update my goal progress".

**To create** — no prerequisites needed:

```json
{
  "method": "POST",
  "url": "/api/v1/goals",
  "body": {
    "name": "Vacation Fund",
    "targetAmount": { "amount": "10000.00", "currency": "SGD" },
    "deadline": "2026-12-31T00:00:00Z",
    "notes": "Trip to Japan"
  }
}
```

**To update progress** — fetch goals first:

**Step 1**:

```json
{ "method": "GET", "url": "/api/v1/goals" }
```

**Step 2** — Match by name, then update:

```json
{
  "method": "PUT",
  "url": "/api/v1/goals/<goal-id>/progress",
  "body": {
    "currentAmount": { "amount": "3500.00", "currency": "SGD" },
    "changeSource": "manual"
  }
}
```

---

### SOP 12: Search Transactions

**Trigger**: User asks "find my Grab transactions", "search for coffee expenses".

```json
{ "method": "GET", "url": "/api/v1/transactions?keyword=grab&pagination.pageSize=20" }
```

Additional filters can be combined:

```json
{
  "method": "GET",
  "url": "/api/v1/transactions?keyword=coffee&type=TRANSACTION_TYPE_EXPENSE&dateRange.startDate=2026-01-01T00:00:00Z&dateRange.endDate=2026-03-31T23:59:59Z"
}
```

---

### SOP 13: Log a Partially-Budgeted Expense

**Trigger**: User says "spent $100 on groceries but only $50 counts toward budget", "company reimbursed half, only log $25 to budget".

Follow SOP 1 but set `budgetAmount` to the portion that counts:

```json
{
  "method": "POST",
  "url": "/api/v1/transactions",
  "body": {
    "categoryId": "<uuid>",
    "sourceAssetId": "<uuid>",
    "amount": { "amount": "100.00", "currency": "SGD" },
    "transactionDate": "2026-03-11T00:00:00Z",
    "description": "Groceries (half reimbursed)",
    "budgetAmount": "50.00"
  }
}
```

The full $100 is recorded as the expense, but only $50 counts against the budget.

---

## 2. Prerequisites Quick Reference

| Action | Must fetch first | Why |
|--------|-----------------|-----|
| Create transaction | `GET /categories` + `GET /assets` | Need `categoryId` + `sourceAssetId` |
| Update transaction | `GET /categories` + `GET /assets` | May need new `categoryId` or `sourceAssetId` |
| Create transfer | `GET /assets` | Need `fromAssetId` + `toAssetId` |
| Create budget | `GET /categories?type=TRANSACTION_TYPE_EXPENSE` | Need `categoryId` |
| Create asset | `GET /asset-types` | Need `assetTypeId` |
| Update asset | `GET /assets` | Need asset `id` |
| Update goal progress | `GET /goals` | Need goal `id` |
| Delete anything | Relevant list endpoint | Need the resource `id` |

---

## 3. API Reference

All endpoints are prefixed with `/api/v1`. Requests/responses use JSON. Authenticated endpoints require `Authorization: Bearer <token>`.

Full OpenAPI/Swagger spec available at `/swagger/` on the running server.

### Transactions

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/transactions` | Create a new transaction |
| `GET` | `/transactions` | List transactions (filterable) |
| `GET` | `/transactions/{id}` | Get a single transaction |
| `PATCH` | `/transactions/{id}` | Update a transaction |
| `DELETE` | `/transactions/{id}` | Delete a transaction |

**Create / Update fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `categoryId` | string (uuid) | create: yes | Category for this transaction |
| `sourceAssetId` | string (uuid) | create: yes | Asset account money flows from/to |
| `amount` | `{ amount, currency }` | create: yes | Monetary amount |
| `type` | enum | no | `TRANSACTION_TYPE_EXPENSE` or `TRANSACTION_TYPE_INCOME` (inferred from category if omitted) |
| `transactionDate` | ISO timestamp | create: yes | When the transaction occurred |
| `description` | string | no | Free-text note |
| `tags` | string[] | no | Tags for grouping |
| `budgetAmount` | string (decimal) | no | Portion that counts toward the budget. Omit = full amount (100%). Example: expense of 100, `budgetAmount: "50"` → only 50 counts against budget |

**List query params**: `pagination.page`, `pagination.pageSize`, `dateRange.startDate`, `dateRange.endDate`, `categoryId`, `type`, `currency`, `keyword`.

**Response fields** include all create fields plus: `id`, `categoryName`, `createdAt`, `budgetAmount` (only present when explicitly set).

### Transfers

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/transfers` | Transfer between assets |
| `GET` | `/transfers` | List all transfers |
| `PATCH` | `/transfers/{id}` | Update a transfer |
| `DELETE` | `/transfers/{id}` | Delete a transfer |

**Create fields**: `fromAssetId`, `toAssetId`, `fromAmount`, `toAmount`, `fromCurrency`, `toCurrency`, `exchangeRate`, `transferDate`, `description`.

### Budgets

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/budgets` | List configured budgets |
| `POST` | `/budgets` | Create a budget for a category |
| `PATCH` | `/budgets/{id}` | Update a budget |
| `DELETE` | `/budgets/{id}` | Delete a budget |
| `GET` | `/budgets/status` | All budget statuses (`?periodType=PERIOD_TYPE_MONTHLY`) |
| `GET` | `/budgets/{budget_id}/status` | Status of a specific budget |

Budget "spent" amounts automatically use `budgetAmount` from each transaction when set, falling back to the full `amount` otherwise.

### Categories

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/categories` | List categories (`?type=TRANSACTION_TYPE_EXPENSE` or `TRANSACTION_TYPE_INCOME`) |
| `POST` | `/categories` | Create a new category |
| `PATCH` | `/categories/{id}` | Update a category |
| `DELETE` | `/categories/{id}` | Delete a category |

### Assets

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/asset-types` | List asset types (`?category=ASSET_CATEGORY_BANK`) |
| `GET` | `/assets` | List all assets (always includes liabilities) |
| `POST` | `/assets` | Create an asset |
| `PATCH` | `/assets/{id}` | Update an asset |
| `DELETE` | `/assets/{id}` | Delete an asset |
| `POST` | `/assets/{asset_id}/snapshots` | Record a value snapshot |
| `GET` | `/assets/{asset_id}/history` | Get asset value history |

**List assets**: Always returns all assets and liabilities. Use client-side filtering if you need to separate them (check `isLiability` field).

**Update asset fields**: `name`, `currentValue`, `cost`, `assetTypeId`, `currency` — all optional, only provided fields change.

### Goals

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/goals` | List saving goals |
| `POST` | `/goals` | Create a goal |
| `PATCH` | `/goals/{id}` | Update a goal |
| `DELETE` | `/goals/{id}` | Delete a goal |
| `PUT` | `/goals/{id}/progress` | Update current amount (`changeSource`: `"manual"` / `"auto"`) |
| `GET` | `/goals/{id}/progress` | Get goal progress |
| `GET` | `/goals/progress` | Get all goals progress |

### Net Worth Goal

| Method | Endpoint | Description |
|--------|----------|-------------|
| `PUT` | `/net-worth-goal` | Set/update net worth goal |
| `GET` | `/net-worth-goal` | Get net worth goal |
| `DELETE` | `/net-worth-goal` | Delete net worth goal |
| `GET` | `/net-worth-goal/progress` | Get progress |

### Reports

| Method | Endpoint | Params | Description |
|--------|----------|--------|-------------|
| `GET` | `/reports/monthly` | `month=YYYY-MM` | Monthly summary |
| `GET` | `/reports/weekly` | `weekOf=<timestamp>` | Weekly snapshot |
| `GET` | `/reports/net-worth` | — | Current net worth |
| `GET` | `/reports/net-worth-trend` | `months`, `interval`, `year`, `month` | Net worth over time |
| `GET` | `/reports/spending-trend` | `months`, `categoryId` | Spending over time |
| `GET` | `/reports/budget-tracking` | `periodType`, `year`, `month` | Budget vs actual |
| `GET` | `/reports/goals` | — | Goals report |

All spending reports (monthly, weekly, budget-tracking, spending-trend) automatically use `budgetAmount` when calculating budget-related totals.

### Currencies

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/currencies` | List supported currencies |
| `GET` | `/currencies/rate` | Exchange rate (`?fromCurrency=USD&toCurrency=SGD`) |
| `GET` | `/currencies/convert` | Convert amount |

### Accounting

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/accounting/accounts` | Ledger accounts and balances |
| `GET` | `/accounting/journal` | Journal entries (`?limit=50`) |

---

## 4. Failure Behavior

- **401/403 Unauthorized**: Respond with `"API key invalid/expired, please check your connection settings."`
- **429 Too Many Requests**: Retry with exponential backoff silently.
- **400 Validation Error**: Ask the user for the specific missing or invalid field.
- **404 Not Found**: The resource ID is wrong. Re-fetch the list and try again.

## 5. Authentication & Secrets

Authentication parameters must **never** be output to chat or stored in conversation history.

```env
BUDGET_BASE_URL=https://budget.tet.sg
BUDGET_API_KEY=...
```

Header format: `Authorization: Bearer <BUDGET_API_KEY>`

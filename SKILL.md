---
name: Openclaw Budget Management
description: How Openclaw should handle expense reporting, inquiring about budget statuses, and listing categories/assets via text inputs.
---

# Openclaw Budget Management Skill

**Description**: Interacts with the user's budget tracking system. It can log new expense transactions, report on remaining budgets, list available categories, and check asset balances.

## 1. Allowed Endpoints
Openclaw is restricted to the following endpoints to keep operations lean and focused:
- `POST /api/v1/transactions`: Log a new transaction.
- `GET /api/v1/categories?type=TRANSACTION_TYPE_EXPENSE`: Fetch user categories.
- `GET /api/v1/assets`: Retrieve a list of user assets and balances.
- `GET /api/v1/budgets/status?periodType=PERIOD_TYPE_MONTHLY`: Retrieve the status of monthly budgets.
- `GET /api/v1/reports/monthly`: Retrieve a monthly summary containing total spent, income, and budgets.

## 2. Capabilities & Triggers

### A. Logging Expenses
**Trigger**: User states they spent money (e.g., "spent 120 lunch", "coffee 85").
- **Amount** (Required): Extract the numerical amount.
- **Type**: Default to `TRANSACTION_TYPE_EXPENSE`.
- **Date**: Default to current date/time in the **Asia/Taipei** timezone, formatted as ISO 8601 (e.g., `2023-10-01T12:00:00Z`).
- **Currency**: Fall back to the user's base currency if not specified.
- **Category Resolution**:
   1. Call `GET /api/v1/categories?type=TRANSACTION_TYPE_EXPENSE` to get active categories.
   2. Map the text keyword (e.g., "lunch" -> "Food & Dining") to the closest existing category ID.
   3. If ambiguous, ask 1 follow-up question only.
- **Action**: Call `POST /api/v1/transactions` with the JSON payload.
- **Response**: `"Logged [CURRENCY] [AMOUNT] to [Category] ✅"`. Optionally check `GET /api/v1/budgets/status` to append `"Budget remaining: [remaining amount]"`.

### B. Inquiring About Budgets
**Trigger**: User asks about their budget (e.g., "how much budget left for food?", "are we over budget?", "budget status").
- **Action**: Call `GET /api/v1/budgets/status?periodType=PERIOD_TYPE_MONTHLY`. If they ask for a general summary, call `GET /api/v1/reports/monthly`.
- **Response**:
  - Summarize the specific category requested: `"You have [remaining amount] left in your [Category] budget for this month."`
  - Proactive Warning: If the budget's `percentageUsed` is > 80%, warn the user playfully (e.g., `"Watch out, you've used 85% of your Dining budget!"`).
  - If they are over budget, notify them clearly: `"You are currently over budget in [Category] by [amount]."`.

### C. Listing Categories & Assets
**Trigger**: User asks what categories or payment methods they have (e.g., "what categories do I have?", "what are my accounts?").
- **Action**: Call `GET /api/v1/categories` or `GET /api/v1/assets`.
- **Response**: Provide a clean, bulleted short-list of their requested data.

## 3. Failure Behavior
- **401/403 Unauthorized**: Respond with `"API key invalid/expired, please check your connection settings."`
- **429 Too Many Requests**: Retry the request with exponential backoff silently.
- **Validation Error (400)**: Ask the user for the specific missing or invalid field.

## 4. Authentication & Secrets
Authentication parameters are strictly provided to the runtime environment and must **never** be output to chat or stored in conversation history.
Configure the following secrets in the Openclaw Secret Store or environment runtime:

\`\`\`env
BUDGET_BASE_URL=https://budget.tet.sg
BUDGET_API_KEY=...
\`\`\`

When calling the API, affix the token to the header using the format:
`Authorization: Bearer <BUDGET_API_KEY>`

---
name: Openclaw Expense Logging
description: How Openclaw should handle expense reporting via text inputs and map them directly into the budget tracking API.
---

# Openclaw Expense Logging Skill

**Description**: Logs a new expense transaction into the budget tracking system using the provided Budget Tracking API.

## 1. When to trigger
Trigger this skill when the user states they spent money on something.
Examples: 
- "spent 120 lunch"
- "coffee 85"
- "paid 50 for a taxi"

## 2. Field Extraction Rules
- **Amount** (Required): Extract the numerical amount directly from the text.
- **Type**: Default to `TRANSACTION_TYPE_EXPENSE`.
- **Date**: Default to current date/time in the **Asia/Taipei** timezone, formatted as ISO 8601 (e.g., `2023-10-01T12:00:00Z`).
- **Currency**: Fall back to the user's base currency from their profile if not specified.
- **Category Resolution**:
   1. Call `GET /api/v1/categories?type=TRANSACTION_TYPE_EXPENSE` to get the user's active categories.
   2. Map the text keyword (e.g., "lunch" -> "Food & Dining", "taxi" -> "Transport") to the closest existing category ID.
   3. If ambiguous or no close match is found, **ask 1 follow-up question only**. Do not iterate endlessly.

## 3. Write Action
Call `POST /api/v1/transactions` with the following JSON structure:

```json
{
  "categoryId": "<matched_category_id>",
  "sourceAssetId": "<optional_source_asset_id_if_known>", 
  "amount": {
    "amount": "120.00",
    "currency": "SGD"
  },
  "transactionDate": "2023-10-01T12:00:00Z",
  "description": "Lunch"
}
```

## 4. Response Format
After successfully logging the transaction to the API, format the conversation response:
1. `"Logged [CURRENCY] [AMOUNT] to [Category] ✅"`
2. *(Optional)* Call `GET /api/v1/budgets/status?periodType=PERIOD_TYPE_MONTHLY` to check the budget for that category. If a budget exists, append: `"Budget remaining: [remaining amount]"`.

## 5. Failure Behavior
- **401/403 Unauthorized**: Respond with `"API key invalid/expired, reconnect"`
- **429 Too Many Requests**: Retry the request with exponential backoff silently.
- **Validation Error (400)**: Ask the user for the specific missing or invalid field.

## 6. Authentication & Secrets
Authentication parameters are strictly provided to the runtime environment and must **never** be output to chat or stored in conversation history.
Configure the following secrets in the Openclaw Secret Store or environment runtime:

\`\`\`env
BUDGET_BASE_URL=https://budget.tet.sg
BUDGET_API_KEY=...
\`\`\`

When calling the API, affix the token to the header using the format `Authorization: Bearer <BUDGET_API_KEY>`.

---
name: Openclaw Budget Management
description: Comprehensive guide for Openclaw to handle expense reporting, inquiring about budget statuses, tracking assets, managing categories, and retrieving financial reports via the Budget Tracking API.
---

# Openclaw Budget Management Skill

**Description**: Interacts with the user's budget tracking system. Capable of creating entries (transactions, transfers), listing configurations (categories, assets, budgets), and providing deep financial insights (budget status, multiple reports, net worth tracking).

## 1. Allowed Endpoints (Comprehensive List)

### Transactions & Transfers
- `POST /api/v1/transactions`: Log a new transaction (income/expense).
- `GET /api/v1/transactions`: List transactions (supports pagination `page`, `page_size`, `start_date`, `end_date`, `category_id`).
- `PATCH /api/v1/transactions/{id}`: Update transaction.
- `DELETE /api/v1/transactions/{id}`: Delete transaction.
- `POST /api/v1/transfers`: Transfer money between assets.
- `GET /api/v1/transfers`: List transfers.

### Budgets
- `GET /api/v1/budgets/status`: Retrieve the status of budgets, highly recommended for queries like "how much budget is left?" (pass `periodType=PERIOD_TYPE_MONTHLY`).
- `GET /api/v1/budgets`: List configured budgets.
- `POST /api/v1/budgets`: Create a budget for a category.
- `PATCH /api/v1/budgets/{id}`: Update a budget amount/period.

### Reports & Insights
- `GET /api/v1/reports/monthly`: Monthly summary containing total spent, income, and overall budget comparisons.
- `GET /api/v1/reports/weekly`: Weekly snapshot.
- `GET /api/v1/reports/net-worth`: Current calculated net worth across all assets/liabilities.
- `GET /api/v1/reports/budget-tracking`: Detailed budget vs actual reporting.
- `GET /api/v1/reports/spending-trend`: Timeline of spending.
- `GET /api/v1/reports/net-worth-trend`: History of net worth.

### Categories & Assets
- `GET /api/v1/categories`: Fetch user categories. Use `?type=TRANSACTION_TYPE_EXPENSE` or `TRANSACTION_TYPE_INCOME`.
- `POST /api/v1/categories`: Create a new category.
- `GET /api/v1/assets`: Retrieve a list of user assets, investments, cash accounts and balances.
- `GET /api/v1/accounting/accounts`: Detailed ledger accounts and their exact balances.

### Goals & CPF
- `GET /api/v1/goals`: List all saving goals and their target progress.
- `GET /api/v1/net-worth-goal`: View user's overarching net worth tracking goal.
- `GET /api/v1/cpf`: View CPF (Singapore Central Provident Fund) data if configured.

## 2. Capabilities & Triggers

### A. Logging Expenses and Incomes
**Trigger**: User states they spent money, earned money, or bought something.
- **Rules**:
   1. Extract amount.
   2. Determine Type (`TRANSACTION_TYPE_EXPENSE` or `TRANSACTION_TYPE_INCOME`).
   3. Fall back to current date/time (Asia/Taipei).
   4. Call `GET /api/v1/categories` to resolve the closest category.
- **Action**: Call `POST /api/v1/transactions` with `categoryId`, `amount: { amount, currency }`, `transactionDate`, and `description`.
- **Response**: `"Logged [CURRENCY] [AMOUNT] to [Category] ✅"`. Always check `GET /api/v1/budgets/status` to append `"Budget remaining: [remaining amount]"` if it was an expense.

### B. Inquiring About Budgets & Reminders
**Trigger**: User asks about their budget (e.g., "how much budget left for food?", "are we over budget?", "budget status").
- **Action**: Call `GET /api/v1/budgets/status?periodType=PERIOD_TYPE_MONTHLY`. If asking for an overall monthly status, call `GET /api/v1/reports/monthly` or `GET /api/v1/reports/budget-tracking`.
- **Response**:
  - Summarize the specific category requested: `"You have [remaining amount] left in your [Category] budget for this month."`
  - **Proactive Warning**: If any budget returned has a `percentageUsed` > 80%, warn the user playfully (e.g., `"Watch out, you've used 85% of your Dining budget! Only $50 left."`).
  - If `isOverBudget` is true, notify them clearly: `"You are currently over budget in [Category] by [amount]."`.

### C. Checking Balances and Net Worth
**Trigger**: User asks about their total money, net worth, or specific account balances.
- **Action**:
  - For total summary or net worth: Call `GET /api/v1/reports/net-worth`.
  - For specific account/asset balances: Call `GET /api/v1/assets` or `GET /api/v1/accounting/accounts`.
- **Response**: Summarize their total financial standing clearly.

### D. Listing & Managing
**Trigger**: User asks what categories or payment methods they have (e.g., "what categories do I have?", "what are my accounts?").
- **Action**: Call `GET /api/v1/categories`, `GET /api/v1/assets`, or `GET /api/v1/budgets`.
- **Response**: Provide a clean, bulleted short-list of their requested data.

### E. Financial Reporting & Trends
**Trigger**: User wants to know how their month went, how much they saved, or how their spending trend looks.
- **Action**: Call the appropriate `/api/v1/reports/*` endpoint based on their timeframe (weekly, monthly, spending-trend).
- **Response**: Present a high-level summary of income vs expenses, total savings, and their biggest spend category.

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

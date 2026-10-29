# API Documentation

Base URL (local): `http://localhost:8080/api`
All bodies are JSON (`Content-Type: application/json`). Protected endpoints need `Authorization: Bearer <token>`.

## Response format

```json
{ "success": true,  "data": { ... }, "meta": { ... } }
{ "success": false, "error": { "code": "validation_failed", "message": "…", "details": { "amount": "…" } } }
```

`meta` appears only on paginated lists. `details` appears only for validation errors.

| Status | Meaning |
|---|---|
| 200 / 201 | OK / created |
| 400 | invalid JSON, validation error, bad filter or bad id |
| 401 | missing/invalid/expired token or wrong credentials (`token_expired`, `unauthorized`, `invalid_credentials`) |
| 404 | record not found **or belongs to another user** |
| 409 | duplicate e-mail (`email_taken`) or duplicate budget (`duplicate_budget`) |
| 429 | auth rate limit exceeded |
| 500 | internal error (details are logged on the server only) |

**Money:** amounts are sent as a number or string with at most 2 decimals (`"250.50"`), must be > 0 and ≤ 999,999,999.99. Responses return amounts as JSON numbers with two decimals.
**Dates:** `YYYY-MM-DD`. **Date presets:** `range=this_month | last_month | last_3_months | custom` (`custom` needs `from` and `to`). "Last 3 months" = the current month and the two before it.

## Authentication

| Method & path | Auth | Body | Notes |
|---|---|---|---|
| `POST /auth/register` | – | `{name, email, password}` | name 2–100 chars; password 8–72 chars with a letter and a digit. 201 → `{token, user}` |
| `POST /auth/login` | – | `{email, password}` | 200 → `{token, user}` |
| `POST /auth/logout` | ✔ | – | stateless; client discards the token |
| `GET /auth/me` | ✔ | – | current user |
| `PUT /auth/me` | ✔ | `{name}` | update display name |

Example:
```json
POST /auth/login
{ "email": "demo@example.com", "password": "Demo@12345" }

200 { "success": true, "data": { "token": "eyJ…", "user": { "id": 1, "name": "Demo User", "email": "demo@example.com", "created_at": "…", "updated_at": "…" } } }
```

## Transactions

| Method & path | Description |
|---|---|
| `GET /transactions` | list with filters (below) |
| `POST /transactions` | create |
| `GET /transactions/:id` | read one |
| `PUT /transactions/:id` | replace fields |
| `DELETE /transactions/:id` | delete |
| `GET /income` | same as list with `type=income` |
| `GET /expenses` | same as list with `type=expense` |
| `GET /meta/categories` | allowed categories and payment methods |

Create / update body:
```json
{ "type": "expense", "category": "Food", "amount": "250.50", "description": "Lunch",
  "transaction_date": "2026-01-15", "payment_method": "UPI" }
```
* `type`: `income` or `expense`. Income categories: Salary, Freelancing, Scholarship, Business, Other. Expense categories: Food, Transport, Education, Shopping, Entertainment, Healthcare, Bills, Rent, Other.
* `payment_method` (expenses only, required): Cash, Card, UPI, Net Banking, Other. Ignored for income.
* date between 2000-01-01 and one year ahead; description ≤ 255 characters.

Response record: `{id, user_id, type, category, amount, description, transaction_date, payment_method, created_at, updated_at}`.

List query parameters: `type`, `category`, `payment_method`, `search` (description/category), `from`, `to`, `sort` (`date`|`amount`), `order` (`asc`|`desc`), `page` (≥1), `page_size` (1–100, default 10).
Response: `data: [records]`, `meta: {page, page_size, total, total_pages}`.

## Budgets

| Method & path | Description |
|---|---|
| `GET /budgets?month=&year=` | budgets of a month (default: current) with spending |
| `POST /budgets` | `{category, amount, month, year}` – expense categories only; one budget per user/category/month |
| `PUT /budgets/:id` | same body; 409 if it would duplicate another budget |
| `DELETE /budgets/:id` | delete |

Budget view: `{id, category, month, year, amount, spent, remaining, utilization, status}` where
`remaining = amount − spent`, `utilization = spent ÷ amount × 100`, and `status` is `ok` (< 80 %), `warning` (≥ 80 %, not over) or `exceeded` (spent > amount). `spent` is calculated from the month's real expense transactions.

## Dashboard

| Endpoint | Query | Returns |
|---|---|---|
| `GET /dashboard/summary` | `range`, `from`, `to` | `total_income`, `total_expenses`, `net_savings`, `savings_percentage`, `average_daily_expenditure`, `highest_spending_category`, `monthly_expense_growth_pct`, `category_breakdown`, `previous_period{…}`, `recent_transactions` (5), `budget_utilization{month,year,budgets}`, `note` |
| `GET /dashboard/trends` | `months` (1–24, default 6) | one point per month: `{year, month, label, income, expenses, net_savings, savings_rate}` (zero-filled) |
| `GET /dashboard/categories` | `range`, `from`, `to`, `type` | `{categories: [{category, total, count, percentage}]}` |
| `GET /dashboard/prediction` | – | `{available: true, predicted_total, range_low, range_high, target_month, method, months_used, disclaimer}` or `{available: false, reason}` |

`net_savings` = income − expenses in the period (not a bank balance). Percentage fields are `null` when the comparison base is zero. `average_daily_expenditure` divides by the days elapsed so far for ranges that include today.

## Reports

| Endpoint | Query | Returns |
|---|---|---|
| `GET /reports/monthly` | `range` (default `last_3_months`), `from`, `to` | `months[]` (zero-filled) and `totals{income, expenses, net_savings, savings_rate}` |
| `GET /reports/categories` | `range`, `from`, `to` | `expense_categories[]`, `income_categories[]` |
| `GET /reports/daily` | `range`, `from`, `to` | `days[]`: days with activity `{date, income, expenses, net_savings}` |
| `GET /reports/budgets` | `month`, `year` | `budgets[]` and `summary{total_budgeted, total_spent, remaining, utilization, exceeded}` |
| `GET /reports/export` | `range`, `from`, `to` | CSV file (`Date,Type,Category,Amount,Payment Method,Description`) |

## Other

`GET /health` – no auth; checks the database connection.

# Viva Questions and Answers

**1. What does the project do?**
It is a web application that lets a user record income and expenses, set monthly budgets per category, see dashboards and charts, export reports, and (optionally) see an estimate of next month's spending.

**2. Why Go for the backend?**
Go compiles to one fast binary, has a strong standard library for HTTP and JSON, strict typing that suits financial code, and easy concurrency. Gin adds routing and middleware with very little overhead.

**3. Explain the architecture.**
A React single-page app calls a REST API written in Go (Gin). Requests pass through middleware (CORS, body limit, JWT auth, rate limit) to handlers, which use GORM/SQL to read and write PostgreSQL. Pure calculation code lives in `services`. An optional Python service predicts spending.

**4. How is authentication done?**
Passwords are hashed with bcrypt. At login the server signs a JWT (HS256) containing the user id and expiry. The client sends it in the `Authorization: Bearer` header. Middleware validates signature, algorithm and expiry and puts the user id into the request context.

**5. How do you make sure users only see their own data?**
The user id is taken from the validated token, never from the request. Every query has `WHERE user_id = ?`. Fetching another user's record returns 404, so it does not reveal that the record exists. Tests `TestUserOwnershipIsolation` check this.

**6. Why store money as integers?**
Floating point cannot represent values like 0.1 exactly, so sums drift. The database stores paise in a `BIGINT`; the `money` package parses and prints exactly two decimals.

**7. How are budget alerts calculated?**
`remaining = budget − spent`, `utilization = spent ÷ budget × 100`. `spent` is the sum of that category's expenses in the budget's month, computed by SQL. Status: ok below 80 %, warning from 80 %, exceeded above 100 %.

**8. What stops duplicate budgets?**
A `UNIQUE (user_id, category, month, year)` constraint plus a check inside a database transaction; the API answers 409.

**9. What is a database transaction and where is it used?**
A group of statements that succeeds or fails as a whole. Budget creation/update checks for duplicates and writes inside one transaction; the seed command inserts all demo data in one transaction.

**10. How do you prevent SQL injection?**
Only parameterised queries (`?` placeholders) are used. Sort columns are chosen from a whitelist, and LIKE wildcards in search text are escaped.

**11. What security measures exist besides bcrypt and JWT?**
Input validation, CORS allow-list, 1 MB body limit, rate limiting on login/register, generic error messages (details only in server logs), required 32-character JWT secret, CSV formula-injection protection, no secrets in the repository.

**12. Where is the JWT stored and what is the trade-off?**
In `localStorage`. It survives refreshes but could be read by injected JavaScript (XSS). React escapes output, which lowers the risk. HttpOnly cookies avoid script access but need CSRF protection.

**13. Difference between net savings and cash balance?**
Net savings = income − expenses in the chosen period. A cash balance would require the user's real account balances, which the app does not have, so it never claims one.

**14. How does the optional machine-learning feature work, and what if it is down?**
The Go API sends monthly totals to a FastAPI service. It needs at least 4 months; with 6+ months it fits a linear regression (if R² ≥ 0.30), otherwise it uses the average of the last 3 months. The result is labelled an estimate. If the service is missing or fails, the dashboard shows "unavailable" and everything else works.

**15. How were the features tested?**
Go unit tests for money parsing, JWT, middleware and calculations with known inputs; API integration tests against a separate `_test` database (registration, login, CRUD, ownership, validation, budgets, dashboard figures, reports, CSV); Python tests for the predictor; Vitest for frontend helpers. *(State honestly which of these you have run on your machine.)*

**16. What happens when a JWT expires?**
The API answers 401 `token_expired`; the frontend clears the token and redirects to the login page.

**17. How would you improve the project?**
Password change/reset, refresh tokens or revocation, recurring transactions, multiple accounts, receipts upload, e-mail alerts, Docker deployment of the full stack, and better forecasting models.

**18. Why a separate test database?**
Integration tests empty tables. A database whose name must end with `_test` guarantees real user data is never deleted.

**19. What are REST principles used here?**
Resource-based URLs (`/transactions/:id`), HTTP verbs for actions, status codes for outcomes, stateless authentication and a consistent JSON format.

**20. How does pagination work?**
`page` and `page_size` produce `LIMIT/OFFSET`; the response `meta` contains `total` and `total_pages` so the UI can show page controls.

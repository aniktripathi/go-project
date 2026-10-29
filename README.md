# Smart Personal Finance and Expense Tracker

A full-stack web application to manage income, track expenses, set monthly budgets, monitor savings and generate financial reports.

| Layer | Technology |
|---|---|
| Backend | Go, Gin, GORM, JWT, bcrypt |
| Database | PostgreSQL (SQL migrations in `backend/migrations`) |
| Frontend | React 18, Vite, React Router, Recharts |
| Optional ML | Python, pandas, scikit-learn, FastAPI |

> **New here?** Start with the [complete setup and user guide](USER_GUIDE.md).

> **Verification status.** In the current Windows workspace, the Go unit tests
> (`go test ./internal/...`) and frontend production build (`npm run build`)
> passed, and the API health endpoint returned HTTP 200. Database-backed API
> integration tests have not been run here.

---

## 1. What the application does

* Register / log in / log out (bcrypt passwords, JWT, rate-limited auth endpoints)
* Income and expense management with categories, payment methods, search, filters, sorting and pagination
* Monthly budgets per category with live "spent / remaining / utilization" and warnings at 80 % and over 100 %
* Dashboard: totals, savings rate, average daily spending, highest category, month-over-month growth, charts, recent transactions
* Reports: monthly, category-wise, daily, budget performance, **CSV export** and a **print-friendly** layout
* Optional: next-month expense prediction (the app works fully without it)

All amounts are stored as whole **paise** (integers) in PostgreSQL, so there is no floating-point rounding in money maths. Net savings is *income − expenses for the period*; it is **not** a bank balance.

## 2. Project structure

```
finance-tracker/
├── backend/                Go API
│   ├── cmd/server/         starts the API
│   ├── cmd/seed/           creates demo data
│   ├── config/             reads environment variables
│   ├── internal/
│   │   ├── auth/           bcrypt + JWT
│   │   ├── database/       connection + migration runner
│   │   ├── handlers/       HTTP endpoints
│   │   ├── middleware/     auth, rate limit, body limit
│   │   ├── models/         table structs and category lists
│   │   ├── money/          exact money type
│   │   ├── routes/         URL -> handler wiring
│   │   └── services/       pure calculations (budget %, ranges, ...)
│   ├── migrations/         *.sql schema files
│   ├── tests/              API integration tests
│   └── .env.example
├── frontend/               React app (src/components, pages, services, hooks)
├── ml-service/             optional prediction service
├── postman/                Postman collection
├── docs/                   API docs, capstone report, viva Q&A, checklist
└── docker-compose.yml      optional PostgreSQL in Docker
```

## 3. Install the prerequisites (Windows, PowerShell)

Open **PowerShell** (right-click Start → *Terminal*). You can use the installers from the websites or `winget`:

```powershell
winget install --id GoLang.Go -e
winget install --id OpenJS.NodeJS.LTS -e
winget install --id PostgreSQL.PostgreSQL.16 -e
winget install --id Git.Git -e
```

* The PostgreSQL installer asks for a password for the `postgres` user. **Remember it.** Keep port `5432`.
* **Close and reopen PowerShell** afterwards so the new programs are found.
* Check the installs:

```powershell
go version        # go1.22 or newer
node --version    # v18 or newer
npm --version
git --version
```

## 4. Create the databases

Replace the path if your PostgreSQL version differs. You will be asked for the `postgres` password.

```powershell
$psql = "C:\Program Files\PostgreSQL\16\bin\psql.exe"
& $psql -U postgres -c "CREATE DATABASE finance_tracker;"
& $psql -U postgres -c "CREATE DATABASE finance_tracker_test;"
```

(`finance_tracker_test` is a **separate** database used only by the automated tests.)

*Docker alternative:* copy `.env.example` to `.env`, set `POSTGRES_PASSWORD`, then `docker compose up -d db`. This creates both databases.

## 5. Configure and start the backend

```powershell
cd finance-tracker\backend
Copy-Item .env.example .env
notepad .env
```

In `.env` set:

1. `DATABASE_URL` – put your PostgreSQL password in place of `YOUR_PASSWORD`. If the password contains special characters such as `@` or `#`, URL-encode them (`@` → `%40`, `#` → `%23`).
2. `JWT_SECRET` – at least 32 random characters. Generate one:

```powershell
[Convert]::ToBase64String((1..48 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])
```

Then download dependencies, create the tables (migrations run automatically at start-up) and run:

```powershell
go mod tidy
go run ./cmd/server
```

You should see `database ready, migrations applied` and `API listening on http://localhost:8080`. Check it: open <http://localhost:8080/api/health> in a browser (`{"data":{"status":"ok"},"success":true}`).

**Optional demo data** (in a second PowerShell window, inside `backend`):

```powershell
go run ./cmd/seed
```

Demo login: `demo@example.com` / `Demo@12345` (local demos only – never use these on a public server).

## 6. Start the frontend

In another PowerShell window:

```powershell
cd finance-tracker\frontend
Copy-Item .env.example .env
npm install
npm run dev
```

Open <http://localhost:5173>. Vite forwards `/api` requests to the Go backend on port 8080, so the two parts work together without CORS trouble.

Production build check: `npm run build` (creates `frontend/dist`).

## 7. Test the APIs with Postman

1. Install Postman, choose *Import* and select `postman/FinanceTracker.postman_collection.json`.
2. With the backend running, run the **Authentication → Register** request (it stores the token automatically), then try the other folders in order.
3. Endpoint details: `docs/API.md`.

## 8. Run the automated tests

**Unit tests (no database needed):**

```powershell
cd finance-tracker\backend
go test ./internal/...
```

**API integration tests (need the *separate* test database from step 4):**

```powershell
$env:TEST_DATABASE_URL = "postgres://postgres:YOUR_PASSWORD@localhost:5432/finance_tracker_test?sslmode=disable"
go test ./tests/... -v
```

The tests **empty the tables of the test database** and refuse to run unless the database name ends with `_test`, so your real data in `finance_tracker` is never touched. Without `TEST_DATABASE_URL` they are skipped.

Run everything with verbose output: `go test ./... -v`. Also run `go vet ./...` and `go test -race ./...` for extra checks.

**Frontend checks:** `cd frontend; npm test` (helper tests) and `npm run build`.

**Prediction service tests:** `cd ml-service; python -m unittest -v`.

## 9. Optional: expense prediction

```powershell
cd finance-tracker\ml-service
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
uvicorn app:app --port 8001
```

Then add `ML_SERVICE_URL=http://localhost:8001` to `backend\.env` and restart the Go server. The dashboard shows an estimate once you have at least 4 completed months of expenses (the seed command creates 6). If the service is stopped, the dashboard simply says it is unavailable.

## 10. Security notes

* Passwords: bcrypt hashes only. JWT signed with HS256, algorithm pinned, secret from environment (min 32 chars).
* The user id always comes from the validated token, never from the request. Other users' records answer with `404`.
* Parameterised SQL everywhere; whitelisted sort columns; LIKE wildcards escaped.
* CORS restricted to `CORS_ALLOWED_ORIGINS`; 1 MB body limit; 10 auth requests/minute/IP by default.
* Token storage: the frontend keeps the JWT in `localStorage`. Trade-off: JavaScript on the page could read it if an XSS bug existed. React escapes output and the app never injects raw HTML. An HttpOnly cookie is the stricter alternative but needs CSRF protection and `Secure`/`SameSite` configuration.
* CSV export neutralises spreadsheet formulas (cells starting with `= + - @`).
* Never commit `.env`. Use HTTPS in any real deployment.

## 11. Troubleshooting

| Problem | Fix |
|---|---|
| `go : The term 'go' is not recognized` | Close and reopen PowerShell after installing Go. |
| `configuration error: JWT_SECRET must be set…` | Fill `JWT_SECRET` (32+ chars) in `backend\.env`. |
| `password authentication failed for user "postgres"` | Wrong password in `DATABASE_URL`; URL-encode special characters. |
| `connection refused` / `database is not reachable` | PostgreSQL is not running: open *Services* and start `postgresql-x64-16`, or `docker compose up -d db`. |
| `database "finance_tracker" does not exist` | Run the `CREATE DATABASE` commands in step 4. |
| `missing go.sum entry` | Run `go mod tidy` inside `backend`. |
| `port 8080 is already in use` | Change `APP_PORT` in `.env` and `VITE_BACKEND_URL` in `frontend\.env`. |
| Browser shows "Cannot reach the server" | Backend is not running or the port differs. |
| `429 Too many attempts` | Auth rate limit; wait one minute. |
| `running scripts is disabled` (venv activation) | `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned` |
| Tests skipped | `TEST_DATABASE_URL` is not set in the *same* PowerShell window. |

## 12. Known limitations

* No password change / reset or e-mail verification.
* JWT logout is client-side (tokens cannot be revoked before they expire).
* Single currency (₹ INR). No bank/UPI account linking, so "net savings" is not a bank balance.
* The prediction is a simple trend/average estimate, not a financial forecast.

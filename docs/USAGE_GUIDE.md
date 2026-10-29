# Finance Tracker: Complete Setup and Usage Guide

This guide covers setting up the project on Windows, starting each service, using
the app, running checks, and publishing the code to GitHub.

## 1. What you need

Install these tools and reopen PowerShell afterward:

- Go 1.22 or newer
- Node.js 18 or newer (includes npm)
- PostgreSQL 16, or Docker Desktop
- Git

Check that they are available:

```powershell
go version
node --version
npm --version
git --version
```

All commands below assume the project is at:

```text
C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker
```

Use the actual folder path on your computer.

## 2. Start PostgreSQL

### Option A: Use an installed PostgreSQL server

Start the PostgreSQL service from Windows Services. Create the application
database using pgAdmin or PowerShell. Update the executable path if you installed
a different PostgreSQL version:

```powershell
$psql = "C:\Program Files\PostgreSQL\16\bin\psql.exe"
& $psql -U postgres -c "CREATE DATABASE finance_tracker;"
```

Enter the password you chose during PostgreSQL setup when prompted.

### Option B: Use Docker Desktop

From the project root, copy the example Docker settings and choose a local-only
password:

```powershell
Copy-Item .env.example .env
notepad .env
```

Set `POSTGRES_PASSWORD`, save the file, then start the database:

```powershell
docker compose up -d db
```

Do not use the example password on an internet-facing server.

## 3. Configure and start the Go API

Open a PowerShell terminal in the project root:

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\backend"
Copy-Item .env.example .env
notepad .env
```

In `backend\.env`, set:

- `DATABASE_URL` to your PostgreSQL connection. For a local server on port 5432,
  a typical value is
  `postgres://postgres:YOUR_PASSWORD@localhost:5432/finance_tracker?sslmode=disable`.
- `JWT_SECRET` to a random value of at least 32 characters. Generate one in
  PowerShell with:

```powershell
[Convert]::ToBase64String((1..48 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])
```

Keep `.env` private; it is ignored by Git. Then install Go dependencies and start
the API:

```powershell
go mod tidy
go run ./cmd/server
```

Migrations run automatically on startup. Keep this terminal open. Confirm the
API is ready by opening <http://localhost:8080/api/health>; it should return:

```json
{"data":{"status":"ok"},"success":true}
```

## 4. Start the React app

Open a second PowerShell terminal:

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\frontend"
Copy-Item .env.example .env
npm install
npm run dev
```

Keep this terminal open and open <http://localhost:5173> in your browser. The
Vite development server forwards API requests to the Go service on port 8080.

## 5. Create an account and use the app

1. Select **Register** and create an account with your name, email, and password.
   Each account has its own transactions and budgets.
2. Add income or expenses from the transaction pages. Enter the amount in rupees;
   the application stores money internally as whole paise.
3. Use categories, dates, payment methods, and descriptions to organize and find
   transactions.
4. Open **Budgets** to set a monthly limit for a category and monitor spending.
   The app shows utilization and warnings as spending approaches or exceeds the
   limit.
5. Open **Dashboard** to review income, expenses, savings, spending trends, and
   category totals for the selected period.
6. Open **Reports** for monthly, daily, category, and budget reports. Use the
   export option to download a CSV or the print-friendly layout to print a
   report.
7. Open **Profile** to update your account information, or log out when done.

Net savings in the reports means income minus expenses for the selected period;
it is not a bank balance. This app does not connect to banks or payment accounts.

### Optional demo data

To load sample transactions, run this from a third PowerShell terminal after
configuring `backend\.env`:

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\backend"
go run ./cmd/seed
```

Demo login: `demo@example.com` / `Demo@12345`. This account is only for local
demos. Do not use the demo password for a real account or public deployment. The
seed command creates the demo user only if it does not already exist.

## 6. Run tests and build

Go unit tests (no database required):

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\backend"
go test ./internal/...
```

Frontend tests and production build:

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\frontend"
npm test
npm run build
```

Python prediction service tests (Python and its requirements must be installed):

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\ml-service"
python -m unittest -v
```

API integration tests require a separate database named `finance_tracker_test`.
Those tests clear tables in that test database, so never point
`TEST_DATABASE_URL` at your normal application database:

```powershell
$env:TEST_DATABASE_URL = "postgres://postgres:YOUR_PASSWORD@localhost:5432/finance_tracker_test?sslmode=disable"
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\backend"
go test ./tests/... -v
```

## 7. Optional expense prediction service

The app works without the prediction service. To run it, create a Python virtual
environment and install its requirements:

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker\ml-service"
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
uvicorn app:app --port 8001
```

Set `ML_SERVICE_URL=http://localhost:8001` in `backend\.env`, then restart the Go
API. The prediction needs at least four completed months of expense history.

## 8. Publish the project to GitHub

Create an empty GitHub repository named `go-project` (do not initialize it with
a README, license, or `.gitignore`). Then run these commands from the project
root, replacing the URL with your repository URL:

```powershell
Set-Location "C:\Users\<your-name>\OneDrive\Desktop\finance-tracker\finance-tracker"
git init -b main
git add .
git status --short
git commit -m "Initial finance tracker project"
git remote add origin https://github.com/YOUR_USERNAME/go-project.git
git push -u origin main
```

Before committing, confirm that `.env`, `backend\.env`, and `frontend\.env`
do not appear in `git status`. The project `.gitignore` excludes these files,
dependencies, and generated build output. GitHub may ask you to authenticate in
the browser or through Git Credential Manager; never put a password or token in
the remote URL.

For later updates:

```powershell
git add .
git status --short
git commit -m "Describe your change"
git push
```

## 9. Stop and restart

To stop the API and frontend, focus each terminal and press **Ctrl+C**. Stop the
Docker database with `docker compose down` if you started it with Docker; this
keeps its named data volume. For a Windows PostgreSQL service, stop it from
Windows Services only when you no longer need it.

To start the app again, start PostgreSQL, then repeat the API steps in one
terminal and the frontend steps in another.

## 10. Troubleshooting

| Problem | What to check |
|---|---|
| API says `DATABASE_URL is not set` | Confirm `backend\.env` exists and has a `DATABASE_URL`. |
| Database connection refused | Start PostgreSQL and confirm the host and port in `DATABASE_URL`. |
| `JWT_SECRET must be set` | Set a secret of at least 32 characters in `backend\.env`. |
| Port 8080 is already in use | Stop the other process or change `APP_PORT` and the Vite backend target together. |
| Frontend says it cannot reach the server | Confirm the API health URL responds and the backend port matches the frontend proxy configuration. |
| `git push` asks for a password or fails | Sign in with Git Credential Manager/browser authentication; GitHub does not accept account passwords for HTTPS Git pushes. |
| Git wants to add `.env` | Stop before committing; verify `.gitignore` and remove the file from the Git index if it was already staged. |

More endpoint details are in [API.md](./API.md); the project overview and
additional checks are in [the README](../README.md).

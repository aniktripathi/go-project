// Package tests contains API integration tests.
//
// They need a SEPARATE PostgreSQL database whose name ends with "_test", for example
// finance_tracker_test. Set TEST_DATABASE_URL to run them. Without it the tests are skipped.
// The tests empty the tables of that test database, so they refuse to run on any other database.
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"financetracker/config"
	"financetracker/internal/auth"
	"financetracker/internal/database"
	"financetracker/internal/routes"

	"github.com/gin-gonic/gin"
)

const testSecret = "test-secret-test-secret-test-secret-123"
const password = "Passw0rd123"

type response struct {
	Code int
	Raw  string
	Body map[string]interface{}
}

func setup(t *testing.T) *gin.Engine {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping API integration tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(strings.TrimPrefix(u.Path, "/"), "_test") {
		t.Fatal("refusing to run: the database name in TEST_DATABASE_URL must end with _test")
	}
	db, err := database.Connect(dsn)
	if err != nil {
		t.Fatalf("cannot connect to test database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE budgets, transactions, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWTSecret: testSecret, JWTExpiryHours: 1,
		AllowedOrigins: []string{"http://localhost:5173"}, AuthRateLimitPerMin: 0,
	}
	return routes.SetupRouter(db, cfg)
}

func call(t *testing.T, r *gin.Engine, method, path, token string, body interface{}) response {
	t.Helper()
	var reader *bytes.Reader
	switch b := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(b))
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := response{Code: w.Code, Raw: w.Body.String()}
	_ = json.Unmarshal(w.Body.Bytes(), &res.Body)
	return res
}

// dig walks nested JSON: dig(m, "data", "user", "id").
func dig(m interface{}, keys ...string) interface{} {
	cur := m
	for _, k := range keys {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

func num(t *testing.T, v interface{}) float64 {
	t.Helper()
	f, ok := v.(float64)
	if !ok {
		t.Fatalf("expected a number, got %v (%T)", v, v)
	}
	return f
}

func near(t *testing.T, name string, got interface{}, want float64) {
	t.Helper()
	if math.Abs(num(t, got)-want) > 0.011 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func expectCode(t *testing.T, res response, want int) {
	t.Helper()
	if res.Code != want {
		t.Fatalf("status %d, want %d, body: %s", res.Code, want, res.Raw)
	}
}

func register(t *testing.T, r *gin.Engine, name, email string) string {
	t.Helper()
	res := call(t, r, "POST", "/api/auth/register", "", map[string]string{"name": name, "email": email, "password": password})
	expectCode(t, res, 201)
	token, _ := dig(res.Body, "data", "token").(string)
	if token == "" {
		t.Fatal("no token returned")
	}
	return token
}

func addTx(t *testing.T, r *gin.Engine, token, typ, cat, amount, date string) float64 {
	t.Helper()
	body := map[string]interface{}{"type": typ, "category": cat, "amount": amount, "transaction_date": date, "description": "test"}
	if typ == "expense" {
		body["payment_method"] = "UPI"
	}
	res := call(t, r, "POST", "/api/transactions", token, body)
	expectCode(t, res, 201)
	return num(t, dig(res.Body, "data", "id"))
}

// ---------------------------------------------------------------- authentication

func TestRegisterAndLogin(t *testing.T) {
	r := setup(t)
	register(t, r, "Asha Rao", "asha@example.com")

	// duplicate e-mail (different letter case) is rejected
	res := call(t, r, "POST", "/api/auth/register", "", map[string]string{"name": "Other", "email": "ASHA@example.com", "password": password})
	expectCode(t, res, 409)

	// validation errors
	res = call(t, r, "POST", "/api/auth/register", "", map[string]string{"name": "A", "email": "not-an-email", "password": "short"})
	expectCode(t, res, 400)
	for _, f := range []string{"name", "email", "password"} {
		if dig(res.Body, "error", "details", f) == nil {
			t.Errorf("missing validation message for %s", f)
		}
	}
	expectCode(t, call(t, r, "POST", "/api/auth/register", "", "{bad json"), 400)

	// login
	res = call(t, r, "POST", "/api/auth/login", "", map[string]string{"email": "asha@example.com", "password": password})
	expectCode(t, res, 200)
	token, _ := dig(res.Body, "data", "token").(string)
	if strings.Contains(res.Raw, "password_hash") {
		t.Error("password hash must never appear in a response")
	}
	expectCode(t, call(t, r, "POST", "/api/auth/login", "", map[string]string{"email": "asha@example.com", "password": "WrongPass1"}), 401)
	expectCode(t, call(t, r, "POST", "/api/auth/login", "", map[string]string{"email": "nobody@example.com", "password": password}), 401)

	// /me
	res = call(t, r, "GET", "/api/auth/me", token, nil)
	expectCode(t, res, 200)
	if dig(res.Body, "data", "email") != "asha@example.com" {
		t.Errorf("me returned %s", res.Raw)
	}
	expectCode(t, call(t, r, "GET", "/api/auth/me", "", nil), 401)

	expired, _ := auth.GenerateToken(testSecret, 1, -time.Minute)
	res = call(t, r, "GET", "/api/auth/me", expired, nil)
	expectCode(t, res, 401)
	if dig(res.Body, "error", "code") != "token_expired" {
		t.Errorf("expected token_expired, got %s", res.Raw)
	}
}

func TestProtectedRoutesRequireLogin(t *testing.T) {
	r := setup(t)
	paths := []struct{ method, path string }{
		{"GET", "/api/transactions"}, {"POST", "/api/transactions"}, {"GET", "/api/transactions/1"},
		{"PUT", "/api/transactions/1"}, {"DELETE", "/api/transactions/1"}, {"GET", "/api/income"},
		{"GET", "/api/expenses"}, {"GET", "/api/budgets"}, {"POST", "/api/budgets"},
		{"GET", "/api/dashboard/summary"}, {"GET", "/api/dashboard/trends"}, {"GET", "/api/dashboard/categories"},
		{"GET", "/api/reports/monthly"}, {"GET", "/api/reports/categories"}, {"GET", "/api/reports/export"},
	}
	for _, p := range paths {
		if res := call(t, r, p.method, p.path, "", nil); res.Code != 401 {
			t.Errorf("%s %s without token: status %d, want 401", p.method, p.path, res.Code)
		}
	}
}

// ---------------------------------------------------------------- transactions

func TestTransactionCRUD(t *testing.T) {
	r := setup(t)
	token := register(t, r, "Asha Rao", "asha@example.com")

	res := call(t, r, "POST", "/api/transactions", token, map[string]interface{}{
		"type": "expense", "category": "Food", "amount": 120.5, "description": "Lunch",
		"transaction_date": "2026-01-10", "payment_method": "Cash",
	})
	expectCode(t, res, 201)
	near(t, "amount", dig(res.Body, "data", "amount"), 120.50)
	id := int(num(t, dig(res.Body, "data", "id")))
	path := fmt.Sprintf("/api/transactions/%d", id)

	res = call(t, r, "GET", path, token, nil)
	expectCode(t, res, 200)
	if dig(res.Body, "data", "category") != "Food" || dig(res.Body, "data", "transaction_date") != "2026-01-10" {
		t.Errorf("unexpected record %s", res.Raw)
	}

	res = call(t, r, "PUT", path, token, map[string]interface{}{
		"type": "expense", "category": "Transport", "amount": "99.99", "description": "Cab",
		"transaction_date": "2026-01-11", "payment_method": "UPI",
	})
	expectCode(t, res, 200)
	near(t, "updated amount", dig(res.Body, "data", "amount"), 99.99)
	if dig(res.Body, "data", "category") != "Transport" {
		t.Errorf("category not updated: %s", res.Raw)
	}

	res = call(t, r, "GET", "/api/transactions", token, nil)
	expectCode(t, res, 200)
	near(t, "total", dig(res.Body, "meta", "total"), 1)

	expectCode(t, call(t, r, "DELETE", path, token, nil), 200)
	expectCode(t, call(t, r, "GET", path, token, nil), 404)
	expectCode(t, call(t, r, "DELETE", path, token, nil), 404)
}

func TestUserOwnershipIsolation(t *testing.T) {
	r := setup(t)
	tokenA := register(t, r, "User A", "a@example.com")
	tokenB := register(t, r, "User B", "b@example.com")

	id := int(addTx(t, r, tokenA, "expense", "Food", "50.00", "2026-01-05"))
	path := fmt.Sprintf("/api/transactions/%d", id)

	// User B cannot read, change or delete User A's record.
	expectCode(t, call(t, r, "GET", path, tokenB, nil), 404)
	expectCode(t, call(t, r, "PUT", path, tokenB, map[string]interface{}{
		"type": "expense", "category": "Food", "amount": "1.00", "transaction_date": "2026-01-05", "payment_method": "Cash",
	}), 404)
	expectCode(t, call(t, r, "DELETE", path, tokenB, nil), 404)

	// User B's lists, dashboard and export contain nothing from User A.
	res := call(t, r, "GET", "/api/transactions", tokenB, nil)
	near(t, "B total", dig(res.Body, "meta", "total"), 0)
	res = call(t, r, "GET", "/api/dashboard/summary?range=custom&from=2026-01-01&to=2026-01-31", tokenB, nil)
	near(t, "B expenses", dig(res.Body, "data", "total_expenses"), 0)
	res = call(t, r, "GET", "/api/reports/export?range=custom&from=2026-01-01&to=2026-01-31", tokenB, nil)
	if strings.Contains(res.Raw, "Food") {
		t.Error("export leaked another user's data")
	}

	// A budget of user A cannot be deleted or edited by user B.
	res = call(t, r, "POST", "/api/budgets", tokenA, map[string]interface{}{"category": "Food", "amount": "100", "month": 1, "year": 2026})
	expectCode(t, res, 201)
	bpath := fmt.Sprintf("/api/budgets/%d", int(num(t, dig(res.Body, "data", "id"))))
	expectCode(t, call(t, r, "DELETE", bpath, tokenB, nil), 404)
	expectCode(t, call(t, r, "PUT", bpath, tokenB, map[string]interface{}{"category": "Food", "amount": "5", "month": 1, "year": 2026}), 404)

	// The id sent by a client is ignored: user_id in the body cannot be spoofed.
	res = call(t, r, "POST", "/api/transactions", tokenB, map[string]interface{}{
		"user_id": 1, "type": "income", "category": "Salary", "amount": "10", "transaction_date": "2026-01-05",
	})
	expectCode(t, res, 201)
	if num(t, dig(res.Body, "data", "user_id")) != 2 {
		t.Errorf("record was stored for the wrong user: %s", res.Raw)
	}

	// User A's record is untouched.
	expectCode(t, call(t, r, "GET", path, tokenA, nil), 200)
}

func TestInvalidInput(t *testing.T) {
	r := setup(t)
	token := register(t, r, "Asha Rao", "asha@example.com")
	base := func() map[string]interface{} {
		return map[string]interface{}{"type": "expense", "category": "Food", "amount": "10.00",
			"transaction_date": "2026-01-10", "payment_method": "Cash"}
	}
	cases := []struct {
		name, field string
		value       interface{}
	}{
		{"negative amount", "amount", "-5"},
		{"zero amount", "amount", "0"},
		{"text amount", "amount", "abc"},
		{"too many decimals", "amount", "1.234"},
		{"missing amount", "amount", ""},
		{"exponent amount", "amount", "1e3"},
		{"bad date format", "transaction_date", "10/01/2026"},
		{"impossible date", "transaction_date", "2026-02-31"},
		{"far future date", "transaction_date", "2099-01-01"},
		{"wrong category", "category", "Salary"},
		{"unknown type", "type", "transfer"},
		{"bad payment method", "payment_method", "Bitcoin"},
	}
	for _, tc := range cases {
		body := base()
		body[tc.field] = tc.value
		res := call(t, r, "POST", "/api/transactions", token, body)
		if res.Code != 400 {
			t.Errorf("%s: status %d, want 400 (%s)", tc.name, res.Code, res.Raw)
			continue
		}
		if dig(res.Body, "error", "code") != "validation_failed" {
			t.Errorf("%s: unexpected error %s", tc.name, res.Raw)
		}
	}
	expectCode(t, call(t, r, "POST", "/api/transactions", token, "{not json"), 400)
	expectCode(t, call(t, r, "GET", "/api/transactions/abc", token, nil), 400)
	expectCode(t, call(t, r, "GET", "/api/transactions/999999", token, nil), 404)
	for _, q := range []string{"sort=price", "order=sideways", "page=0", "page_size=1000", "from=bad", "type=loan", "from=2026-02-01&to=2026-01-01"} {
		if res := call(t, r, "GET", "/api/transactions?"+q, token, nil); res.Code != 400 {
			t.Errorf("filter %q: status %d, want 400", q, res.Code)
		}
	}
	// invalid budget input
	for _, b := range []map[string]interface{}{
		{"category": "Salary", "amount": "10", "month": 1, "year": 2026},
		{"category": "Food", "amount": "-1", "month": 1, "year": 2026},
		{"category": "Food", "amount": "10", "month": 13, "year": 2026},
		{"category": "Food", "amount": "10", "month": 1, "year": 1900},
	} {
		if res := call(t, r, "POST", "/api/budgets", token, b); res.Code != 400 {
			t.Errorf("budget %v: status %d, want 400", b, res.Code)
		}
	}
	expectCode(t, call(t, r, "GET", "/api/dashboard/summary?range=custom&from=2026-02-01&to=2026-01-01", token, nil), 400)
	expectCode(t, call(t, r, "GET", "/api/dashboard/summary?range=forever", token, nil), 400)
}

func TestIncomeExpenseListsFilteringSortingPagination(t *testing.T) {
	r := setup(t)
	token := register(t, r, "Asha Rao", "asha@example.com")
	addTx(t, r, token, "income", "Salary", "1000.00", "2026-01-01")
	for i := 1; i <= 12; i++ {
		addTx(t, r, token, "expense", "Food", fmt.Sprintf("%d.00", i*10), fmt.Sprintf("2026-01-%02d", i))
	}
	addTx(t, r, token, "expense", "Rent", "5000.00", "2026-02-01")

	res := call(t, r, "GET", "/api/income", token, nil)
	near(t, "income total", dig(res.Body, "meta", "total"), 1)
	res = call(t, r, "GET", "/api/expenses?page_size=5", token, nil)
	near(t, "expense total", dig(res.Body, "meta", "total"), 13)
	near(t, "total_pages", dig(res.Body, "meta", "total_pages"), 3)
	res = call(t, r, "GET", "/api/expenses?category=Rent", token, nil)
	near(t, "rent total", dig(res.Body, "meta", "total"), 1)
	res = call(t, r, "GET", "/api/transactions?from=2026-01-05&to=2026-01-08&type=expense", token, nil)
	near(t, "date filtered", dig(res.Body, "meta", "total"), 4)

	// sorting by amount ascending: smallest expense first
	res = call(t, r, "GET", "/api/expenses?sort=amount&order=asc&page_size=3", token, nil)
	items, _ := dig(res.Body, "data").([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	near(t, "smallest amount", dig(items[0], "amount"), 10)

	// search
	addTx(t, r, token, "expense", "Other", "7.00", "2026-01-20")
	res = call(t, r, "POST", "/api/transactions", token, map[string]interface{}{
		"type": "expense", "category": "Other", "amount": "3.00", "description": "Birthday gift 100%",
		"transaction_date": "2026-01-21", "payment_method": "Cash",
	})
	expectCode(t, res, 201)
	res = call(t, r, "GET", "/api/transactions?search=birthday", token, nil)
	near(t, "search total", dig(res.Body, "meta", "total"), 1)
	res = call(t, r, "GET", "/api/transactions?search=%25", token, nil) // a literal % must not match everything
	near(t, "percent search total", dig(res.Body, "meta", "total"), 1)
}

// ---------------------------------------------------------------- budgets

func TestBudgetCalculations(t *testing.T) {
	r := setup(t)
	token := register(t, r, "Asha Rao", "asha@example.com")
	addTx(t, r, token, "expense", "Food", "1200.25", "2026-01-06")
	addTx(t, r, token, "expense", "Food", "799.75", "2026-01-20")
	addTx(t, r, token, "expense", "Transport", "300.00", "2026-01-08")
	addTx(t, r, token, "expense", "Food", "500.00", "2026-02-02") // another month: must not count

	res := call(t, r, "POST", "/api/budgets", token, map[string]interface{}{"category": "Food", "amount": "2500", "month": 1, "year": 2026})
	expectCode(t, res, 201)
	near(t, "spent", dig(res.Body, "data", "spent"), 2000)
	near(t, "remaining", dig(res.Body, "data", "remaining"), 500)
	near(t, "utilization", dig(res.Body, "data", "utilization"), 80)
	if dig(res.Body, "data", "status") != "warning" {
		t.Errorf("80%% should be a warning: %s", res.Raw)
	}
	foodID := int(num(t, dig(res.Body, "data", "id")))

	res = call(t, r, "POST", "/api/budgets", token, map[string]interface{}{"category": "Transport", "amount": "250", "month": 1, "year": 2026})
	expectCode(t, res, 201)
	near(t, "remaining over budget", dig(res.Body, "data", "remaining"), -50)
	near(t, "utilization over", dig(res.Body, "data", "utilization"), 120)
	if dig(res.Body, "data", "status") != "exceeded" {
		t.Errorf("120%% should be exceeded: %s", res.Raw)
	}

	// duplicate budget for the same category and month is rejected
	res = call(t, r, "POST", "/api/budgets", token, map[string]interface{}{"category": "Food", "amount": "10", "month": 1, "year": 2026})
	expectCode(t, res, 409)
	// the same category in another month is fine
	expectCode(t, call(t, r, "POST", "/api/budgets", token, map[string]interface{}{"category": "Food", "amount": "10", "month": 2, "year": 2026}), 201)

	// listing and editing
	res = call(t, r, "GET", "/api/budgets?month=1&year=2026", token, nil)
	expectCode(t, res, 200)
	if items, _ := dig(res.Body, "data").([]interface{}); len(items) != 2 {
		t.Errorf("expected 2 budgets in January, got %s", res.Raw)
	}
	path := fmt.Sprintf("/api/budgets/%d", foodID)
	res = call(t, r, "PUT", path, token, map[string]interface{}{"category": "Food", "amount": "4000", "month": 1, "year": 2026})
	expectCode(t, res, 200)
	near(t, "utilization after edit", dig(res.Body, "data", "utilization"), 50)
	if dig(res.Body, "data", "status") != "ok" {
		t.Errorf("50%% should be ok: %s", res.Raw)
	}
	// moving a budget onto an existing one is a conflict
	res = call(t, r, "PUT", path, token, map[string]interface{}{"category": "Food", "amount": "4000", "month": 2, "year": 2026})
	expectCode(t, res, 409)

	expectCode(t, call(t, r, "DELETE", path, token, nil), 200)
	expectCode(t, call(t, r, "DELETE", path, token, nil), 404)
}

// ---------------------------------------------------------------- dashboard

func TestDashboardCalculations(t *testing.T) {
	r := setup(t)
	token := register(t, r, "Asha Rao", "asha@example.com")
	// January 2026 (a fixed past period so the numbers are deterministic)
	addTx(t, r, token, "income", "Salary", "50000.00", "2026-01-05")
	addTx(t, r, token, "income", "Freelancing", "2000.50", "2026-01-10")
	addTx(t, r, token, "expense", "Food", "1200.25", "2026-01-06")
	addTx(t, r, token, "expense", "Food", "799.75", "2026-01-20")
	addTx(t, r, token, "expense", "Transport", "300.00", "2026-01-08")
	// December 2025: previous period of equal length
	addTx(t, r, token, "expense", "Food", "1000.00", "2025-12-15")
	// outside both periods
	addTx(t, r, token, "expense", "Food", "9999.00", "2026-03-01")

	res := call(t, r, "GET", "/api/dashboard/summary?range=custom&from=2026-01-01&to=2026-01-31", token, nil)
	expectCode(t, res, 200)
	d := func(k string) interface{} { return dig(res.Body, "data", k) }
	near(t, "total_income", d("total_income"), 52000.50)
	near(t, "total_expenses", d("total_expenses"), 2300.00)
	near(t, "net_savings", d("net_savings"), 49700.50)
	near(t, "savings_percentage", d("savings_percentage"), 95.58)
	near(t, "average_daily_expenditure", d("average_daily_expenditure"), 74.19) // 2300 / 31 days
	if dig(res.Body, "data", "highest_spending_category", "category") != "Food" {
		t.Errorf("highest category wrong: %s", res.Raw)
	}
	near(t, "highest total", dig(res.Body, "data", "highest_spending_category", "total"), 2000)
	near(t, "previous expenses", dig(res.Body, "data", "previous_period", "total_expenses"), 1000)
	near(t, "expense change", dig(res.Body, "data", "previous_period", "expense_change_pct"), 130) // 1000 -> 2300
	near(t, "monthly growth", d("monthly_expense_growth_pct"), 130)
	if recent, _ := d("recent_transactions").([]interface{}); len(recent) != 5 {
		t.Errorf("expected 5 recent transactions, got %d", len(recent))
	}

	// categories endpoint
	res = call(t, r, "GET", "/api/dashboard/categories?range=custom&from=2026-01-01&to=2026-01-31", token, nil)
	expectCode(t, res, 200)
	cats, _ := dig(res.Body, "data", "categories").([]interface{})
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %s", res.Raw)
	}
	near(t, "food percentage", dig(cats[0], "percentage"), 86.96)
	near(t, "transport percentage", dig(cats[1], "percentage"), 13.04)

	// trends are zero-filled and always cover the requested number of months
	res = call(t, r, "GET", "/api/dashboard/trends?months=6", token, nil)
	expectCode(t, res, 200)
	if months, _ := dig(res.Body, "data").([]interface{}); len(months) != 6 {
		t.Errorf("expected 6 trend months, got %s", res.Raw)
	}

	// empty period returns zeros and no division errors
	res = call(t, r, "GET", "/api/dashboard/summary?range=custom&from=2020-01-01&to=2020-01-31", token, nil)
	expectCode(t, res, 200)
	near(t, "empty income", dig(res.Body, "data", "total_income"), 0)
	near(t, "empty savings %", dig(res.Body, "data", "savings_percentage"), 0)
	if dig(res.Body, "data", "highest_spending_category") != nil {
		t.Error("empty period must have no highest category")
	}

	// prediction is optional: without an ML service the endpoint answers "unavailable", not an error
	res = call(t, r, "GET", "/api/dashboard/prediction", token, nil)
	expectCode(t, res, 200)
	if dig(res.Body, "data", "available") != false {
		t.Errorf("prediction should be unavailable without a service: %s", res.Raw)
	}
}

// ---------------------------------------------------------------- reports

func TestReportsFilteringAndExport(t *testing.T) {
	r := setup(t)
	token := register(t, r, "Asha Rao", "asha@example.com")
	addTx(t, r, token, "income", "Salary", "1000.00", "2026-01-05")
	addTx(t, r, token, "expense", "Food", "200.00", "2026-01-06")
	addTx(t, r, token, "expense", "Food", "300.00", "2026-02-06")
	addTx(t, r, token, "expense", "Rent", "400.00", "2025-12-06") // outside the report range
	res := call(t, r, "POST", "/api/transactions", token, map[string]interface{}{
		"type": "expense", "category": "Other", "amount": "5.00", "description": "=HYPERLINK(\"http://evil\")",
		"transaction_date": "2026-01-07", "payment_method": "Cash",
	})
	expectCode(t, res, 201)

	q := "?range=custom&from=2026-01-01&to=2026-03-31"
	res = call(t, r, "GET", "/api/reports/monthly"+q, token, nil)
	expectCode(t, res, 200)
	months, _ := dig(res.Body, "data", "months").([]interface{})
	if len(months) != 3 {
		t.Fatalf("expected Jan, Feb, Mar, got %s", res.Raw)
	}
	near(t, "jan expenses", dig(months[0], "expenses"), 205)
	near(t, "feb expenses", dig(months[1], "expenses"), 300)
	near(t, "mar expenses", dig(months[2], "expenses"), 0)
	near(t, "total expenses", dig(res.Body, "data", "totals", "expenses"), 505)
	near(t, "total net", dig(res.Body, "data", "totals", "net_savings"), 495)

	res = call(t, r, "GET", "/api/reports/categories"+q, token, nil)
	expectCode(t, res, 200)
	exp, _ := dig(res.Body, "data", "expense_categories").([]interface{})
	if len(exp) != 2 || dig(exp[0], "category") != "Food" {
		t.Errorf("category report wrong: %s", res.Raw)
	}
	near(t, "food total", dig(exp[0], "total"), 500)

	res = call(t, r, "GET", "/api/reports/daily?range=custom&from=2026-01-01&to=2026-01-31", token, nil)
	expectCode(t, res, 200)
	if days, _ := dig(res.Body, "data", "days").([]interface{}); len(days) != 3 {
		t.Errorf("expected 3 active days, got %s", res.Raw)
	}

	res = call(t, r, "GET", "/api/reports/budgets?month=1&year=2026", token, nil)
	expectCode(t, res, 200)

	// CSV export respects the date range and protects against formula injection
	res = call(t, r, "GET", "/api/reports/export?range=custom&from=2026-01-01&to=2026-01-31", token, nil)
	expectCode(t, res, 200)
	lines := strings.Split(strings.TrimSpace(strings.TrimPrefix(res.Raw, "\uFEFF")), "\n")
	if len(lines) != 4 { // header + 3 January transactions
		t.Fatalf("expected 4 CSV lines, got %d:\n%s", len(lines), res.Raw)
	}
	if !strings.HasPrefix(lines[0], "Date,Type,Category,Amount") {
		t.Errorf("bad CSV header: %s", lines[0])
	}
	if strings.Contains(res.Raw, "400.00") {
		t.Error("export contains a transaction outside the range")
	}
	if !strings.Contains(res.Raw, "'=HYPERLINK") {
		t.Error("CSV formula injection was not neutralised")
	}
}

func TestNotFoundAndHealth(t *testing.T) {
	r := setup(t)
	expectCode(t, call(t, r, "GET", "/api/health", "", nil), http.StatusOK)
	res := call(t, r, "GET", "/api/does-not-exist", "", nil)
	expectCode(t, res, 404)
	if dig(res.Body, "success") != false {
		t.Errorf("not found must use the standard error format: %s", res.Raw)
	}
}

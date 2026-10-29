package services

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestResolveRangePresets(t *testing.T) {
	now := d("2026-10-23")
	cases := []struct{ preset, from, to string }{
		{"this_month", "2026-10-01", "2026-10-31"},
		{"last_month", "2026-09-01", "2026-09-30"},
		{"last_3_months", "2026-08-01", "2026-10-31"},
	}
	for _, c := range cases {
		r, err := ResolveRange(c.preset, "", "", now)
		if err != nil || r.From != d(c.from) || r.To != d(c.to) {
			t.Errorf("%s: got %v..%v err=%v", c.preset, r.From, r.To, err)
		}
	}
	// Year boundary and leap year.
	r, _ := ResolveRange("last_month", "", "", d("2026-01-15"))
	if r.From != d("2025-12-01") || r.To != d("2025-12-31") {
		t.Errorf("year boundary wrong: %v..%v", r.From, r.To)
	}
	r, _ = ResolveRange("last_month", "", "", d("2024-03-10"))
	if r.To != d("2024-02-29") {
		t.Errorf("leap year wrong: %v", r.To)
	}
}

func TestResolveRangeCustomErrors(t *testing.T) {
	now := d("2026-10-23")
	if _, err := ResolveRange("custom", "2026-01-01", "2026-01-31", now); err != nil {
		t.Fatal(err)
	}
	bad := [][3]string{
		{"custom", "bad", "2026-01-31"},
		{"custom", "2026-01-01", "31/01/2026"},
		{"custom", "2026-02-01", "2026-01-01"},
		{"custom", "2000-01-01", "2026-01-01"},
		{"nonsense", "", ""},
	}
	for _, b := range bad {
		if _, err := ResolveRange(b[0], b[1], b[2], now); err == nil {
			t.Errorf("expected error for %v", b)
		}
	}
}

func TestPreviousRange(t *testing.T) {
	p := PreviousRange(DateRange{d("2026-01-01"), d("2026-01-31")})
	if p.From != d("2025-12-01") || p.To != d("2025-12-31") {
		t.Errorf("got %v..%v", p.From, p.To)
	}
}

func TestBudgetCalculations(t *testing.T) {
	// budget 2500.00, spent 2000.00 (minor units)
	if got := RemainingBudget(250000, 200000); got != 50000 {
		t.Errorf("remaining = %d", got)
	}
	if got := BudgetUtilization(250000, 200000); got != 80 {
		t.Errorf("utilization = %v", got)
	}
	if got := BudgetUtilization(25000, 30000); got != 120 {
		t.Errorf("utilization over = %v", got)
	}
	if got := RemainingBudget(25000, 30000); got != -5000 {
		t.Errorf("negative remaining = %d", got)
	}
	status := []struct {
		budget, spent int64
		want          string
	}{{1000, 0, "ok"}, {1000, 799, "ok"}, {1000, 800, "warning"}, {1000, 1000, "warning"}, {1000, 1001, "exceeded"}}
	for _, s := range status {
		if got := BudgetStatus(s.budget, s.spent); got != s.want {
			t.Errorf("status(%d,%d) = %s, want %s", s.budget, s.spent, got, s.want)
		}
	}
	if BudgetUtilization(0, 100) != 0 {
		t.Error("zero budget must not divide by zero")
	}
}

func TestSavingsAndChange(t *testing.T) {
	if got := SavingsRate(5200050, 230000); got != 95.58 {
		t.Errorf("savings rate = %v", got)
	}
	if SavingsRate(0, 100) != 0 {
		t.Error("no income must give 0")
	}
	if got := SavingsRate(1000, 1500); got != -50 {
		t.Errorf("negative savings rate = %v", got)
	}
	if pct, ok := PercentChange(10000, 12500); !ok || pct != 25 {
		t.Errorf("change = %v %v", pct, ok)
	}
	if _, ok := PercentChange(0, 500); ok {
		t.Error("change from zero must be undefined")
	}
	if Percentage(200000, 230000) != 86.96 {
		t.Errorf("percentage = %v", Percentage(200000, 230000))
	}
}

func TestAverageDaily(t *testing.T) {
	// Entire January 2026 is in the past: 230000 / 31 days = 7419.35 -> 7419
	r := DateRange{d("2026-01-01"), d("2026-01-31")}
	if got := AverageDaily(230000, r, d("2026-10-23")); got != 7419 {
		t.Errorf("avg = %d", got)
	}
	// Current month: only days up to today count (10 days elapsed).
	r = DateRange{d("2026-10-01"), d("2026-10-31")}
	if got := AverageDaily(100000, r, d("2026-10-10")); got != 10000 {
		t.Errorf("avg current month = %d", got)
	}
}

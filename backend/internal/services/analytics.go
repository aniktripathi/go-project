// Package services holds pure business calculations (no database access)
// so they are easy to unit-test with known inputs and outputs.
package services

import (
	"fmt"
	"math"
	"time"
)

const DateLayout = "2006-01-02"

// DateRange is an inclusive range of calendar dates (UTC).
type DateRange struct {
	From time.Time
	To   time.Time
}

// Days returns how many calendar days the range covers.
func (r DateRange) Days() int {
	return int(r.To.Sub(r.From).Hours()/24) + 1
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// EndOfMonth returns the last calendar day of the month that contains t.
func EndOfMonth(t time.Time) time.Time {
	return startOfMonth(t).AddDate(0, 1, -1)
}

// ResolveRange converts a preset name or custom dates into a DateRange.
// Presets: this_month, last_month, last_3_months (current month + the two before it), custom.
func ResolveRange(preset, from, to string, now time.Time) (DateRange, error) {
	now = now.UTC()
	switch preset {
	case "", "this_month":
		return DateRange{startOfMonth(now), EndOfMonth(now)}, nil
	case "last_month":
		prev := startOfMonth(now).AddDate(0, -1, 0)
		return DateRange{prev, EndOfMonth(prev)}, nil
	case "last_3_months":
		return DateRange{startOfMonth(now).AddDate(0, -2, 0), EndOfMonth(now)}, nil
	case "custom":
		f, err := time.Parse(DateLayout, from)
		if err != nil {
			return DateRange{}, fmt.Errorf("from must be a date in YYYY-MM-DD format")
		}
		t, err := time.Parse(DateLayout, to)
		if err != nil {
			return DateRange{}, fmt.Errorf("to must be a date in YYYY-MM-DD format")
		}
		if t.Before(f) {
			return DateRange{}, fmt.Errorf("from date must not be after the to date")
		}
		if t.Sub(f) > 5*366*24*time.Hour {
			return DateRange{}, fmt.Errorf("date range cannot be longer than 5 years")
		}
		return DateRange{f, t}, nil
	}
	return DateRange{}, fmt.Errorf("range must be one of this_month, last_month, last_3_months, custom")
}

// PreviousRange returns the range of equal length that ends the day before r starts.
func PreviousRange(r DateRange) DateRange {
	days := r.Days()
	to := r.From.AddDate(0, 0, -1)
	return DateRange{From: to.AddDate(0, 0, -(days - 1)), To: to}
}

// Round2 rounds to two decimal places (used for percentages only, never money).
func Round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// RemainingBudget = budget - spent. It is negative when the budget is exceeded.
func RemainingBudget(budget, spent int64) int64 {
	return budget - spent
}

// BudgetUtilization = spent / budget * 100.
func BudgetUtilization(budget, spent int64) float64 {
	if budget <= 0 {
		return 0
	}
	return Round2(float64(spent) / float64(budget) * 100)
}

// BudgetStatus is "exceeded" when spent > budget, "warning" from 80% upwards, otherwise "ok".
func BudgetStatus(budget, spent int64) string {
	if spent > budget {
		return "exceeded"
	}
	if budget > 0 && float64(spent)/float64(budget) >= 0.8 {
		return "warning"
	}
	return "ok"
}

// SavingsRate = (income - expenses) / income * 100. It is 0 when there is no income.
func SavingsRate(income, expenses int64) float64 {
	if income <= 0 {
		return 0
	}
	return Round2(float64(income-expenses) / float64(income) * 100)
}

// PercentChange returns the percentage change from prev to cur.
// ok is false when prev is 0 because a percentage change is undefined.
func PercentChange(prev, cur int64) (pct float64, ok bool) {
	if prev == 0 {
		return 0, false
	}
	return Round2(float64(cur-prev) / float64(prev) * 100), true
}

// Percentage returns part / total * 100 (0 when total is 0).
func Percentage(part, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return Round2(float64(part) / float64(total) * 100)
}

// AverageDaily returns total / days (rounded to the nearest minor unit).
// For a range that includes today, only the days up to today are counted.
func AverageDaily(total int64, r DateRange, now time.Time) int64 {
	end := r.To
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if today.Before(end) {
		end = today
	}
	days := int(end.Sub(r.From).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	return int64(math.Round(float64(total) / float64(days)))
}

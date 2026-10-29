package handlers

import (
	"sort"
	"time"

	"financetracker/internal/money"
	"financetracker/internal/services"
)

// All aggregate figures are calculated by PostgreSQL from the stored transactions.
// Every query is filtered by user_id and uses ? placeholders (parameterised SQL).

const dateCond = "transaction_date BETWEEN CAST(? AS DATE) AND CAST(? AS DATE)"

type periodTotals struct {
	Income   int64
	Expenses int64
}

func (h *Handler) periodTotals(uid uint, from, to time.Time) (periodTotals, error) {
	var t periodTotals
	err := h.DB.Raw(`SELECT
			COALESCE(SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END), 0)::BIGINT AS income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)::BIGINT AS expenses
		FROM transactions WHERE user_id = ? AND `+dateCond, uid, dstr(from), dstr(to)).Scan(&t).Error
	return t, err
}

type categoryRow struct {
	Category string
	Total    int64
	Count    int64
}

func (h *Handler) categoryTotals(uid uint, txType string, from, to time.Time) ([]categoryRow, error) {
	var rows []categoryRow
	err := h.DB.Raw(`SELECT category, SUM(amount)::BIGINT AS total, COUNT(*)::BIGINT AS count
		FROM transactions WHERE user_id = ? AND type = ? AND `+dateCond+`
		GROUP BY category ORDER BY total DESC, category ASC`, uid, txType, dstr(from), dstr(to)).Scan(&rows).Error
	return rows, err
}

type monthRow struct {
	Year     int
	Month    int
	Income   int64
	Expenses int64
}

func (h *Handler) monthlyTotals(uid uint, from, to time.Time) ([]monthRow, error) {
	var rows []monthRow
	err := h.DB.Raw(`SELECT
			EXTRACT(YEAR FROM transaction_date)::INT  AS year,
			EXTRACT(MONTH FROM transaction_date)::INT AS month,
			COALESCE(SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END), 0)::BIGINT AS income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)::BIGINT AS expenses
		FROM transactions WHERE user_id = ? AND `+dateCond+`
		GROUP BY 1, 2 ORDER BY 1, 2`, uid, dstr(from), dstr(to)).Scan(&rows).Error
	return rows, err
}

type dayRow struct {
	Date     string
	Income   int64
	Expenses int64
}

func (h *Handler) dailyTotals(uid uint, from, to time.Time) ([]dayRow, error) {
	var rows []dayRow
	err := h.DB.Raw(`SELECT
			TO_CHAR(transaction_date, 'YYYY-MM-DD') AS date,
			COALESCE(SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END), 0)::BIGINT AS income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)::BIGINT AS expenses
		FROM transactions WHERE user_id = ? AND `+dateCond+`
		GROUP BY transaction_date ORDER BY transaction_date`, uid, dstr(from), dstr(to)).Scan(&rows).Error
	return rows, err
}

// ---- JSON shapes --------------------------------------------------------------

// CategoryShare is one slice of a category breakdown.
type CategoryShare struct {
	Category   string      `json:"category"`
	Total      money.Money `json:"total"`
	Count      int64       `json:"count"`
	Percentage float64     `json:"percentage"`
}

func buildShares(rows []categoryRow) []CategoryShare {
	var sum int64
	for _, r := range rows {
		sum += r.Total
	}
	out := make([]CategoryShare, 0, len(rows))
	for _, r := range rows {
		out = append(out, CategoryShare{
			Category: r.Category, Total: money.Money(r.Total), Count: r.Count,
			Percentage: services.Percentage(r.Total, sum),
		})
	}
	return out
}

// MonthPoint is one calendar month of income and expenses.
type MonthPoint struct {
	Year        int         `json:"year"`
	Month       int         `json:"month"`
	Label       string      `json:"label"`
	Income      money.Money `json:"income"`
	Expenses    money.Money `json:"expenses"`
	NetSavings  money.Money `json:"net_savings"`
	SavingsRate float64     `json:"savings_rate"`
}

// fillMonths returns one MonthPoint for every month between from and to (zero when no data).
func fillMonths(rows []monthRow, from, to time.Time) []MonthPoint {
	byKey := map[int]monthRow{}
	for _, r := range rows {
		byKey[r.Year*100+r.Month] = r
	}
	var out []MonthPoint
	cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(to) {
		r := byKey[cur.Year()*100+int(cur.Month())]
		out = append(out, MonthPoint{
			Year: cur.Year(), Month: int(cur.Month()), Label: cur.Format("2006-01"),
			Income: money.Money(r.Income), Expenses: money.Money(r.Expenses),
			NetSavings:  money.Money(r.Income - r.Expenses),
			SavingsRate: services.SavingsRate(r.Income, r.Expenses),
		})
		cur = cur.AddDate(0, 1, 0)
	}
	return out
}

// BudgetView is a budget together with the real spending for its month.
type BudgetView struct {
	ID          uint        `json:"id"`
	Category    string      `json:"category"`
	Month       int         `json:"month"`
	Year        int         `json:"year"`
	Amount      money.Money `json:"amount"`
	Spent       money.Money `json:"spent"`
	Remaining   money.Money `json:"remaining"`
	Utilization float64     `json:"utilization"`
	Status      string      `json:"status"`
}

func (h *Handler) budgetViews(uid uint, month, year int) ([]BudgetView, error) {
	type budgetRow struct {
		ID       uint
		Category string
		Amount   int64
	}
	var budgets []budgetRow
	err := h.DB.Raw(`SELECT id, category, amount FROM budgets
		WHERE user_id = ? AND month = ? AND year = ? ORDER BY category ASC`, uid, month, year).Scan(&budgets).Error
	if err != nil {
		return nil, err
	}
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	spentRows, err := h.categoryTotals(uid, "expense", first, services.EndOfMonth(first))
	if err != nil {
		return nil, err
	}
	spent := map[string]int64{}
	for _, r := range spentRows {
		spent[r.Category] = r.Total
	}
	out := make([]BudgetView, 0, len(budgets))
	for _, b := range budgets {
		s := spent[b.Category]
		out = append(out, BudgetView{
			ID: b.ID, Category: b.Category, Month: month, Year: year,
			Amount: money.Money(b.Amount), Spent: money.Money(s),
			Remaining:   money.Money(services.RemainingBudget(b.Amount, s)),
			Utilization: services.BudgetUtilization(b.Amount, s),
			Status:      services.BudgetStatus(b.Amount, s),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Utilization > out[j].Utilization })
	return out, nil
}

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"financetracker/internal/models"
	"financetracker/internal/money"
	"financetracker/internal/services"

	"github.com/gin-gonic/gin"
)

func rangeJSON(r services.DateRange, preset string) gin.H {
	return gin.H{"preset": preset, "from": dstr(r.From), "to": dstr(r.To)}
}

// DashboardSummary handles GET /api/dashboard/summary?range=&from=&to=
// Net savings is income minus expenses for the period. It is NOT a bank balance.
func (h *Handler) DashboardSummary(c *gin.Context) {
	uid := userID(c)
	r, ok := rangeFromQuery(c, "this_month")
	if !ok {
		return
	}
	now := time.Now().UTC()

	cur, err := h.periodTotals(uid, r.From, r.To)
	if err != nil {
		serverError(c, err)
		return
	}
	prevRange := services.PreviousRange(r)
	prev, err := h.periodTotals(uid, prevRange.From, prevRange.To)
	if err != nil {
		serverError(c, err)
		return
	}
	catRows, err := h.categoryTotals(uid, models.TypeExpense, r.From, r.To)
	if err != nil {
		serverError(c, err)
		return
	}
	var recent []models.Transaction
	if err := h.DB.Where("user_id = ? AND transaction_date BETWEEN CAST(? AS DATE) AND CAST(? AS DATE)", uid, dstr(r.From), dstr(r.To)).
		Order("transaction_date DESC, id DESC").Limit(5).Find(&recent).Error; err != nil {
		serverError(c, err)
		return
	}

	// Budgets are monthly, so show the budgets of the month in which the range ends.
	budgets, err := h.budgetViews(uid, int(r.To.Month()), r.To.Year())
	if err != nil {
		serverError(c, err)
		return
	}

	// Month-over-month expense growth for the month in which the range ends.
	monthStart := time.Date(r.To.Year(), r.To.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevMonthStart := monthStart.AddDate(0, -1, 0)
	thisMonth, err := h.periodTotals(uid, monthStart, services.EndOfMonth(monthStart))
	if err != nil {
		serverError(c, err)
		return
	}
	lastMonth, err := h.periodTotals(uid, prevMonthStart, services.EndOfMonth(prevMonthStart))
	if err != nil {
		serverError(c, err)
		return
	}
	var growth interface{}
	if pct, ok := services.PercentChange(lastMonth.Expenses, thisMonth.Expenses); ok {
		growth = pct
	}
	var expChange, incChange interface{}
	if pct, ok := services.PercentChange(prev.Expenses, cur.Expenses); ok {
		expChange = pct
	}
	if pct, ok := services.PercentChange(prev.Income, cur.Income); ok {
		incChange = pct
	}

	var highest interface{}
	shares := buildShares(catRows)
	if len(shares) > 0 {
		highest = shares[0]
	}

	respondOK(c, http.StatusOK, gin.H{
		"range":                       rangeJSON(r, c.DefaultQuery("range", "this_month")),
		"total_income":                money.Money(cur.Income),
		"total_expenses":              money.Money(cur.Expenses),
		"net_savings":                 money.Money(cur.Income - cur.Expenses),
		"savings_percentage":          services.SavingsRate(cur.Income, cur.Expenses),
		"average_daily_expenditure":   money.Money(services.AverageDaily(cur.Expenses, r, now)),
		"highest_spending_category":   highest,
		"monthly_expense_growth_pct":  growth,
		"category_breakdown":          shares,
		"previous_period": gin.H{
			"from": dstr(prevRange.From), "to": dstr(prevRange.To),
			"total_income": money.Money(prev.Income), "total_expenses": money.Money(prev.Expenses),
			"income_change_pct": incChange, "expense_change_pct": expChange,
		},
		"recent_transactions": toDTOs(recent),
		"budget_utilization": gin.H{
			"month": int(r.To.Month()), "year": r.To.Year(), "budgets": budgets,
		},
		"note": "Net savings = income minus expenses in the selected period. It is not your bank balance.",
	})
}

// DashboardTrends handles GET /api/dashboard/trends?months=6
func (h *Handler) DashboardTrends(c *gin.Context) {
	n, err := strconv.Atoi(c.DefaultQuery("months", "6"))
	if err != nil || n < 1 || n > 24 {
		respondError(c, http.StatusBadRequest, "invalid_filter", "months must be between 1 and 24.", nil)
		return
	}
	now := time.Now().UTC()
	end := services.EndOfMonth(now)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(n - 1), 0)
	rows, err := h.monthlyTotals(userID(c), start, end)
	if err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusOK, fillMonths(rows, start, end))
}

// DashboardCategories handles GET /api/dashboard/categories?range=&type=expense
func (h *Handler) DashboardCategories(c *gin.Context) {
	r, ok := rangeFromQuery(c, "this_month")
	if !ok {
		return
	}
	txType := c.DefaultQuery("type", models.TypeExpense)
	if txType != models.TypeExpense && txType != models.TypeIncome {
		respondError(c, http.StatusBadRequest, "invalid_filter", "type must be income or expense.", nil)
		return
	}
	rows, err := h.categoryTotals(userID(c), txType, r.From, r.To)
	if err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusOK, gin.H{
		"range": rangeJSON(r, c.DefaultQuery("range", "this_month")), "type": txType, "categories": buildShares(rows),
	})
}

// DashboardPrediction handles GET /api/dashboard/prediction (optional ML feature).
// The rest of the application works even when this endpoint reports "unavailable".
func (h *Handler) DashboardPrediction(c *gin.Context) {
	unavailable := func(reason string) {
		respondOK(c, http.StatusOK, gin.H{"available": false, "reason": reason})
	}
	if h.Cfg.MLServiceURL == "" {
		unavailable("The prediction service is not configured.")
		return
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -12, 0)
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1) // completed months only
	rows, err := h.monthlyTotals(userID(c), start, end)
	if err != nil {
		serverError(c, err)
		return
	}
	var history []services.MonthPoint
	for _, p := range fillMonths(rows, start, end) {
		if len(history) == 0 && p.Expenses == 0 {
			continue // ignore empty months before the first recorded expense
		}
		history = append(history, services.MonthPoint{Month: p.Label, Total: p.Expenses.Float()})
	}
	if len(history) < 4 {
		unavailable("At least 4 completed months of expense history are needed for a prediction.")
		return
	}
	result, err := services.FetchPrediction(h.Cfg.MLServiceURL, history)
	if err != nil {
		unavailable("The prediction service is currently unavailable.")
		return
	}
	if _, present := result["available"]; !present {
		result["available"] = true
	}
	respondOK(c, http.StatusOK, result)
}

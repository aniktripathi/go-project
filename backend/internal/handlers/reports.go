package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"

	"financetracker/internal/models"
	"financetracker/internal/money"
	"financetracker/internal/services"

	"github.com/gin-gonic/gin"
)

// ReportMonthly handles GET /api/reports/monthly?range=&from=&to=
func (h *Handler) ReportMonthly(c *gin.Context) {
	r, ok := rangeFromQuery(c, "last_3_months")
	if !ok {
		return
	}
	rows, err := h.monthlyTotals(userID(c), r.From, r.To)
	if err != nil {
		serverError(c, err)
		return
	}
	var income, expenses int64
	for _, row := range rows {
		income += row.Income
		expenses += row.Expenses
	}
	respondOK(c, http.StatusOK, gin.H{
		"range":  rangeJSON(r, c.DefaultQuery("range", "last_3_months")),
		"months": fillMonths(rows, r.From, r.To),
		"totals": gin.H{
			"income": money.Money(income), "expenses": money.Money(expenses),
			"net_savings": money.Money(income - expenses), "savings_rate": services.SavingsRate(income, expenses),
		},
	})
}

// ReportCategories handles GET /api/reports/categories?range=&from=&to=
func (h *Handler) ReportCategories(c *gin.Context) {
	r, ok := rangeFromQuery(c, "this_month")
	if !ok {
		return
	}
	uid := userID(c)
	exp, err := h.categoryTotals(uid, models.TypeExpense, r.From, r.To)
	if err != nil {
		serverError(c, err)
		return
	}
	inc, err := h.categoryTotals(uid, models.TypeIncome, r.From, r.To)
	if err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusOK, gin.H{
		"range":              rangeJSON(r, c.DefaultQuery("range", "this_month")),
		"expense_categories": buildShares(exp),
		"income_categories":  buildShares(inc),
	})
}

// ReportDaily handles GET /api/reports/daily?range=&from=&to= (days that have activity)
func (h *Handler) ReportDaily(c *gin.Context) {
	r, ok := rangeFromQuery(c, "this_month")
	if !ok {
		return
	}
	rows, err := h.dailyTotals(userID(c), r.From, r.To)
	if err != nil {
		serverError(c, err)
		return
	}
	type day struct {
		Date       string      `json:"date"`
		Income     money.Money `json:"income"`
		Expenses   money.Money `json:"expenses"`
		NetSavings money.Money `json:"net_savings"`
	}
	out := make([]day, 0, len(rows))
	for _, d := range rows {
		out = append(out, day{d.Date, money.Money(d.Income), money.Money(d.Expenses), money.Money(d.Income - d.Expenses)})
	}
	respondOK(c, http.StatusOK, gin.H{"range": rangeJSON(r, c.DefaultQuery("range", "this_month")), "days": out})
}

// ReportBudgets handles GET /api/reports/budgets?month=&year= (budget performance)
func (h *Handler) ReportBudgets(c *gin.Context) {
	month, year, ok := monthYearFromQuery(c)
	if !ok {
		return
	}
	views, err := h.budgetViews(userID(c), month, year)
	if err != nil {
		serverError(c, err)
		return
	}
	var budgeted, spent int64
	exceeded := 0
	for _, v := range views {
		budgeted += int64(v.Amount)
		spent += int64(v.Spent)
		if v.Status == "exceeded" {
			exceeded++
		}
	}
	respondOK(c, http.StatusOK, gin.H{
		"month": month, "year": year, "budgets": views,
		"summary": gin.H{
			"total_budgeted": money.Money(budgeted), "total_spent": money.Money(spent),
			"remaining":   money.Money(budgeted - spent),
			"utilization": services.BudgetUtilization(budgeted, spent),
			"exceeded":    exceeded,
		},
	})
}

// csvSafe stops spreadsheet programs from running text that starts with = + - @ as a formula.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// ReportExport handles GET /api/reports/export?range=&from=&to= and returns a CSV file.
func (h *Handler) ReportExport(c *gin.Context) {
	r, ok := rangeFromQuery(c, "this_month")
	if !ok {
		return
	}
	var rows []models.Transaction
	err := h.DB.Where("user_id = ? AND transaction_date BETWEEN CAST(? AS DATE) AND CAST(? AS DATE)", userID(c), dstr(r.From), dstr(r.To)).
		Order("transaction_date ASC, id ASC").Limit(100000).Find(&rows).Error
	if err != nil {
		serverError(c, err)
		return
	}
	filename := fmt.Sprintf("transactions_%s_to_%s.csv", dstr(r.From), dstr(r.To))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.WriteString("\uFEFF") // lets Excel detect UTF-8
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"Date", "Type", "Category", "Amount", "Payment Method", "Description"})
	for _, t := range rows {
		_ = w.Write([]string{
			dstr(t.TransactionDate), t.Type, t.Category, t.Amount.String(), t.PaymentMethod, csvSafe(t.Description),
		})
	}
	w.Flush()
}

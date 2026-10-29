package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"financetracker/internal/models"
	"financetracker/internal/money"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type budgetInput struct {
	Category string    `json:"category"`
	Amount   rawAmount `json:"amount"`
	Month    int       `json:"month"`
	Year     int       `json:"year"`
}

func validateBudget(in budgetInput) (models.Budget, map[string]string) {
	details := map[string]string{}
	b := models.Budget{Category: strings.TrimSpace(in.Category), Month: in.Month, Year: in.Year}
	if !models.Contains(models.ExpenseCategories, b.Category) {
		details["category"] = "Please choose a valid expense category."
	}
	amt, err := money.Parse(string(in.Amount))
	if err != nil {
		details["amount"] = capitalise(err.Error()) + "."
	}
	b.Amount = amt
	if b.Month < 1 || b.Month > 12 {
		details["month"] = "Month must be between 1 and 12."
	}
	if b.Year < 2000 || b.Year > 2100 {
		details["year"] = "Year must be between 2000 and 2100."
	}
	return b, details
}

func monthYearFromQuery(c *gin.Context) (int, int, bool) {
	now := time.Now().UTC()
	month, err1 := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))
	year, err2 := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))
	if err1 != nil || err2 != nil || month < 1 || month > 12 || year < 2000 || year > 2100 {
		respondError(c, http.StatusBadRequest, "invalid_filter", "month must be 1-12 and year must be 2000-2100.", nil)
		return 0, 0, false
	}
	return month, year, true
}

// ListBudgets handles GET /api/budgets?month=&year= (default: current month)
func (h *Handler) ListBudgets(c *gin.Context) {
	month, year, ok := monthYearFromQuery(c)
	if !ok {
		return
	}
	views, err := h.budgetViews(userID(c), month, year)
	if err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusOK, views)
}

func (h *Handler) budgetViewByID(uid, id uint) (*BudgetView, error) {
	var b models.Budget
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&b).Error; err != nil {
		return nil, err
	}
	views, err := h.budgetViews(uid, b.Month, b.Year)
	if err != nil {
		return nil, err
	}
	for i := range views {
		if views[i].ID == b.ID {
			return &views[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// CreateBudget handles POST /api/budgets
func (h *Handler) CreateBudget(c *gin.Context) {
	var in budgetInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "The request body must be valid JSON.", nil)
		return
	}
	b, details := validateBudget(in)
	if len(details) > 0 {
		validationError(c, details)
		return
	}
	b.UserID = userID(c)

	// Check + insert inside one database transaction. The UNIQUE constraint is the final guard.
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.Budget{}).
			Where("user_id = ? AND category = ? AND month = ? AND year = ?", b.UserID, b.Category, b.Month, b.Year).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrDuplicatedKey
		}
		return tx.Create(&b).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		respondError(c, http.StatusConflict, "duplicate_budget", "A budget for this category and month already exists.", nil)
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	view, err := h.budgetViewByID(b.UserID, b.ID)
	if err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusCreated, view)
}

// UpdateBudget handles PUT /api/budgets/:id
func (h *Handler) UpdateBudget(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in budgetInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "The request body must be valid JSON.", nil)
		return
	}
	nb, details := validateBudget(in)
	if len(details) > 0 {
		validationError(c, details)
		return
	}
	uid := userID(c)

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var b models.Budget
		if err := tx.Where("id = ? AND user_id = ?", id, uid).First(&b).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&models.Budget{}).
			Where("user_id = ? AND category = ? AND month = ? AND year = ? AND id <> ?", uid, nb.Category, nb.Month, nb.Year, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrDuplicatedKey
		}
		b.Category, b.Amount, b.Month, b.Year = nb.Category, nb.Amount, nb.Month, nb.Year
		return tx.Save(&b).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "Budget not found.", nil)
		return
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		respondError(c, http.StatusConflict, "duplicate_budget", "A budget for this category and month already exists.", nil)
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	view, err := h.budgetViewByID(uid, id)
	if err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusOK, view)
}

// DeleteBudget handles DELETE /api/budgets/:id
func (h *Handler) DeleteBudget(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	res := h.DB.Where("id = ? AND user_id = ?", id, userID(c)).Delete(&models.Budget{})
	if res.Error != nil {
		serverError(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "not_found", "Budget not found.", nil)
		return
	}
	respondOK(c, http.StatusOK, gin.H{"message": "Budget deleted."})
}

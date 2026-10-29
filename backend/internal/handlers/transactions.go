package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"financetracker/internal/models"
	"financetracker/internal/money"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TransactionDTO is the JSON shape returned to the frontend.
type TransactionDTO struct {
	ID              uint        `json:"id"`
	UserID          uint        `json:"user_id"`
	Type            string      `json:"type"`
	Category        string      `json:"category"`
	Amount          money.Money `json:"amount"`
	Description     string      `json:"description"`
	TransactionDate string      `json:"transaction_date"`
	PaymentMethod   string      `json:"payment_method"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

func toDTO(t models.Transaction) TransactionDTO {
	return TransactionDTO{
		ID: t.ID, UserID: t.UserID, Type: t.Type, Category: t.Category, Amount: t.Amount,
		Description: t.Description, TransactionDate: dstr(t.TransactionDate),
		PaymentMethod: t.PaymentMethod, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toDTOs(rows []models.Transaction) []TransactionDTO {
	out := make([]TransactionDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDTO(r))
	}
	return out
}

// txInput is the request body for create and update.
// Amount is read as raw text so that "12.5" and 12.5 are both accepted and exact.
type txInput struct {
	Type            string `json:"type"`
	Category        string `json:"category"`
	Amount          rawAmount `json:"amount"`
	Description     string `json:"description"`
	TransactionDate string `json:"transaction_date"`
	PaymentMethod   string `json:"payment_method"`
}

// rawAmount keeps the exact characters the client sent (number or string).
type rawAmount string

func (r *rawAmount) UnmarshalJSON(b []byte) error {
	*r = rawAmount(strings.Trim(string(b), `"`))
	return nil
}

type validatedTx struct {
	Type, Category, Description, PaymentMethod string
	Amount                                     money.Money
	Date                                       time.Time
}

func validateTx(in txInput) (validatedTx, map[string]string) {
	details := map[string]string{}
	v := validatedTx{
		Type:        strings.ToLower(strings.TrimSpace(in.Type)),
		Category:    strings.TrimSpace(in.Category),
		Description: strings.TrimSpace(in.Description),
	}
	if v.Type != models.TypeIncome && v.Type != models.TypeExpense {
		details["type"] = "Type must be income or expense."
	} else if !models.Contains(models.CategoriesFor(v.Type), v.Category) {
		details["category"] = "Please choose a valid category for this type."
	}

	amt, err := money.Parse(string(in.Amount))
	if err != nil {
		details["amount"] = capitalise(err.Error()) + "."
	}
	v.Amount = amt

	if utf8.RuneCountInString(v.Description) > 255 {
		details["description"] = "Description must be at most 255 characters."
	}

	date, err := parseDate(strings.TrimSpace(in.TransactionDate))
	switch {
	case err != nil:
		details["transaction_date"] = "Date must be in YYYY-MM-DD format."
	case date.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)):
		details["transaction_date"] = "Date cannot be before 2000-01-01."
	case date.After(time.Now().UTC().AddDate(1, 0, 0)):
		details["transaction_date"] = "Date cannot be more than one year in the future."
	}
	v.Date = date

	if v.Type == models.TypeExpense {
		v.PaymentMethod = strings.TrimSpace(in.PaymentMethod)
		if !models.Contains(models.PaymentMethods, v.PaymentMethod) {
			details["payment_method"] = "Please choose a valid payment method."
		}
	} // payment method is not applicable to income and is stored as empty text

	return v, details
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// bindTx reads and validates the JSON body.
func bindTx(c *gin.Context) (validatedTx, bool) {
	var in txInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "The request body must be valid JSON.", nil)
		return validatedTx{}, false
	}
	v, details := validateTx(in)
	if len(details) > 0 {
		validationError(c, details)
		return validatedTx{}, false
	}
	return v, true
}

// CreateTransaction handles POST /api/transactions
func (h *Handler) CreateTransaction(c *gin.Context) {
	v, ok := bindTx(c)
	if !ok {
		return
	}
	tx := models.Transaction{
		UserID: userID(c), Type: v.Type, Category: v.Category, Amount: v.Amount,
		Description: v.Description, TransactionDate: v.Date, PaymentMethod: v.PaymentMethod,
	}
	if err := h.DB.Create(&tx).Error; err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusCreated, toDTO(tx))
}

// findOwned loads a transaction only if it belongs to the logged-in user.
// Records of other users look exactly like missing records (404).
func (h *Handler) findOwned(c *gin.Context) (models.Transaction, bool) {
	var tx models.Transaction
	id, ok := parseID(c)
	if !ok {
		return tx, false
	}
	err := h.DB.Where("id = ? AND user_id = ?", id, userID(c)).First(&tx).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "Transaction not found.", nil)
		return tx, false
	}
	if err != nil {
		serverError(c, err)
		return tx, false
	}
	return tx, true
}

// GetTransaction handles GET /api/transactions/:id
func (h *Handler) GetTransaction(c *gin.Context) {
	if tx, ok := h.findOwned(c); ok {
		respondOK(c, http.StatusOK, toDTO(tx))
	}
}

// UpdateTransaction handles PUT /api/transactions/:id
func (h *Handler) UpdateTransaction(c *gin.Context) {
	tx, ok := h.findOwned(c)
	if !ok {
		return
	}
	v, ok := bindTx(c)
	if !ok {
		return
	}
	tx.Type, tx.Category, tx.Amount = v.Type, v.Category, v.Amount
	tx.Description, tx.TransactionDate, tx.PaymentMethod = v.Description, v.Date, v.PaymentMethod
	if err := h.DB.Save(&tx).Error; err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusOK, toDTO(tx))
}

// DeleteTransaction handles DELETE /api/transactions/:id
func (h *Handler) DeleteTransaction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	res := h.DB.Where("id = ? AND user_id = ?", id, userID(c)).Delete(&models.Transaction{})
	if res.Error != nil {
		serverError(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "not_found", "Transaction not found.", nil)
		return
	}
	respondOK(c, http.StatusOK, gin.H{"message": "Transaction deleted."})
}

// PageMeta describes pagination for list endpoints.
type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	return strings.ReplaceAll(s, `_`, `\_`)
}

// listTransactions is shared by /transactions, /income and /expenses.
// forcedType is "" for all transactions, otherwise "income" or "expense".
func (h *Handler) listTransactions(c *gin.Context, forcedType string) {
	uid := userID(c)

	txType := forcedType
	if txType == "" {
		txType = strings.ToLower(c.Query("type"))
		if txType != "" && txType != models.TypeIncome && txType != models.TypeExpense {
			respondError(c, http.StatusBadRequest, "invalid_filter", "type must be income or expense.", nil)
			return
		}
	}
	category := strings.TrimSpace(c.Query("category"))
	method := strings.TrimSpace(c.Query("payment_method"))
	search := strings.TrimSpace(c.Query("search"))

	var from, to string
	if f := c.Query("from"); f != "" {
		if _, err := parseDate(f); err != nil {
			respondError(c, http.StatusBadRequest, "invalid_filter", "from must be a date in YYYY-MM-DD format.", nil)
			return
		}
		from = f
	}
	if t := c.Query("to"); t != "" {
		if _, err := parseDate(t); err != nil {
			respondError(c, http.StatusBadRequest, "invalid_filter", "to must be a date in YYYY-MM-DD format.", nil)
			return
		}
		to = t
	}
	if from != "" && to != "" && from > to {
		respondError(c, http.StatusBadRequest, "invalid_filter", "from date must not be after the to date.", nil)
		return
	}

	sortCol := "transaction_date"
	switch c.DefaultQuery("sort", "date") {
	case "date":
	case "amount":
		sortCol = "amount"
	default:
		respondError(c, http.StatusBadRequest, "invalid_filter", "sort must be date or amount.", nil)
		return
	}
	order := strings.ToLower(c.DefaultQuery("order", "desc"))
	if order != "asc" && order != "desc" {
		respondError(c, http.StatusBadRequest, "invalid_filter", "order must be asc or desc.", nil)
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		respondError(c, http.StatusBadRequest, "invalid_filter", "page must be a positive whole number.", nil)
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		respondError(c, http.StatusBadRequest, "invalid_filter", "page_size must be between 1 and 100.", nil)
		return
	}

	// build() returns a fresh query each time so Count and Find do not interfere.
	build := func() *gorm.DB {
		q := h.DB.Model(&models.Transaction{}).Where("user_id = ?", uid)
		if txType != "" {
			q = q.Where("type = ?", txType)
		}
		if category != "" {
			q = q.Where("category = ?", category)
		}
		if method != "" {
			q = q.Where("payment_method = ?", method)
		}
		if from != "" {
			q = q.Where("transaction_date >= CAST(? AS DATE)", from)
		}
		if to != "" {
			q = q.Where("transaction_date <= CAST(? AS DATE)", to)
		}
		if search != "" {
			like := "%" + escapeLike(search) + "%"
			q = q.Where("(description ILIKE ? OR category ILIKE ?)", like, like)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		serverError(c, err)
		return
	}
	var rows []models.Transaction
	err = build().Order(sortCol + " " + order + ", id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	if err != nil {
		serverError(c, err)
		return
	}
	respondList(c, toDTOs(rows), PageMeta{
		Page: page, PageSize: pageSize, Total: total,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	})
}

// ListTransactions handles GET /api/transactions
func (h *Handler) ListTransactions(c *gin.Context) { h.listTransactions(c, "") }

// ListIncome handles GET /api/income
func (h *Handler) ListIncome(c *gin.Context) { h.listTransactions(c, models.TypeIncome) }

// ListExpenses handles GET /api/expenses
func (h *Handler) ListExpenses(c *gin.Context) { h.listTransactions(c, models.TypeExpense) }

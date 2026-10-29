// Package handlers contains the HTTP endpoints (controllers).
package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"financetracker/config"
	"financetracker/internal/middleware"
	"financetracker/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler groups all endpoints and the dependencies they share.
type Handler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

// New creates a Handler.
func New(db *gorm.DB, cfg *config.Config) *Handler {
	return &Handler{DB: db, Cfg: cfg}
}

// ---- consistent JSON responses ------------------------------------------------
// Success: {"success": true, "data": ..., "meta": ...}
// Failure: {"success": false, "error": {"code": "...", "message": "...", "details": {...}}}

func respondOK(c *gin.Context, status int, data interface{}) {
	c.JSON(status, gin.H{"success": true, "data": data})
}

func respondList(c *gin.Context, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "meta": meta})
}

func respondError(c *gin.Context, status int, code, message string, details map[string]string) {
	body := gin.H{"code": code, "message": message}
	if len(details) > 0 {
		body["details"] = details
	}
	c.JSON(status, gin.H{"success": false, "error": body})
}

func validationError(c *gin.Context, details map[string]string) {
	respondError(c, http.StatusBadRequest, "validation_failed", "Please correct the highlighted fields.", details)
}

// serverError logs the real error on the server but gives the user a generic message.
func serverError(c *gin.Context, err error) {
	log.Printf("internal error on %s %s: %v", c.Request.Method, c.FullPath(), err)
	respondError(c, http.StatusInternalServerError, "internal_error", "Something went wrong on our side. Please try again later.", nil)
}

// NotFound answers unknown API routes.
func NotFound(c *gin.Context) {
	respondError(c, http.StatusNotFound, "not_found", "The requested resource was not found.", nil)
}

// Health reports whether the API and database are reachable.
func (h *Handler) Health(c *gin.Context) {
	sqlDB, err := h.DB.DB()
	if err == nil {
		err = sqlDB.Ping()
	}
	if err != nil {
		respondError(c, http.StatusServiceUnavailable, "database_unavailable", "The database is not reachable.", nil)
		return
	}
	respondOK(c, http.StatusOK, gin.H{"status": "ok"})
}

// ---- small helpers --------------------------------------------------------------

// userID returns the authenticated user's id that the Auth middleware stored.
func userID(c *gin.Context) uint {
	v, _ := c.Get(middleware.CtxUserID)
	id, _ := v.(uint)
	return id
}

func parseID(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || n == 0 {
		respondError(c, http.StatusBadRequest, "invalid_id", "The id in the URL is not valid.", nil)
		return 0, false
	}
	return uint(n), true
}

func parseDate(s string) (time.Time, error) {
	return time.Parse(services.DateLayout, s)
}

func dstr(t time.Time) string { return t.Format(services.DateLayout) }

// rangeFromQuery reads ?range=&from=&to= and turns it into a DateRange.
func rangeFromQuery(c *gin.Context, defaultPreset string) (services.DateRange, bool) {
	preset := c.DefaultQuery("range", defaultPreset)
	r, err := services.ResolveRange(preset, c.Query("from"), c.Query("to"), time.Now())
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_filter", err.Error(), nil)
		return services.DateRange{}, false
	}
	return r, true
}

func (h *Handler) Categories(c *gin.Context) {
	respondOK(c, http.StatusOK, gin.H{
		"income_categories":  modelsIncome(),
		"expense_categories": modelsExpense(),
		"payment_methods":    modelsPayments(),
	})
}

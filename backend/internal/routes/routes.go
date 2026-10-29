// Package routes wires URLs to handlers and middleware.
package routes

import (
	"time"

	"financetracker/config"
	"financetracker/internal/handlers"
	"financetracker/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter builds the Gin engine with every API route.
func SetupRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	_ = r.SetTrustedProxies(nil) // use the direct connection address for rate limiting

	r.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:       12 * time.Hour,
	}))
	r.Use(middleware.BodyLimit(1 << 20)) // 1 MB

	h := handlers.New(db, cfg)
	api := r.Group("/api")
	api.GET("/health", h.Health)

	limiter := middleware.RateLimit(cfg.AuthRateLimitPerMin)
	api.POST("/auth/register", limiter, h.Register)
	api.POST("/auth/login", limiter, h.Login)

	p := api.Group("")
	p.Use(middleware.Auth(cfg.JWTSecret))

	p.POST("/auth/logout", h.Logout)
	p.GET("/auth/me", h.Me)
	p.PUT("/auth/me", h.UpdateMe)
	p.GET("/meta/categories", h.Categories)

	p.GET("/transactions", h.ListTransactions)
	p.POST("/transactions", h.CreateTransaction)
	p.GET("/transactions/:id", h.GetTransaction)
	p.PUT("/transactions/:id", h.UpdateTransaction)
	p.DELETE("/transactions/:id", h.DeleteTransaction)
	p.GET("/income", h.ListIncome)
	p.GET("/expenses", h.ListExpenses)

	p.GET("/budgets", h.ListBudgets)
	p.POST("/budgets", h.CreateBudget)
	p.PUT("/budgets/:id", h.UpdateBudget)
	p.DELETE("/budgets/:id", h.DeleteBudget)

	p.GET("/dashboard/summary", h.DashboardSummary)
	p.GET("/dashboard/trends", h.DashboardTrends)
	p.GET("/dashboard/categories", h.DashboardCategories)
	p.GET("/dashboard/prediction", h.DashboardPrediction)

	p.GET("/reports/monthly", h.ReportMonthly)
	p.GET("/reports/categories", h.ReportCategories)
	p.GET("/reports/daily", h.ReportDaily)
	p.GET("/reports/budgets", h.ReportBudgets)
	p.GET("/reports/export", h.ReportExport)

	r.NoRoute(handlers.NotFound)
	return r
}

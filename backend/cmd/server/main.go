// Command server starts the Smart Personal Finance and Expense Tracker API.
package main

import (
	"log"
	"net/http"
	"time"

	"financetracker/config"
	"financetracker/internal/database"
	"financetracker/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	gin.SetMode(cfg.GinMode)

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migration error: %v", err)
	}
	log.Println("database ready, migrations applied")

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.SetupRouter(db, cfg),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("API listening on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

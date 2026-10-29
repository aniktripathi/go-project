// Command seed fills the database with a demo account and sample data.
//
//	go run ./cmd/seed
//
// Demo login: demo@example.com / Demo@12345  (for local demos only)
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"financetracker/config"
	"financetracker/internal/auth"
	"financetracker/internal/database"
	"financetracker/internal/models"
	"financetracker/internal/money"

	"gorm.io/gorm"
)

const demoEmail = "demo@example.com"
const demoPassword = "Demo@12345"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	var existing int64
	db.Model(&models.User{}).Where("email = ?", demoEmail).Count(&existing)
	if existing > 0 {
		fmt.Println("Demo user already exists. Nothing to do.")
		return
	}

	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		log.Fatal(err)
	}
	rng := rand.New(rand.NewSource(42)) // fixed seed: the same demo data every time

	err = db.Transaction(func(tx *gorm.DB) error {
		user := models.User{Name: "Demo User", Email: demoEmail, PasswordHash: hash}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

		add := func(t, cat string, rupees int, desc, method string, date time.Time) error {
			return tx.Create(&models.Transaction{
				UserID: user.ID, Type: t, Category: cat, Amount: money.Money(rupees * 100),
				Description: desc, PaymentMethod: method, TransactionDate: date,
			}).Error
		}

		for m := 5; m >= 0; m-- {
			start := thisMonth.AddDate(0, -m, 0)
			last := start.AddDate(0, 1, -1)
			lastDay := last.Day()
			if m == 0 && now.Day() < lastDay {
				lastDay = now.Day() // do not create future transactions this month
			}
			day := func(d int) time.Time {
				if d > lastDay {
					d = lastDay
				}
				return start.AddDate(0, 0, d-1)
			}
			if err := add("income", "Salary", 45000, "Monthly salary", "", day(1)); err != nil {
				return err
			}
			if m%2 == 0 {
				if err := add("income", "Freelancing", 6000+rng.Intn(4000), "Freelance project", "", day(15)); err != nil {
					return err
				}
			}
			if err := add("expense", "Rent", 12000, "House rent", "Net Banking", day(2)); err != nil {
				return err
			}
			if err := add("expense", "Bills", 1500+rng.Intn(800), "Electricity and internet", "UPI", day(8)); err != nil {
				return err
			}
			for i := 0; i < 8; i++ {
				if err := add("expense", "Food", 150+rng.Intn(500), "Groceries / meals", []string{"UPI", "Cash", "Card"}[rng.Intn(3)], day(3+i*3)); err != nil {
					return err
				}
			}
			for i := 0; i < 5; i++ {
				if err := add("expense", "Transport", 50+rng.Intn(300), "Metro / cab", []string{"UPI", "Cash"}[rng.Intn(2)], day(4+i*5)); err != nil {
					return err
				}
			}
			if err := add("expense", "Shopping", 800+rng.Intn(2500), "Clothes / online order", "Card", day(18)); err != nil {
				return err
			}
			if err := add("expense", "Entertainment", 300+rng.Intn(900), "Movies / subscriptions", "Card", day(22)); err != nil {
				return err
			}
			if m%3 == 0 {
				if err := add("expense", "Healthcare", 400+rng.Intn(1200), "Pharmacy", "Cash", day(12)); err != nil {
					return err
				}
			}
		}

		budgets := map[string]int{"Food": 5000, "Transport": 1500, "Shopping": 2500, "Entertainment": 1200, "Rent": 12000, "Bills": 2000}
		for cat, amt := range budgets {
			b := models.Budget{UserID: user.ID, Category: cat, Amount: money.Money(amt * 100), Month: int(now.Month()), Year: now.Year()}
			if err := tx.Create(&b).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("seeding failed: %v", err)
	}
	fmt.Printf("Demo data created. Log in with %s / %s\n", demoEmail, demoPassword)
}

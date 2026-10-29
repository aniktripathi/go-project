// Package models describes the database tables as Go structs.
package models

import (
	"time"

	"financetracker/internal/money"
)

// User is a registered person. PasswordHash is never sent in JSON.
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Transaction is one income or expense record that belongs to a single user.
type Transaction struct {
	ID              uint
	UserID          uint
	Type            string
	Category        string
	Amount          money.Money
	Description     string
	TransactionDate time.Time
	PaymentMethod   string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Budget is a monthly spending limit for one expense category.
type Budget struct {
	ID        uint
	UserID    uint
	Category  string
	Amount    money.Money
	Month     int
	Year      int
	CreatedAt time.Time
	UpdatedAt time.Time
}

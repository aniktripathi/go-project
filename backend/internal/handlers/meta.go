package handlers

import "financetracker/internal/models"

func modelsIncome() []string   { return models.IncomeCategories }
func modelsExpense() []string  { return models.ExpenseCategories }
func modelsPayments() []string { return models.PaymentMethods }

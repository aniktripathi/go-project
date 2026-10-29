package models

// Allowed values. The backend validates every request against these lists.
var (
	IncomeCategories  = []string{"Salary", "Freelancing", "Scholarship", "Business", "Other"}
	ExpenseCategories = []string{"Food", "Transport", "Education", "Shopping", "Entertainment", "Healthcare", "Bills", "Rent", "Other"}
	PaymentMethods    = []string{"Cash", "Card", "UPI", "Net Banking", "Other"}
)

const (
	TypeIncome  = "income"
	TypeExpense = "expense"
)

// Contains reports whether value is in list (case-sensitive).
func Contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

// CategoriesFor returns the allowed categories for a transaction type.
func CategoriesFor(txType string) []string {
	if txType == TypeIncome {
		return IncomeCategories
	}
	return ExpenseCategories
}

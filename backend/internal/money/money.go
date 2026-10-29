// Package money represents currency amounts as whole minor units (paise)
// so that financial calculations never suffer from floating point errors.
package money

import (
	"fmt"
	"strconv"
	"strings"
)

// Money is an amount in minor units. 1234.50 is Money(123450).
type Money int64

// MaxMinor is the largest amount accepted for a single record: 999,999,999.99.
const MaxMinor Money = 99999999999

// Parse converts text such as "1234.5" or "1234.50" into Money.
// It rejects negative numbers, zero, more than two decimals and exponent notation.
func Parse(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("amount is required")
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("amount must be a valid number")
	}
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if whole == "" && frac == "" {
		return 0, fmt.Errorf("amount must be a valid number")
	}
	if whole == "" {
		whole = "0"
	}
	if !allDigits(whole) || (frac != "" && !allDigits(frac)) {
		return 0, fmt.Errorf("amount must be a positive number")
	}
	if len(frac) > 2 {
		return 0, fmt.Errorf("amount can have at most 2 decimal places")
	}
	if len(whole) > 12 {
		return 0, fmt.Errorf("amount is too large")
	}
	for len(frac) < 2 {
		frac += "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount must be a valid number")
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount must be a valid number")
	}
	v := Money(w*100 + f)
	if v <= 0 {
		return 0, fmt.Errorf("amount must be greater than zero")
	}
	if v > MaxMinor {
		return 0, fmt.Errorf("amount is too large")
	}
	return v, nil
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// String formats the amount with exactly two decimals, e.g. "1234.50" or "-50.00".
func (m Money) String() string {
	v := int64(m)
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

// MarshalJSON writes the amount as a JSON number such as 1234.50 (no float rounding).
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalJSON accepts either a JSON number or a quoted decimal string.
func (m *Money) UnmarshalJSON(b []byte) error {
	v, err := Parse(strings.Trim(string(b), `"`))
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// Float returns the amount in major units for display-only uses (e.g. ML input).
func (m Money) Float() float64 {
	return float64(m) / 100
}

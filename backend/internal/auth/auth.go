// Package auth handles password hashing and JWT creation / validation.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims is the data stored inside a login token.
type Claims struct {
	UserID uint `json:"uid"`
	jwt.RegisteredClaims
}

// HashPassword returns a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword compares a plain password with a stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken creates a signed HS256 token that expires after ttl.
func GenerateToken(secret string, userID uint, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "finance-tracker",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ErrExpired is returned when a token is valid but out of date.
var ErrExpired = fmt.Errorf("token expired")

// ParseToken validates the signature, algorithm and expiry of a token.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		if isExpired(err) {
			return nil, ErrExpired
		}
		return nil, fmt.Errorf("invalid token")
	}
	if !token.Valid || claims.UserID == 0 {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func isExpired(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}

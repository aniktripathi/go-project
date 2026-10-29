// Package middleware contains Gin handlers that run before the real endpoint.
package middleware

import (
	"net/http"
	"strings"

	"financetracker/internal/auth"

	"github.com/gin-gonic/gin"
)

// CtxUserID is the Gin context key that stores the authenticated user's id.
const CtxUserID = "userID"

func abort(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error":   gin.H{"code": code, "message": message},
	})
}

// Auth requires "Authorization: Bearer <token>" and stores the user id in the context.
// The user id always comes from the validated token, never from the request body.
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			abort(c, "unauthorized", "Please log in to continue.")
			return
		}
		claims, err := auth.ParseToken(secret, strings.TrimSpace(parts[1]))
		if err != nil {
			if err == auth.ErrExpired {
				abort(c, "token_expired", "Your session has expired. Please log in again.")
				return
			}
			abort(c, "unauthorized", "Please log in to continue.")
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Next()
	}
}

// BodyLimit rejects request bodies larger than max bytes.
func BodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		c.Next()
	}
}

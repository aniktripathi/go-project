package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit allows at most perMinute requests per minute for each client IP.
// A value of 0 disables limiting. Use it on login and register routes.
func RateLimit(perMinute int) gin.HandlerFunc {
	if perMinute <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	var mu sync.Mutex
	visitors := map[string]*visitor{}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		v, ok := visitors[ip]
		if !ok {
			v = &visitor{limiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(perMinute)), perMinute)}
			visitors[ip] = v
		}
		v.lastSeen = time.Now()
		// Occasionally drop clients we have not seen for an hour so the map cannot grow forever.
		if len(visitors) > 5000 {
			for k, vv := range visitors {
				if time.Since(vv.lastSeen) > time.Hour {
					delete(visitors, k)
				}
			}
		}
		allowed := v.limiter.Allow()
		mu.Unlock()

		if !allowed {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   gin.H{"code": "rate_limited", "message": "Too many attempts. Please wait a minute and try again."},
			})
			return
		}
		c.Next()
	}
}

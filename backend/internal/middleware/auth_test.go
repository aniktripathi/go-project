package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financetracker/internal/auth"

	"github.com/gin-gonic/gin"
)

const secret = "0123456789abcdef0123456789abcdef"

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/private", Auth(secret), func(c *gin.Context) {
		id, _ := c.Get(CtxUserID)
		c.JSON(200, gin.H{"id": id})
	})
	return r
}

func call(r *gin.Engine, header string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware(t *testing.T) {
	r := newRouter()
	good, _ := auth.GenerateToken(secret, 7, time.Hour)
	expired, _ := auth.GenerateToken(secret, 7, -time.Minute)
	foreign, _ := auth.GenerateToken("a-completely-different-secret-value!!", 7, time.Hour)

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", 401},
		{"wrong scheme", "Basic abc", 401},
		{"garbage token", "Bearer abc.def.ghi", 401},
		{"expired token", "Bearer " + expired, 401},
		{"wrong secret", "Bearer " + foreign, 401},
		{"valid token", "Bearer " + good, 200},
	}
	for _, tc := range cases {
		if got := call(r, tc.header).Code; got != tc.want {
			t.Errorf("%s: status %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", RateLimit(3), func(c *gin.Context) { c.Status(200) })
	codes := []int{}
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		codes = append(codes, w.Code)
	}
	if codes[0] != 200 || codes[2] != 200 || codes[4] != 429 {
		t.Fatalf("unexpected codes %v", codes)
	}
}

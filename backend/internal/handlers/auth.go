package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"financetracker/internal/auth"
	"financetracker/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// A real bcrypt hash is compared even when the e-mail does not exist, so response
// time does not reveal which e-mails are registered.
var dummyHash, _ = auth.HashPassword("not-a-real-password-1")

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func validEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	at := strings.LastIndex(email, "@")
	return at > 0 && strings.Contains(email[at+1:], ".")
}

func validPassword(p string) string {
	if len(p) < 8 {
		return "Password must be at least 8 characters."
	}
	if len(p) > 72 {
		return "Password must be at most 72 characters."
	}
	hasLetter, hasDigit := false, false
	for _, r := range p {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "Password must contain at least one letter and one number."
	}
	return ""
}

func validName(n string) string {
	if l := utf8.RuneCountInString(n); l < 2 || l > 100 {
		return "Name must be between 2 and 100 characters."
	}
	return ""
}

func (h *Handler) tokenResponse(c *gin.Context, status int, user models.User) {
	token, err := auth.GenerateToken(h.Cfg.JWTSecret, user.ID, time.Duration(h.Cfg.JWTExpiryHours)*time.Hour)
	if err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, status, gin.H{"token": token, "user": user})
}

// Register creates a new account. POST /api/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "The request body must be valid JSON.", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	details := map[string]string{}
	if msg := validName(req.Name); msg != "" {
		details["name"] = msg
	}
	if !validEmail(req.Email) {
		details["email"] = "Please enter a valid e-mail address."
	}
	if msg := validPassword(req.Password); msg != "" {
		details["password"] = msg
	}
	if len(details) > 0 {
		validationError(c, details)
		return
	}

	var count int64
	if err := h.DB.Model(&models.User{}).Where("LOWER(email) = ?", req.Email).Count(&count).Error; err != nil {
		serverError(c, err)
		return
	}
	if count > 0 {
		respondError(c, http.StatusConflict, "email_taken", "An account with this e-mail already exists.", nil)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		serverError(c, err)
		return
	}
	user := models.User{Name: req.Name, Email: req.Email, PasswordHash: hash}
	if err := h.DB.Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) { // two sign-ups at the same moment
			respondError(c, http.StatusConflict, "email_taken", "An account with this e-mail already exists.", nil)
			return
		}
		serverError(c, err)
		return
	}
	h.tokenResponse(c, http.StatusCreated, user)
}

// Login checks the credentials and returns a token. POST /api/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "The request body must be valid JSON.", nil)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		validationError(c, map[string]string{"email": "E-mail and password are required."})
		return
	}

	var user models.User
	err := h.DB.Where("LOWER(email) = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		auth.CheckPassword(dummyHash, req.Password) // keep timing similar
		respondError(c, http.StatusUnauthorized, "invalid_credentials", "Incorrect e-mail or password.", nil)
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		respondError(c, http.StatusUnauthorized, "invalid_credentials", "Incorrect e-mail or password.", nil)
		return
	}
	h.tokenResponse(c, http.StatusOK, user)
}

// Logout is stateless: the client discards its token. POST /api/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	respondOK(c, http.StatusOK, gin.H{"message": "Logged out. Please discard your token."})
}

func (h *Handler) currentUser(c *gin.Context) (models.User, bool) {
	var user models.User
	err := h.DB.First(&user, userID(c)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, http.StatusUnauthorized, "unauthorized", "Please log in to continue.", nil)
		return user, false
	}
	if err != nil {
		serverError(c, err)
		return user, false
	}
	return user, true
}

// Me returns the logged-in user. GET /api/auth/me
func (h *Handler) Me(c *gin.Context) {
	if user, ok := h.currentUser(c); ok {
		respondOK(c, http.StatusOK, user)
	}
}

// UpdateMe changes the user's display name. PUT /api/auth/me
func (h *Handler) UpdateMe(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "The request body must be valid JSON.", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if msg := validName(req.Name); msg != "" {
		validationError(c, map[string]string{"name": msg})
		return
	}
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	user.Name = req.Name
	if err := h.DB.Save(&user).Error; err != nil {
		serverError(c, err)
		return
	}
	respondOK(c, http.StatusOK, user)
}

package handlers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"icecream-api/internal/auth"
	"icecream-api/internal/middleware"
	"icecream-api/internal/models"
	"icecream-api/internal/response"
	"icecream-api/internal/validation"
)

type AuthHandler struct {
	db     *gorm.DB
	tokens *auth.TokenManager
}

func NewAuthHandler(db *gorm.DB, tm *auth.TokenManager) *AuthHandler {
	return &AuthHandler{db: db, tokens: tm}
}

type loginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// used so a missing user costs the same as a wrong password (timing side channel)
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var in loginInput
	if err := bind(c, &in); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid JSON body")
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if errs := validation.Struct(in); errs != nil {
		return response.Validation(c, errs)
	}

	var user models.User
	err := h.db.Where("email = ?", in.Email).First(&user).Error
	if err != nil {
		if !isNotFound(err) {
			return err
		}
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password))
		return response.Error(c, fiber.StatusUnauthorized, "invalid email or password")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		return response.Error(c, fiber.StatusUnauthorized, "invalid email or password")
	}

	token, exp, err := h.tokens.Generate(&user)
	if err != nil {
		return err
	}
	return response.Data(c, fiber.StatusOK, fiber.Map{
		"token":      token,
		"token_type": "Bearer",
		"expires_at": exp,
		"user":       user,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	claims := middleware.CurrentClaims(c)

	// housekeeping: drop revoked tokens that have expired anyway
	_ = h.db.Where("expires_at < ?", time.Now()).Delete(&models.RevokedToken{}).Error

	rt := models.RevokedToken{ID: claims.ID, ExpiresAt: claims.ExpiresAt.Time}
	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rt).Error; err != nil {
		return err
	}
	return response.Data(c, fiber.StatusOK, fiber.Map{"message": "logged out"})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	return response.Data(c, fiber.StatusOK, middleware.CurrentUser(c))
}

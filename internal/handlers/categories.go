package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"icecream-api/internal/models"
	"icecream-api/internal/response"
)

type CategoryHandler struct{ db *gorm.DB }

func NewCategoryHandler(db *gorm.DB) *CategoryHandler { return &CategoryHandler{db: db} }

// List is read-only; it lets clients discover valid category_id values.
func (h *CategoryHandler) List(c *fiber.Ctx) error {
	var cats []models.Category
	if err := h.db.Order("name ASC").Find(&cats).Error; err != nil {
		return err
	}
	return response.Data(c, fiber.StatusOK, cats)
}

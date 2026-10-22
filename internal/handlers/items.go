package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"icecream-api/internal/models"
	"icecream-api/internal/response"
	"icecream-api/internal/validation"
)

type ItemHandler struct{ db *gorm.DB }

func NewItemHandler(db *gorm.DB) *ItemHandler { return &ItemHandler{db: db} }

type itemInput struct {
	Name        string  `json:"name" validate:"required,min=1,max=150"`
	Description string  `json:"description" validate:"max=1000"`
	CategoryID  uint    `json:"category_id" validate:"required"`
	Price       float64 `json:"price" validate:"gte=0"`
	Quantity    int     `json:"quantity" validate:"gte=0"`
}

// itemPatch uses pointers so PUT/PATCH can update only the fields that were sent.
type itemPatch struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	CategoryID  *uint    `json:"category_id"`
	Price       *float64 `json:"price"`
	Quantity    *int     `json:"quantity"`
}

func (h *ItemHandler) categoryExists(id uint) (bool, error) {
	var n int64
	err := h.db.Model(&models.Category{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (h *ItemHandler) List(c *fiber.Ctx) error {
	page, limit := pagination(c)

	q := h.db.Model(&models.Item{})
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		q = q.Where(`LOWER(items.name) LIKE ? ESCAPE '\'`, likePattern(s))
	}
	if cid := c.QueryInt("category_id", 0); cid > 0 {
		q = q.Where("items.category_id = ?", cid)
	}
	q = q.Session(&gorm.Session{}) // make the query safely reusable

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return err
	}
	var items []models.Item
	if err := q.Preload("Category").Order("items.id ASC").
		Limit(limit).Offset((page - 1) * limit).Find(&items).Error; err != nil {
		return err
	}
	return response.List(c, items, newMeta(page, limit, total))
}

func (h *ItemHandler) Get(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid item id")
	}
	var item models.Item
	if err := h.db.Preload("Category").First(&item, id).Error; err != nil {
		if isNotFound(err) {
			return response.Error(c, fiber.StatusNotFound, "item not found")
		}
		return err
	}
	return response.Data(c, fiber.StatusOK, item)
}

func (h *ItemHandler) Create(c *fiber.Ctx) error {
	var in itemInput
	if err := bind(c, &in); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid JSON body")
	}
	in.Name = strings.TrimSpace(in.Name)
	if errs := validation.Struct(in); errs != nil {
		return response.Validation(c, errs)
	}
	ok, err := h.categoryExists(in.CategoryID)
	if err != nil {
		return err
	}
	if !ok {
		return response.Validation(c, map[string]string{"category_id": "category does not exist"})
	}

	item := models.Item{
		Name: in.Name, Description: in.Description, CategoryID: in.CategoryID,
		Price: in.Price, Quantity: in.Quantity,
	}
	if err := h.db.Create(&item).Error; err != nil {
		return err
	}
	if err := h.db.Preload("Category").First(&item, item.ID).Error; err != nil {
		return err
	}
	return response.Data(c, fiber.StatusCreated, item)
}

// Update handles both PUT and PATCH (partial update of the supplied fields).
func (h *ItemHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid item id")
	}
	var item models.Item
	if err := h.db.First(&item, id).Error; err != nil {
		if isNotFound(err) {
			return response.Error(c, fiber.StatusNotFound, "item not found")
		}
		return err
	}

	var p itemPatch
	if err := bind(c, &p); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid JSON body")
	}
	categoryChanged := false
	if p.Name != nil {
		item.Name = strings.TrimSpace(*p.Name)
	}
	if p.Description != nil {
		item.Description = *p.Description
	}
	if p.CategoryID != nil {
		categoryChanged = *p.CategoryID != item.CategoryID
		item.CategoryID = *p.CategoryID
	}
	if p.Price != nil {
		item.Price = *p.Price
	}
	if p.Quantity != nil {
		item.Quantity = *p.Quantity
	}

	// validate the final merged state
	merged := itemInput{
		Name: item.Name, Description: item.Description, CategoryID: item.CategoryID,
		Price: item.Price, Quantity: item.Quantity,
	}
	if errs := validation.Struct(merged); errs != nil {
		return response.Validation(c, errs)
	}
	if categoryChanged {
		ok, err := h.categoryExists(item.CategoryID)
		if err != nil {
			return err
		}
		if !ok {
			return response.Validation(c, map[string]string{"category_id": "category does not exist"})
		}
	}

	if err := h.db.Save(&item).Error; err != nil {
		return err
	}
	if err := h.db.Preload("Category").First(&item, item.ID).Error; err != nil {
		return err
	}
	return response.Data(c, fiber.StatusOK, item)
}

func (h *ItemHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid item id")
	}
	res := h.db.Delete(&models.Item{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "item not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

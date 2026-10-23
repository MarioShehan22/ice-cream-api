package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"icecream-api/internal/models"
	"icecream-api/internal/response"
	"icecream-api/internal/validation"
)

type CustomerHandler struct{ db *gorm.DB }

func NewCustomerHandler(db *gorm.DB) *CustomerHandler { return &CustomerHandler{db: db} }

type customerInput struct {
	Name    string `json:"name" validate:"required,min=1,max=100"`
	Email   string `json:"email" validate:"required,email,max=150"`
	Phone   string `json:"phone" validate:"required,min=7,max=20"`
	Address string `json:"address" validate:"max=255"`
}

type customerPatch struct {
	Name    *string `json:"name"`
	Email   *string `json:"email"`
	Phone   *string `json:"phone"`
	Address *string `json:"address"`
}

func (h *CustomerHandler) emailTaken(email string, excludeID uint) (bool, error) {
	var n int64
	err := h.db.Model(&models.Customer{}).Where("email = ? AND id <> ?", email, excludeID).Count(&n).Error
	return n > 0, err
}

func (h *CustomerHandler) List(c *fiber.Ctx) error {
	page, limit := pagination(c)

	q := h.db.Model(&models.Customer{})
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		p := likePattern(s)
		q = q.Where(`LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR phone LIKE ?`, p, p, p)
	}
	q = q.Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return err
	}
	var customers []models.Customer
	if err := q.Order("id ASC").Limit(limit).Offset((page - 1) * limit).Find(&customers).Error; err != nil {
		return err
	}
	return response.List(c, customers, newMeta(page, limit, total))
}

func (h *CustomerHandler) Get(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid customer id")
	}
	var cu models.Customer
	if err := h.db.First(&cu, id).Error; err != nil {
		if isNotFound(err) {
			return response.Error(c, fiber.StatusNotFound, "customer not found")
		}
		return err
	}
	return response.Data(c, fiber.StatusOK, cu)
}

func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	var in customerInput
	if err := bind(c, &in); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid JSON body")
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	if errs := validation.Struct(in); errs != nil {
		return response.Validation(c, errs)
	}
	taken, err := h.emailTaken(in.Email, 0)
	if err != nil {
		return err
	}
	if taken {
		return response.Error(c, fiber.StatusConflict, "a customer with this email already exists")
	}

	cu := models.Customer{Name: in.Name, Email: in.Email, Phone: in.Phone, Address: in.Address}
	if err := h.db.Create(&cu).Error; err != nil {
		return err
	}
	return response.Data(c, fiber.StatusCreated, cu)
}

// Update handles both PUT and PATCH (partial update of the supplied fields).
func (h *CustomerHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid customer id")
	}
	var cu models.Customer
	if err := h.db.First(&cu, id).Error; err != nil {
		if isNotFound(err) {
			return response.Error(c, fiber.StatusNotFound, "customer not found")
		}
		return err
	}

	var p customerPatch
	if err := bind(c, &p); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid JSON body")
	}
	if p.Name != nil {
		cu.Name = strings.TrimSpace(*p.Name)
	}
	if p.Email != nil {
		cu.Email = strings.ToLower(strings.TrimSpace(*p.Email))
	}
	if p.Phone != nil {
		cu.Phone = strings.TrimSpace(*p.Phone)
	}
	if p.Address != nil {
		cu.Address = *p.Address
	}

	merged := customerInput{Name: cu.Name, Email: cu.Email, Phone: cu.Phone, Address: cu.Address}
	if errs := validation.Struct(merged); errs != nil {
		return response.Validation(c, errs)
	}
	taken, err := h.emailTaken(cu.Email, cu.ID)
	if err != nil {
		return err
	}
	if taken {
		return response.Error(c, fiber.StatusConflict, "a customer with this email already exists")
	}

	if err := h.db.Save(&cu).Error; err != nil {
		return err
	}
	return response.Data(c, fiber.StatusOK, cu)
}

func (h *CustomerHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid customer id")
	}
	res := h.db.Delete(&models.Customer{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "customer not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

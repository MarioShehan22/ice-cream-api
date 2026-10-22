package response

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
)

type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func Data(c *fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(fiber.Map{"data": data})
}

func List(c *fiber.Ctx, data any, meta Meta) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": data, "meta": meta})
}

func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"message": message}})
}

func Validation(c *fiber.Ctx, details map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
		"error": fiber.Map{"message": "validation failed", "details": details},
	})
}

// ErrorHandler is Fiber's global error handler (panics, 404s, unexpected errors).
func ErrorHandler(c *fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return Error(c, fe.Code, fe.Message)
	}
	log.Printf("unhandled error: %v", err)
	return Error(c, fiber.StatusInternalServerError, "internal server error")
}

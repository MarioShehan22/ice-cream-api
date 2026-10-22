package handlers

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"icecream-api/internal/response"
)

func parseID(c *fiber.Ctx) (uint, error) {
	n, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || n == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(n), nil
}

func bind(c *fiber.Ctx, dst any) error {
	return json.Unmarshal(c.Body(), dst)
}

func pagination(c *fiber.Ctx) (page, limit int) {
	page = c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit = c.QueryInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return
}

func newMeta(page, limit int, total int64) response.Meta {
	return response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(limit))),
	}
}

func isNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// likePattern builds a case-insensitive "contains" pattern (use with ESCAPE '\').
func likePattern(s string) string {
	return "%" + likeEscaper.Replace(strings.ToLower(strings.TrimSpace(s))) + "%"
}

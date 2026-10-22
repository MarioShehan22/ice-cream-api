package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"icecream-api/internal/auth"
	"icecream-api/internal/models"
	"icecream-api/internal/response"
)

const (
	localUser   = "auth_user"
	localClaims = "auth_claims"
)

func CurrentUser(c *fiber.Ctx) *models.User {
	u, _ := c.Locals(localUser).(*models.User)
	return u
}

func CurrentClaims(c *fiber.Ctx) *auth.Claims {
	cl, _ := c.Locals(localClaims).(*auth.Claims)
	return cl
}

// Authenticate validates the Bearer JWT, rejects revoked (logged-out) tokens,
// and loads the user fresh from the DB so role changes apply immediately.
func Authenticate(tm *auth.TokenManager, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		parts := strings.SplitN(c.Get(fiber.HeaderAuthorization), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return response.Error(c, fiber.StatusUnauthorized, "missing or malformed Authorization header")
		}
		claims, err := tm.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			return response.Error(c, fiber.StatusUnauthorized, "invalid or expired token")
		}

		var revoked int64
		if err := db.Model(&models.RevokedToken{}).Where("id = ?", claims.ID).Count(&revoked).Error; err != nil {
			return err
		}
		if revoked > 0 {
			return response.Error(c, fiber.StatusUnauthorized, "token has been revoked")
		}

		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return response.Error(c, fiber.StatusUnauthorized, "user no longer exists")
			}
			return err
		}

		c.Locals(localUser, &user)
		c.Locals(localClaims, claims)
		return c.Next()
	}
}

// RequireRoles allows the request only if the authenticated user has one of the roles.
func RequireRoles(roles ...models.Role) fiber.Handler {
	allowed := make(map[models.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		u := CurrentUser(c)
		if u == nil {
			return response.Error(c, fiber.StatusUnauthorized, "authentication required")
		}
		if _, ok := allowed[u.Role]; !ok {
			return response.Error(c, fiber.StatusForbidden, "your role ("+string(u.Role)+") is not allowed to perform this action")
		}
		return c.Next()
	}
}

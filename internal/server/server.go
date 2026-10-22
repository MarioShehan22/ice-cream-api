package server

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"icecream-api/internal/auth"
	"icecream-api/internal/handlers"
	"icecream-api/internal/middleware"
	"icecream-api/internal/models"
	"icecream-api/internal/response"
)

// New builds the Fiber app. Extra middleware (e.g. a request logger) can be passed in.
func New(db *gorm.DB, tm *auth.TokenManager, extra ...fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Ice Cream Shop API",
		ErrorHandler: response.ErrorHandler,
	})
	app.Use(recover.New())
	for _, m := range extra {
		app.Use(m)
	}

	const (
		owner   = models.RoleOwner
		manager = models.RoleManager
	)

	authn := middleware.Authenticate(tm, db)
	ownerOrManager := middleware.RequireRoles(owner, manager)
	ownerOnly := middleware.RequireRoles(owner)

	authH := handlers.NewAuthHandler(db, tm)
	itemH := handlers.NewItemHandler(db)
	customerH := handlers.NewCustomerHandler(db)
	categoryH := handlers.NewCategoryHandler(db)

	loginLimiter := limiter.New(limiter.Config{
		Max:        20,
		Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests, "too many login attempts, try again later")
		},
	})

	api := app.Group("/api")
	api.Get("/health", func(c *fiber.Ctx) error {
		return response.Data(c, fiber.StatusOK, fiber.Map{"status": "ok"})
	})

	// Authentication
	api.Post("/auth/login", loginLimiter, authH.Login)
	api.Post("/auth/logout", authn, authH.Logout)
	api.Get("/auth/me", authn, authH.Me)

	// Categories (flavours) - read only, any authenticated user
	api.Get("/categories", authn, categoryH.List)

	// Inventory:
	//   view: all roles | create: owner, manager | edit: all roles | delete: owner
	items := api.Group("/items", authn)
	items.Get("/", itemH.List)
	items.Get("/:id", itemH.Get)
	items.Post("/", ownerOrManager, itemH.Create)
	items.Put("/:id", itemH.Update)
	items.Patch("/:id", itemH.Update)
	items.Delete("/:id", ownerOnly, itemH.Delete)

	// Customers:
	//   view / create / edit: all roles | delete: owner
	customers := api.Group("/customers", authn)
	customers.Get("/", customerH.List)
	customers.Get("/:id", customerH.Get)
	customers.Post("/", customerH.Create)
	customers.Put("/:id", customerH.Update)
	customers.Patch("/:id", customerH.Update)
	customers.Delete("/:id", ownerOnly, customerH.Delete)

	return app
}

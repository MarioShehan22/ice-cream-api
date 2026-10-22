package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2/middleware/logger"
	"golang.org/x/crypto/bcrypt"

	"icecream-api/internal/auth"
	"icecream-api/internal/config"
	"icecream-api/internal/database"
	"icecream-api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if cfg.SeedOnStart {
		if err := database.Seed(db, bcrypt.DefaultCost); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	tm := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	app := server.New(db, tm, logger.New())

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	_ = app.ShutdownWithTimeout(10 * time.Second)
}

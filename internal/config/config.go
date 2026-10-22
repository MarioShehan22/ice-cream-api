package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseDSN string
	JWTSecret   string
	JWTTTL      time.Duration
	SeedOnStart bool
}

func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional

	cfg := &Config{
		Port:        get("PORT", "8080"),
		DatabaseDSN: get("DATABASE_DSN", "root:password@tcp(localhost:3306)/ice_cream?parseTime=true"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		SeedOnStart: get("SEED_ON_START", "true") == "true",
	}
	if len(cfg.JWTSecret) < 16 {
		return nil, errors.New("JWT_SECRET must be set and at least 16 characters long")
	}
	hours, err := strconv.Atoi(get("JWT_TTL_HOURS", "24"))
	if err != nil || hours <= 0 {
		return nil, errors.New("JWT_TTL_HOURS must be a positive integer")
	}
	cfg.JWTTTL = time.Duration(hours) * time.Hour
	return cfg, nil
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

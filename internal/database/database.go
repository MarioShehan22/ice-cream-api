package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"icecream-api/internal/models"
)

// Connect opens a MySQL connection. The DSN format is:
//
//	user:password@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
//
// It retries for ~30s so the API can start while MySQL (e.g. in Docker) is still booting.
func Connect(dsn string) (*gorm.DB, error) {
	cfg := &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			SlowThreshold:             300 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	}

	var (
		db  *gorm.DB
		err error
	)
	for attempt := 1; attempt <= 15; attempt++ {
		db, err = gorm.Open(mysql.Open(dsn), cfg)
		if err == nil {
			var sqlDB *sql.DB
			if sqlDB, err = db.DB(); err == nil {
				err = sqlDB.Ping()
			}
		}
		if err == nil {
			break
		}
		log.Printf("database not ready (attempt %d/15): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("could not connect to MySQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Item{},
		&models.Customer{},
		&models.RevokedToken{},
	)
}

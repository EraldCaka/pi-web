package db

import (
	"fmt"
	"net/url"

	"github.com/EraldCaka/pi-web/internal/config"
	"github.com/EraldCaka/pi-web/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Path:   "/" + cfg.DBName,
	}
	if cfg.Password != "" {
		u.User = url.UserPassword(cfg.User, cfg.Password)
	} else {
		u.User = url.User(cfg.User)
	}
	q := u.Query()
	q.Set("sslmode", cfg.SSLMode)
	q.Set("TimeZone", "UTC")
	u.RawQuery = q.Encode()
	dsn := u.String()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

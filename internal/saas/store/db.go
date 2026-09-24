package store

import (
	"fmt"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps gorm.DB for the SaaS data layer.
type DB struct {
	*gorm.DB
}

// NewDB opens Postgres, runs AutoMigrate on all models, returns wrapper.
// Schema truth source is Go structs only — no SQL migration files.
func NewDB(cfg *config.Config, log *zap.Logger) (*DB, error) {
	gormLogger := logger.Default.LogMode(logger.Warn)
	if cfg.AppRole == config.RoleDev {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	gdb, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := gdb.AutoMigrate(AllModels()...); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	if log != nil {
		log.Info("database migrated", zap.String("dbname", cfg.Database.DBName))
	}
	return &DB{DB: gdb}, nil
}

func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

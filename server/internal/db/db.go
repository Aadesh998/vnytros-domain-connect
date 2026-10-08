package db

import (
	"domain-connect-backend/internal/config"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

var DB *gorm.DB

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
}

func GetDBConfig() DBConfig {
	return DBConfig{
		User:     config.AppConfig.DBUser,
		Password: config.AppConfig.DBPassword,
		Host:     config.AppConfig.DBHost,
		Port:     config.AppConfig.DBPort,
		DBName:   config.AppConfig.DBName,
	}
}

func GetDSN() string {
	cfg := GetDBConfig()
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port,
	)
	return dsn
}

func InitDB() {
	dsn := GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL DB:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get sql.DB from gorm:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Trace every query on this pool. One pool serves cmd/server (api and
	// mailforge), cmd/mcp and cmd/worker, so this is the only place it needs
	// installing.
	//
	// WithoutQueryVariables is NOT optional: by default the plugin inlines
	// bound parameters into the db.statement attribute, which would ship SMTP
	// passwords (mail_smtp_configs.Password), API keys and password hashes to
	// the trace backend in plain text. The statement shape is what's useful for
	// debugging; the values are not worth the exposure.
	//
	// WithoutMetrics because metrics come from Prometheus (internal/telemetry),
	// not from an OTel meter provider we do not configure.
	if err := db.Use(tracing.NewPlugin(
		tracing.WithoutQueryVariables(),
		tracing.WithoutMetrics(),
	)); err != nil {
		// Diagnostics must never stop the process from starting.
		log.Printf("db: WARN gorm tracing disabled: %v", err)
	}

	DB = db
}

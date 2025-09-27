package db

import (
	"backend/internal/telemetry"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func InitDBConnection() (*sqlx.DB, error) {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "user:password@tcp(db:3306)/42Tokyo2508-db"
	}
	
	// Add optimization parameters to DSN
	dsn := fmt.Sprintf("%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s&maxAllowedPacket=0&interpolateParams=true", dbUrl)
	log.Printf("Database DSN: %s", dsn)

	driverName := telemetry.WrapSQLDriver("mysql")
	dbConn, err := sqlx.Open(driverName, dsn)
	if err != nil {
		log.Printf("Failed to open database connection: %v", err)
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = dbConn.PingContext(ctx)
	if err != nil {
		dbConn.Close()
		log.Printf("Failed to connect to database: %v", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Successfully connected to MySQL!")

	// Optimize connection pool settings
	// Increase MaxOpenConns to 50 for better parallel request handling
	dbConn.SetMaxOpenConns(50)
	// Increase MaxIdleConns to 20 for better connection reuse
	dbConn.SetMaxIdleConns(20)
	// Set connection lifetime to 5 minutes to prevent MySQL connection issues
	dbConn.SetConnMaxLifetime(5 * time.Minute)
	// Set maximum connection idle time
	dbConn.SetConnMaxIdleTime(10 * time.Minute)

	return dbConn, nil
}
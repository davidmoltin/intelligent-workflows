package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/davidmoltin/intelligent-workflows/internal/seeds"
)

var (
	dbHost     = getEnv("DB_HOST", "localhost")
	dbPort     = getEnv("DB_PORT", "5432")
	dbUser     = getEnv("DB_USER", "postgres")
	dbPassword = getEnv("DB_PASSWORD", "postgres")
	dbName     = getEnv("DB_NAME", "workflows")
	dbSSLMode  = getEnv("DB_SSL_MODE", "disable")
)

func main() {
	// Parse command line flags
	var (
		seedUsers   = flag.Bool("users", false, "Seed default users (admin user)")
		verifyOnly  = flag.Bool("verify", false, "Only verify existing RBAC data, don't seed")
		statsOnly   = flag.Bool("stats", false, "Only show RBAC statistics")
		force       = flag.Bool("force", false, "Force re-seeding (updates existing data)")
	)
	flag.Parse()

	// Setup logging
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[RBAC Seed] ")

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	log.Println("Connecting to database...")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Ping database to verify connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✓ Database connection established")

	// Create seeder
	seeder := seeds.NewRBACSeeder(db)

	// Handle different modes
	switch {
	case *statsOnly:
		// Show statistics only
		if err := seeder.Stats(ctx); err != nil {
			log.Fatalf("Failed to get RBAC statistics: %v", err)
		}

	case *verifyOnly:
		// Verify only
		log.Println("\n=== Verification Mode ===")
		if err := seeder.Verify(ctx); err != nil {
			log.Fatalf("Verification failed: %v", err)
			os.Exit(1)
		}
		log.Println("\n✓ All RBAC data verified successfully!")

	default:
		// Seed mode (default)
		log.Println("\n=== Seeding Mode ===")
		if !*force {
			log.Println("Note: Existing data will be preserved (use --force to update)")
		}

		if err := seeder.SeedAll(ctx, *seedUsers); err != nil {
			log.Fatalf("Seeding failed: %v", err)
		}

		// Show stats after seeding
		fmt.Println()
		if err := seeder.Stats(ctx); err != nil {
			log.Printf("Warning: Failed to get statistics: %v", err)
		}

		// Run verification
		fmt.Println()
		log.Println("Running post-seed verification...")
		if err := seeder.Verify(ctx); err != nil {
			log.Fatalf("Post-seed verification failed: %v", err)
		}

		// Show success message
		fmt.Println("\n" + strings.Repeat("=", 60))
		log.Println("✓✓✓ RBAC seeding completed successfully! ✓✓✓")
		fmt.Println(strings.Repeat("=", 60))

		if *seedUsers {
			fmt.Println("\n⚠️  DEFAULT ADMIN CREDENTIALS:")
			fmt.Println("   Username: admin")
			fmt.Println("   Email:    admin@example.com")
			fmt.Println("   Password: admin123")
			fmt.Println("\n⚠️  IMPORTANT: Change the default password after first login!")
			fmt.Println(strings.Repeat("=", 60))
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package main

import (
	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/db"
	"domain-connect-backend/internal/db/migrations"
	"domain-connect-backend/internal/models"
	"log"

	"gorm.io/gorm"
)

func main() {
	config.LoadConfig()

	db.InitDB()
	sqlDB, err := db.DB.DB()
	if err != nil {
		log.Fatalf("Failed to get SQL DB: %v", err)
	}

	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("error closing DB: %v", err)
		}
	}()

	if err := dropBillingSchema(); err != nil {
		log.Fatalf("drop billing schema: %v", err)
	}

	if err := db.DB.AutoMigrate(
		&models.Users{},
		&models.Providers{},
		&models.Domains{},
		&models.ApiKeys{},
		&models.OAuthClient{},
		&models.OAuthAuthRequest{},
		&models.OAuthAuthCode{},
		&models.MailTemplate{},
		&models.MailSmtpConfig{},
		&models.MailCampaign{},
		&models.MailRecipient{},
		&models.MailTrack{},
		&models.MailBranding{},
	); err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully All models migrated.")

	// DNS provider reference data for provider detection. The SQL upserts by
	// id, so re-running the migrator just refreshes it.
	if err := db.DB.Exec(migrations.DNSProviders).Error; err != nil {
		log.Fatalf("load dns providers: %v", err)
	}
	log.Println("DNS provider data loaded.")
}

// billingCleanup removes the schema left behind by the paid plans, top-ups
// and subscriptions that Vnytros no longer has. Every statement is
// idempotent, so it is a no-op on a fresh install and safe to re-run.
//
// Order matters: users.plan_id carries the foreign key to plans, so the
// column goes first (Postgres drops its constraint with it), then the tables.
var billingCleanup = []string{
	`ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS plan_id`,
	`ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS total_count`,
	`ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS domain_count`,
	`DROP TABLE IF EXISTS domain_topups`,
	`DROP TABLE IF EXISTS subscriptions`,
	`DROP TABLE IF EXISTS topup_packs`,
	`DROP TABLE IF EXISTS plans CASCADE`,
}

func dropBillingSchema() error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range billingCleanup {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

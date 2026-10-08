// Command apikey creates an API key for a user without going through the
// dashboard, stores it in the database exactly like the dashboard does, and
// writes it to a local env file so tools like the SDK playground can load it.
//
//	go run ./cmd/apikey -email you@example.com
//	go run ./cmd/apikey -email you@example.com -create -name "Local Dev"
//
// It reads the same env file as the server (.env.production, or ENV_FILE), so
// the key is signed with this deployment's APIKEY_SIGNING_SECRET.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/db"
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/service"
)

func main() {
	email := flag.String("email", "", "email of the account that will own the key (required)")
	webhook := flag.String("webhook", "http://localhost:9000/webhook", "URL that receives this key's domain events")
	create := flag.Bool("create", false, "create the account (already verified) if it does not exist")
	name := flag.String("name", "", "display name when -create makes a new account (default: part of the email before @)")
	out := flag.String("out", "keys/vnytros-api-key.env", "file to save the key to; empty to skip saving")
	flag.Parse()

	*email = strings.ToLower(strings.TrimSpace(*email))
	if *email == "" {
		flag.Usage()
		os.Exit(2)
	}

	config.LoadConfig()
	db.InitDB()
	if sqlDB, err := db.DB.DB(); err == nil {
		defer sqlDB.Close()
	}

	userRepo := repository.NewUserRepository(db.DB)
	user, err := userRepo.GetByEmail(*email)
	if err != nil {
		log.Fatalf("look up %s: %v", *email, err)
	}
	if user == nil {
		if !*create {
			log.Fatalf("no account for %s: sign up in the dashboard first, or rerun with -create", *email)
		}
		password, err := createUser(userRepo, *email, *name)
		if err != nil {
			log.Fatalf("create account: %v", err)
		}
		fmt.Printf("Created verified account %s\nPassword (shown once): %s\n\n", *email, password)
		if user, err = userRepo.GetByEmail(*email); err != nil || user == nil {
			log.Fatalf("reload new account: %v", err)
		}
	}

	auth := service.NewAuthService(userRepo, repository.NewApiKeyRepository(db.DB), repository.NewDomainRepository(db.DB))
	key, err := auth.CreateAPIKey(user.ID, *webhook)
	if err != nil {
		log.Fatalf("create api key: %v", err)
	}

	fmt.Printf("API key for %s (saved in the database):\n\n%s\n\n", user.Email, key)

	if *out != "" {
		if err := saveKey(*out, key); err != nil {
			log.Fatalf("save key: %v", err)
		}
		fmt.Printf("Also saved to %s. Load it with:\n  set -a; source %s; set +a\n", *out, *out)
	}
}

// createUser inserts a verified account with a random password, so a local
// deployment without SMTP can still get a key.
func createUser(repo repository.UserRepository, email, name string) (string, error) {
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	password := base64.RawURLEncoding.EncodeToString(buf)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	if err := repo.Create(&models.Users{
		Name:     name,
		Email:    email,
		Password: string(hash),
		UserType: "user",
		Verified: true,
	}); err != nil {
		return "", err
	}
	return password, nil
}

// saveKey writes the key as env lines, readable only by the current user.
func saveKey(path, key string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	baseURL := config.AppConfig.BaseURL
	if baseURL == "" {
		return errors.New("BASE_URL is not configured")
	}
	body := fmt.Sprintf("VNYTROS_API_KEY=%s\nVNYTROS_BASE_URL=%s\n", key, baseURL)
	return os.WriteFile(path, []byte(body), 0o600)
}

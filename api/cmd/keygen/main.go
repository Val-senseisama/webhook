// keygen creates a new API key for a tenant and prints the raw key once.
// The raw key is never stored — only its SHA-256 hash is written to the DB.
//
// Usage:
//
//	DATABASE_DIRECT_URL=<url> go run ./cmd/keygen \
//	  --tenant 00000000-0000-0000-0000-000000000001 \
//	  --name "my-dashboard-key"
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"webhook/internal/db"
	"webhook/internal/signing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	tenantFlag := flag.String("tenant", "00000000-0000-0000-0000-000000000001", "tenant UUID")
	nameFlag := flag.String("name", "default", "human-readable key name")
	flag.Parse()

	tenantID, err := uuid.Parse(*tenantFlag)
	if err != nil {
		slog.Error("invalid tenant UUID", "error", err)
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_DIRECT_URL")
	if dsn == "" {
		slog.Error("DATABASE_DIRECT_URL not set")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("connect", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		slog.Error("generate key", "error", err)
		os.Exit(1)
	}
	rawKey := "whk_" + hex.EncodeToString(rawBytes)
	keyHash := signing.HashAPIKey(rawKey)

	database := db.New(pool)
	key, err := database.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		TenantID: tenantID,
		Name:     *nameFlag,
		KeyHash:  keyHash,
	})
	if err != nil {
		slog.Error("insert api key", "error", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ API key created\n\n")
	fmt.Printf("  Key ID   : %s\n", key.ID)
	fmt.Printf("  Name     : %s\n", key.Name)
	fmt.Printf("  Tenant   : %s\n", key.TenantID)
	fmt.Printf("  Raw key  : %s\n\n", rawKey)
	fmt.Printf("  ⚠  This is the only time the raw key is shown. Store it now.\n\n")
	fmt.Printf("  Usage:\n")
	fmt.Printf("    Authorization: Bearer %s\n\n", rawKey)
}

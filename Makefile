.PHONY: api worker dev-api dev-worker build test tidy

## Run API server locally (sources .env automatically)
dev-api:
	set -a && . ./.env && set +a && cd api && go run ./cmd/api

## Run worker locally (sources .env automatically)
dev-worker:
	set -a && . ./.env && set +a && cd api && go run ./cmd/worker

## Run dashboard locally
dev-dash:
	cd dashboard && npm run dev

## Build both binaries
build:
	cd api && go build -o ../bin/api ./cmd/api && go build -o ../bin/worker ./cmd/worker

## Tidy Go modules
tidy:
	cd api && go mod tidy

## Run tests
test:
	cd api && go test ./...

## Install dashboard deps
install:
	cd dashboard && npm install

## Build dashboard
build-dash:
	cd dashboard && npm run build

## Docker compose up
up:
	docker compose up --build

## Apply DB migrations manually (uses goose CLI)
migrate:
	cd api && goose -dir migrations postgres "$$DATABASE_DIRECT_URL" up

## ---- Supabase ----

## Login to Supabase (browser-based)
sb-login:
	supabase login

## Link this project to a Supabase project (run once)
## Usage: make sb-link PROJECT_REF=abcdefghijklmnop
sb-link:
	supabase link --project-ref $(PROJECT_REF)

## Push all migrations to the linked Supabase project
sb-push:
	supabase db push

## Pull remote schema changes back (if you edited via Supabase dashboard)
sb-pull:
	supabase db pull

## Open Supabase Studio for the linked project
sb-studio:
	supabase studio

## Generate and register a new API key
## Usage: make keygen NAME="my-key"
keygen:
	set -a && . ./.env && set +a && cd api && go run ./cmd/keygen --name "$(or $(NAME),default)"

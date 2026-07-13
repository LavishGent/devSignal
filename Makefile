# Developer entry points. See docs/MAINTENANCE_GUIDE.md for details.

.PHONY: db-up db-down api web test lint integration build docker

db-up: ## start PostgreSQL only
	docker compose up -d db

db-down: ## stop everything, keep data
	docker compose down

api: ## run the Go API (migrates on startup)
	cd backend && go run ./cmd/api

web: ## run the frontend dev server on :5173
	cd frontend && npm run dev

test: ## unit + handler tests, both halves
	cd backend && go vet ./... && go test -count=1 -cover ./...
	cd frontend && npm run build

lint:
	cd backend && golangci-lint run ./...
	cd frontend && npx oxlint src

integration: ## full-stack tests against real PostgreSQL (needs db-up)
	cd backend && TEST_DATABASE_URL=postgres://devsignal:devsignal@localhost:5432/devsignal?sslmode=disable \
		go test -count=1 -tags=integration ./internal/integration/ -v

build: ## production build of both halves
	cd frontend && npm run build
	cd backend && go build -o bin/devsignal ./cmd/api

docker: ## production container image
	docker build -t devsignal .

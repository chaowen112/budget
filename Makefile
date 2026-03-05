.PHONY: proto build run test clean docker-up docker-down migrate-up migrate-down ui

# Protobuf generation
proto:
	buf generate api/proto
	cp api/openapi/budget.swagger.json cmd/server/openapi/

# Build the application
build: ui
	go build -o bin/server ./cmd/server

# Build only Go binary (no UI)
build-go:
	go build -o bin/server ./cmd/server

# Build frontend UI
ui:
	cd web && npm run build

# Install frontend dependencies
ui-deps:
	cd web && npm install

# Run frontend dev server
ui-dev:
	cd web && npm run dev

# Run the application locally
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/ gen/

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f api

# Database migrations
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $$name

# Development helpers
deps:
	go mod tidy
	go mod download

lint:
	golangci-lint run

# Install development tools
tools:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

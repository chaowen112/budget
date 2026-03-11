# Proto / Swagger generation stage
FROM bufbuild/buf:latest AS proto-builder
WORKDIR /app
COPY buf.gen.yaml ./
COPY api/proto/ ./api/proto/
RUN buf generate api/proto

# Frontend build stage
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web

RUN npm i -g pnpm

# Copy package files
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy frontend source and build
COPY web/ ./
RUN pnpm run build

# Go build stage
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build from previous stage
COPY --from=frontend-builder /app/cmd/server/static ./cmd/server/static

# Copy generated swagger spec from proto stage
COPY --from=proto-builder /app/api/openapi/budget.swagger.json ./cmd/server/openapi/budget.swagger.json

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from builder
COPY --from=go-builder /app/server .

# Copy static files (frontend)
COPY --from=go-builder /app/cmd/server/static ./static

# Expose ports
EXPOSE 8080 9090

# Run the application
CMD ["./server"]

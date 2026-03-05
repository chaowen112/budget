# Frontend build stage
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package.json web/package-lock.json ./
RUN npm ci

# Copy frontend source and build
COPY web/ ./
RUN npm run build

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

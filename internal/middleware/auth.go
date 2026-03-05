package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"

	"github.com/chaowen/budget/internal/pkg/jwt"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

// AuthInterceptor creates a gRPC interceptor for JWT authentication
func AuthInterceptor(jwtManager *jwt.Manager, publicMethods map[string]bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Skip auth for public methods
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Parse Bearer token
		token := strings.TrimPrefix(authHeader[0], "Bearer ")
		if token == authHeader[0] {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			if err == jwt.ErrExpiredToken {
				return nil, status.Error(codes.Unauthenticated, "token expired")
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add user info to context
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

		return handler(ctx, req)
	}
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	return userID, nil
}

// GetUserEmail extracts user email from context
func GetUserEmail(ctx context.Context) (string, error) {
	email, ok := ctx.Value(UserEmailKey).(string)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "user not authenticated")
	}
	return email, nil
}

// PublicMethods returns a map of methods that don't require authentication
func PublicMethods() map[string]bool {
	return map[string]bool{
		"/budget.v1.UserService/Register":              true,
		"/budget.v1.UserService/Login":                 true,
		"/budget.v1.UserService/RefreshToken":          true,
		"/budget.v1.CurrencyService/ListCurrencies":    true,
		"/budget.v1.CurrencyService/GetExchangeRate":   true,
		"/budget.v1.CurrencyService/ConvertCurrency":   true,
		"/budget.v1.CurrencyService/SyncExchangeRates": true,
	}
}

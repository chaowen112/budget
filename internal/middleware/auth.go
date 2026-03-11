package middleware

import (
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"

	"github.com/chaowen/budget/internal/metrics"
	"github.com/chaowen/budget/internal/pkg/jwt"
	"github.com/chaowen/budget/internal/repository"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

// AuthInterceptor creates a gRPC interceptor for JWT authentication
func AuthInterceptor(jwtManager *jwt.Manager, apiKeyRepo *repository.ApiKeyRepository, mc *metrics.Collector, publicMethods map[string]bool) grpc.UnaryServerInterceptor {
	authFail := func(method, reason string) {
		if mc != nil {
			mc.AuthFailuresTotal.WithLabelValues(reason, method).Inc()
		}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		// Skip auth for public methods
		if publicMethods[info.FullMethod] {
			resp, err := handler(ctx, req)
			logGRPCRequest(info.FullMethod, start, err, "public", uuid.Nil)
			return resp, err
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.WithField("method", info.FullMethod).Warn("Auth failed: missing metadata")
			authFail(info.FullMethod, "missing_metadata")
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			log.WithField("method", info.FullMethod).Warn("Auth failed: missing authorization header")
			authFail(info.FullMethod, "missing_header")
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Parse Bearer token
		token := strings.TrimPrefix(authHeader[0], "Bearer ")
		if token == authHeader[0] {
			log.WithField("method", info.FullMethod).Warn("Auth failed: invalid authorization format")
			authFail(info.FullMethod, "invalid_format")
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}

		// API Key Auth logic
		if strings.HasPrefix(token, "api_") {
			if apiKeyRepo == nil {
				log.WithField("method", info.FullMethod).Warn("Auth failed: api key verification disabled")
				authFail(info.FullMethod, "api_key_disabled")
				return nil, status.Error(codes.Unauthenticated, "api key verification disabled")
			}
			apiKey, err := apiKeyRepo.GetByKey(ctx, token)
			if err != nil {
				log.WithField("method", info.FullMethod).Warn("Auth failed: invalid api key")
				authFail(info.FullMethod, "invalid_api_key")
				return nil, status.Error(codes.Unauthenticated, "invalid api key")
			}
			// Update last used asynchronously
			go func() {
				_ = apiKeyRepo.UpdateLastUsed(context.Background(), apiKey.ID)
			}()

			ctx = context.WithValue(ctx, UserIDKey, apiKey.UserID)
			resp, handlerErr := handler(ctx, req)
			logGRPCRequest(info.FullMethod, start, handlerErr, "api_key", apiKey.UserID)
			return resp, handlerErr
		}

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			if err == jwt.ErrExpiredToken {
				log.WithField("method", info.FullMethod).Warn("Auth failed: token expired")
				authFail(info.FullMethod, "token_expired")
				return nil, status.Error(codes.Unauthenticated, "token expired")
			}
			log.WithField("method", info.FullMethod).Warn("Auth failed: invalid token")
			authFail(info.FullMethod, "invalid_token")
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add user info to context
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

		resp, handlerErr := handler(ctx, req)
		logGRPCRequest(info.FullMethod, start, handlerErr, "jwt", claims.UserID)
		return resp, handlerErr
	}
}

func logGRPCRequest(method string, start time.Time, err error, authType string, userID uuid.UUID) {
	duration := time.Since(start)
	fields := log.Fields{
		"method":    method,
		"duration":  duration.String(),
		"auth_type": authType,
	}
	if userID != uuid.Nil {
		fields["user_id"] = userID.String()
	}

	if err != nil {
		st, _ := status.FromError(err)
		fields["grpc_code"] = st.Code().String()
		if st.Code() == codes.Internal {
			log.WithFields(fields).WithError(err).Error("gRPC request failed")
		} else {
			log.WithFields(fields).WithError(err).Warn("gRPC request completed with error")
		}
	} else {
		fields["grpc_code"] = codes.OK.String()
		log.WithFields(fields).Info("gRPC request completed")
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

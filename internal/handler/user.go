package handler

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/middleware"
	"github.com/chaowen/budget/internal/model"
	"github.com/chaowen/budget/internal/repository"
	"github.com/chaowen/budget/internal/service"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if len(req.Password) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	result, err := h.userService.Register(ctx, service.RegisterInput{
		Email:        req.Email,
		Password:     req.Password,
		Name:         req.Name,
		BaseCurrency: req.BaseCurrency,
	})

	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &pb.RegisterResponse{
		User:         userToProto(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *UserHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	result, err := h.userService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Error(codes.Internal, "failed to login")
	}

	return &pb.LoginResponse{
		User:         userToProto(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *UserHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	accessToken, refreshToken, err := h.userService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *UserHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.userService.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get profile")
	}

	return &pb.GetProfileResponse{
		User: userToProto(user),
	}, nil
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.userService.UpdateProfile(ctx, userID, req.Name, req.BaseCurrency)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	return &pb.UpdateProfileResponse{
		User: userToProto(user),
	}, nil
}

func (h *UserHandler) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.CurrentPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "current password is required")
	}
	if req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "new password is required")
	}
	if len(req.NewPassword) < 8 {
		return nil, status.Error(codes.InvalidArgument, "new password must be at least 8 characters")
	}

	err = h.userService.ChangePassword(ctx, userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPassword) {
			return nil, status.Error(codes.InvalidArgument, "current password is incorrect")
		}
		return nil, status.Error(codes.Internal, "failed to change password")
	}

	return &pb.ChangePasswordResponse{}, nil
}

func (h *UserHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	_ = h.userService.Logout(ctx, req.RefreshToken)

	return &pb.LogoutResponse{}, nil
}

func (h *UserHandler) CreateApiKey(ctx context.Context, req *pb.CreateApiKeyRequest) (*pb.CreateApiKeyResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	apiKey, err := h.userService.CreateApiKey(ctx, userID, req.Name)
	if err != nil {
		if err.Error() == "maximum limit of 3 API keys reached" {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to create api key")
	}

	return &pb.CreateApiKeyResponse{
		ApiKey: &pb.ApiKey{
			Id:        apiKey.ID.String(),
			Name:      apiKey.Name,
			KeyValue:  apiKey.KeyValue, // Only populated on creation
			CreatedAt: timestamppb.New(apiKey.CreatedAt),
		},
	}, nil
}

func (h *UserHandler) ListApiKeys(ctx context.Context, req *pb.GetProfileRequest) (*pb.ListApiKeysResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	keys, err := h.userService.ListApiKeys(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list api keys")
	}

	var pbKeys []*pb.ApiKey
	for _, key := range keys {
		pbKey := &pb.ApiKey{
			Id:        key.ID.String(),
			Name:      key.Name,
			CreatedAt: timestamppb.New(key.CreatedAt),
		}
		if key.LastUsedAt != nil {
			pbKey.LastUsedAt = timestamppb.New(*key.LastUsedAt)
		}
		pbKeys = append(pbKeys, pbKey)
	}

	return &pb.ListApiKeysResponse{
		ApiKeys: pbKeys,
	}, nil
}

func (h *UserHandler) DeleteApiKey(ctx context.Context, req *pb.DeleteApiKeyRequest) (*pb.DeleteApiKeyResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	keyID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid api key id")
	}

	if err := h.userService.DeleteApiKey(ctx, userID, keyID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete api key")
	}

	return &pb.DeleteApiKeyResponse{}, nil
}

func userToProto(user *model.User) *pb.User {
	return &pb.User{
		Id:           user.ID.String(),
		Email:        user.Email,
		Name:         user.Name,
		BaseCurrency: user.BaseCurrency,
		CreatedAt:    timestamppb.New(user.CreatedAt),
	}
}

package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/chaowen/budget/internal/model"
	"github.com/chaowen/budget/internal/pkg/jwt"
	"github.com/chaowen/budget/internal/pkg/password"
	"github.com/chaowen/budget/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidPassword    = errors.New("invalid current password")
)

type UserService struct {
	userRepo   *repository.UserRepository
	jwtManager *jwt.Manager
}

func NewUserService(userRepo *repository.UserRepository, jwtManager *jwt.Manager) *UserService {
	return &UserService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// RegisterInput represents registration data
type RegisterInput struct {
	Email        string
	Password     string
	Name         string
	BaseCurrency string
}

// AuthResult represents authentication result
type AuthResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
}

// Register creates a new user account
func (s *UserService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	// Hash password
	hash, err := password.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	// Set default currency if not provided
	if input.BaseCurrency == "" {
		input.BaseCurrency = "SGD"
	}

	user := &model.User{
		Email:        input.Email,
		PasswordHash: hash,
		Name:         input.Name,
		BaseCurrency: input.BaseCurrency,
	}

	// Create user
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, tokenHash, expiresAt, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Save refresh token
	if err := s.userRepo.SaveRefreshToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Login authenticates a user
func (s *UserService) Login(ctx context.Context, email, pwd string) (*AuthResult, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify password
	if !password.Verify(pwd, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, tokenHash, expiresAt, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Save refresh token
	if err := s.userRepo.SaveRefreshToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshToken refreshes the access token
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// Validate refresh token format
	if err := s.jwtManager.ValidateRefreshToken(refreshToken); err != nil {
		return "", "", err
	}

	// Check if token exists in database
	tokenHash := jwt.HashToken(refreshToken)
	storedToken, err := s.userRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return "", "", err
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return "", "", err
	}

	// Delete old refresh token
	if err := s.userRepo.DeleteRefreshToken(ctx, tokenHash); err != nil {
		return "", "", err
	}

	// Generate new tokens
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, newTokenHash, expiresAt, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	// Save new refresh token
	if err := s.userRepo.SaveRefreshToken(ctx, user.ID, newTokenHash, expiresAt); err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

// Logout invalidates the refresh token
func (s *UserService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := jwt.HashToken(refreshToken)
	return s.userRepo.DeleteRefreshToken(ctx, tokenHash)
}

// GetProfile retrieves user profile
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// UpdateProfile updates user profile
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, name, baseCurrency string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		user.Name = name
	}
	if baseCurrency != "" {
		user.BaseCurrency = baseCurrency
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword changes user password
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if !password.Verify(currentPassword, user.PasswordHash) {
		return ErrInvalidPassword
	}

	// Hash new password
	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}

	// Invalidate all refresh tokens for security
	return s.userRepo.DeleteUserRefreshTokens(ctx, userID)
}

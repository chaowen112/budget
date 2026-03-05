package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/middleware"
	"github.com/chaowen/budget/internal/model"
	"github.com/chaowen/budget/internal/repository"
)

type CategoryHandler struct {
	pb.UnimplementedCategoryServiceServer
	categoryRepo *repository.CategoryRepository
}

func NewCategoryHandler(categoryRepo *repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{
		categoryRepo: categoryRepo,
	}
}

func (h *CategoryHandler) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	var categoryType *model.CategoryType
	if req.Type != pb.TransactionType_TRANSACTION_TYPE_UNSPECIFIED {
		ct := protoToTransactionType(req.Type)
		categoryType = &ct
	}

	categories, err := h.categoryRepo.ListAll(ctx, userID, categoryType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list categories")
	}

	pbCategories := make([]*pb.Category, len(categories))
	for i, c := range categories {
		pbCategories[i] = categoryToProto(&c)
	}

	return &pb.ListCategoriesResponse{
		Categories: pbCategories,
	}, nil
}

func (h *CategoryHandler) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Type == pb.TransactionType_TRANSACTION_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}

	category := &model.Category{
		UserID: &userID,
		Name:   req.Name,
		Type:   protoToTransactionType(req.Type),
		Icon:   req.Icon,
	}

	if err := h.categoryRepo.Create(ctx, category); err != nil {
		return nil, status.Error(codes.Internal, "failed to create category")
	}

	return &pb.CreateCategoryResponse{
		Category: categoryToProto(category),
	}, nil
}

func (h *CategoryHandler) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.GetCategoryResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category ID")
	}

	category, err := h.categoryRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, "category not found")
		}
		return nil, status.Error(codes.Internal, "failed to get category")
	}

	return &pb.GetCategoryResponse{
		Category: categoryToProto(category),
	}, nil
}

func (h *CategoryHandler) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.UpdateCategoryResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category ID")
	}

	category := &model.Category{
		ID:     id,
		UserID: &userID,
		Name:   req.Name,
		Icon:   req.Icon,
	}

	if err := h.categoryRepo.Update(ctx, category); err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, "category not found or cannot be updated")
		}
		return nil, status.Error(codes.Internal, "failed to update category")
	}

	// Fetch updated category
	updatedCategory, err := h.categoryRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get updated category")
	}

	return &pb.UpdateCategoryResponse{
		Category: categoryToProto(updatedCategory),
	}, nil
}

func (h *CategoryHandler) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.DeleteCategoryResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category ID")
	}

	if err := h.categoryRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, "category not found or cannot be deleted")
		}
		return nil, status.Error(codes.Internal, "failed to delete category")
	}

	return &pb.DeleteCategoryResponse{}, nil
}

func categoryToProto(c *model.Category) *pb.Category {
	return &pb.Category{
		Id:        c.ID.String(),
		Name:      c.Name,
		Type:      transactionTypeToProto(c.Type),
		Icon:      c.Icon,
		IsSystem:  c.IsSystem,
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
}

func transactionTypeToProto(t model.CategoryType) pb.TransactionType {
	switch t {
	case model.CategoryTypeExpense:
		return pb.TransactionType_TRANSACTION_TYPE_EXPENSE
	case model.CategoryTypeIncome:
		return pb.TransactionType_TRANSACTION_TYPE_INCOME
	default:
		return pb.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func protoToTransactionType(t pb.TransactionType) model.CategoryType {
	switch t {
	case pb.TransactionType_TRANSACTION_TYPE_EXPENSE:
		return model.CategoryTypeExpense
	case pb.TransactionType_TRANSACTION_TYPE_INCOME:
		return model.CategoryTypeIncome
	default:
		return model.CategoryTypeExpense
	}
}

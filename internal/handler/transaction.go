package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/middleware"
	"github.com/chaowen/budget/internal/model"
	"github.com/chaowen/budget/internal/repository"
)

type TransactionHandler struct {
	pb.UnimplementedTransactionServiceServer
	transactionRepo *repository.TransactionRepository
	categoryRepo    *repository.CategoryRepository
}

func NewTransactionHandler(transactionRepo *repository.TransactionRepository, categoryRepo *repository.CategoryRepository) *TransactionHandler {
	return &TransactionHandler{
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
	}
}

func (h *TransactionHandler) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.CategoryId == "" {
		return nil, status.Error(codes.InvalidArgument, "category_id is required")
	}
	if req.Amount == nil {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	categoryID, err := uuid.Parse(req.CategoryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category_id")
	}

	// Get category to infer transaction type if not provided
	category, err := h.categoryRepo.GetByID(ctx, categoryID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "category not found")
	}

	// Use provided type or infer from category
	var transactionType model.CategoryType
	if req.Type != pb.TransactionType_TRANSACTION_TYPE_UNSPECIFIED {
		transactionType = protoToTransactionType(req.Type)
	} else {
		transactionType = category.Type
	}

	amount, err := decimal.NewFromString(req.Amount.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount")
	}

	currency := req.Amount.Currency
	if currency == "" {
		currency = "SGD"
	}

	transactionDate := req.TransactionDate.AsTime()
	if transactionDate.IsZero() {
		return nil, status.Error(codes.InvalidArgument, "transaction_date is required")
	}

	transaction := &model.Transaction{
		UserID:          userID,
		CategoryID:      categoryID,
		Amount:          amount,
		Currency:        currency,
		Type:            transactionType,
		TransactionDate: transactionDate,
		Description:     req.Description,
		Tags:            req.Tags,
	}

	if err := h.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, status.Error(codes.Internal, "failed to create transaction")
	}

	// Set category name from the category we already fetched
	transaction.CategoryName = category.Name

	return &pb.CreateTransactionResponse{
		Transaction: transactionToProto(transaction),
	}, nil
}

func (h *TransactionHandler) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid transaction ID")
	}

	transaction, err := h.transactionRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, status.Error(codes.NotFound, "transaction not found")
		}
		return nil, status.Error(codes.Internal, "failed to get transaction")
	}

	return &pb.GetTransactionResponse{
		Transaction: transactionToProto(transaction),
	}, nil
}

func (h *TransactionHandler) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	filter := repository.TransactionFilter{
		UserID:   userID,
		Currency: req.Currency,
		Page:     1,
		PageSize: 20,
	}

	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			filter.Page = int(req.Pagination.Page)
		}
		if req.Pagination.PageSize > 0 {
			filter.PageSize = int(req.Pagination.PageSize)
		}
	}

	if req.CategoryId != "" {
		categoryID, err := uuid.Parse(req.CategoryId)
		if err == nil {
			filter.CategoryID = &categoryID
		}
	}

	if req.Type != pb.TransactionType_TRANSACTION_TYPE_UNSPECIFIED {
		t := protoToTransactionType(req.Type)
		filter.Type = &t
	}

	if req.DateRange != nil {
		if req.DateRange.StartDate != nil {
			startDate := req.DateRange.StartDate.AsTime()
			filter.StartDate = &startDate
		}
		if req.DateRange.EndDate != nil {
			endDate := req.DateRange.EndDate.AsTime()
			filter.EndDate = &endDate
		}
	}

	result, err := h.transactionRepo.List(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list transactions")
	}

	pbTransactions := make([]*pb.Transaction, len(result.Transactions))
	for i, t := range result.Transactions {
		pbTransactions[i] = transactionToProto(&t)
	}

	totalPages := (result.TotalCount + filter.PageSize - 1) / filter.PageSize

	return &pb.ListTransactionsResponse{
		Transactions: pbTransactions,
		Pagination: &pb.PaginationResponse{
			Page:       int32(filter.Page),
			PageSize:   int32(filter.PageSize),
			TotalCount: int32(result.TotalCount),
			TotalPages: int32(totalPages),
		},
	}, nil
}

func (h *TransactionHandler) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.UpdateTransactionResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid transaction ID")
	}

	// Get existing transaction
	transaction, err := h.transactionRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, status.Error(codes.NotFound, "transaction not found")
		}
		return nil, status.Error(codes.Internal, "failed to get transaction")
	}

	// Update fields if provided
	if req.CategoryId != "" {
		categoryID, err := uuid.Parse(req.CategoryId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid category_id")
		}
		transaction.CategoryID = categoryID
	}

	if req.Amount != nil {
		amount, err := decimal.NewFromString(req.Amount.Amount)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid amount")
		}
		transaction.Amount = amount
	}

	if req.TransactionDate != nil {
		transaction.TransactionDate = req.TransactionDate.AsTime()
	}

	if req.Description != "" {
		transaction.Description = req.Description
	}

	if len(req.Tags) > 0 {
		transaction.Tags = req.Tags
	}

	if err := h.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, status.Error(codes.Internal, "failed to update transaction")
	}

	// Refetch to get category name
	transaction, _ = h.transactionRepo.GetByID(ctx, id, userID)

	return &pb.UpdateTransactionResponse{
		Transaction: transactionToProto(transaction),
	}, nil
}

func (h *TransactionHandler) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid transaction ID")
	}

	if err := h.transactionRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, status.Error(codes.NotFound, "transaction not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete transaction")
	}

	return &pb.DeleteTransactionResponse{}, nil
}

func transactionToProto(t *model.Transaction) *pb.Transaction {
	return &pb.Transaction{
		Id:           t.ID.String(),
		CategoryId:   t.CategoryID.String(),
		CategoryName: t.CategoryName,
		Amount: &pb.Money{
			Amount:   t.Amount.String(),
			Currency: t.Currency,
		},
		Type:            transactionTypeToProto(t.Type),
		TransactionDate: timestamppb.New(t.TransactionDate),
		Description:     t.Description,
		Tags:            t.Tags,
		CreatedAt:       timestamppb.New(t.CreatedAt),
	}
}

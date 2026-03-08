package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	assetRepo       *repository.AssetRepository
	accountingRepo  *repository.AccountingRepository
}

func NewTransactionHandler(
	transactionRepo *repository.TransactionRepository,
	categoryRepo *repository.CategoryRepository,
	assetRepo *repository.AssetRepository,
	accountingRepo *repository.AccountingRepository,
) *TransactionHandler {
	return &TransactionHandler{
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
		assetRepo:       assetRepo,
		accountingRepo:  accountingRepo,
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

	sourceAssetID, err := sourceAssetIDFromMetadata(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _, err := h.assetRepo.GetByID(ctx, sourceAssetID, userID); err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			return nil, status.Error(codes.InvalidArgument, "source asset not found")
		}
		return nil, status.Error(codes.Internal, "failed to validate source asset")
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
	if err := h.transactionRepo.SetSourceAssetLink(ctx, transaction.ID, sourceAssetID); err != nil {
		_ = h.transactionRepo.Delete(ctx, transaction.ID, userID)
		if isUndefinedTableError(err) {
			return nil, status.Error(codes.FailedPrecondition, "transaction asset link table missing: run latest database migrations")
		}
		return nil, status.Error(codes.Internal, "failed to link transaction to source asset")
	}

	asset, err := h.assetRepo.GetByID(ctx, sourceAssetID, userID)
	if err != nil {
		_ = h.transactionRepo.Delete(ctx, transaction.ID, userID)
		return nil, status.Error(codes.Internal, "failed to load source asset")
	}
	assetAccountID, err := h.accountingRepo.EnsureAssetAccount(ctx, asset)
	if err != nil {
		_ = h.transactionRepo.Delete(ctx, transaction.ID, userID)
		return nil, status.Error(codes.Internal, "failed to ensure asset account")
	}
	categoryAccountID, err := h.accountingRepo.EnsureCategoryAccount(ctx, userID, category.ID, currency, category.Type, category.Name)
	if err != nil {
		_ = h.transactionRepo.Delete(ctx, transaction.ID, userID)
		return nil, status.Error(codes.Internal, "failed to ensure category account")
	}
	if err := h.accountingRepo.UpsertTransactionEntry(ctx, userID, transaction, assetAccountID, categoryAccountID); err != nil {
		_ = h.transactionRepo.Delete(ctx, transaction.ID, userID)
		return nil, status.Error(codes.Internal, "failed to post transaction journal")
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
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid category_id")
		}
		filter.CategoryID = &categoryID
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

	if len(req.Tags) > 0 {
		filter.Tags = req.Tags
	}

	searchKeyword := searchKeywordFromMetadata(ctx)
	if searchKeyword != "" {
		filter.Search = searchKeyword
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

	oldAssetID, err := h.transactionRepo.GetSourceAssetLink(ctx, transaction.ID)
	if err != nil {
		if isUndefinedTableError(err) {
			return nil, status.Error(codes.FailedPrecondition, "transaction asset link table missing: run latest database migrations")
		}
		if errors.Is(err, repository.ErrTransactionAssetLinkNotFound) {
			return nil, status.Error(codes.FailedPrecondition, "transaction source asset link missing")
		}
		return nil, status.Error(codes.Internal, "failed to fetch transaction source asset")
	}

	newAssetID := oldAssetID
	sourceAssetID, mdErr := sourceAssetIDFromMetadataOptional(ctx)
	if mdErr != nil {
		return nil, status.Error(codes.InvalidArgument, mdErr.Error())
	}
	if sourceAssetID != nil {
		if _, err := h.assetRepo.GetByID(ctx, *sourceAssetID, userID); err != nil {
			if errors.Is(err, repository.ErrAssetNotFound) {
				return nil, status.Error(codes.InvalidArgument, "source asset not found")
			}
			return nil, status.Error(codes.Internal, "failed to validate source asset")
		}
		newAssetID = *sourceAssetID
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

	if newAssetID != oldAssetID {
		if err := h.transactionRepo.SetSourceAssetLink(ctx, transaction.ID, newAssetID); err != nil {
			if isUndefinedTableError(err) {
				return nil, status.Error(codes.FailedPrecondition, "transaction asset link table missing: run latest database migrations")
			}
			return nil, status.Error(codes.Internal, "failed to update transaction source asset")
		}
	}

	updatedCategory, err := h.categoryRepo.GetByID(ctx, transaction.CategoryID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch transaction category")
	}
	updatedAsset, err := h.assetRepo.GetByID(ctx, newAssetID, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch source asset")
	}
	assetAccountID, err := h.accountingRepo.EnsureAssetAccount(ctx, updatedAsset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to ensure asset account")
	}
	categoryAccountID, err := h.accountingRepo.EnsureCategoryAccount(ctx, userID, updatedCategory.ID, transaction.Currency, updatedCategory.Type, updatedCategory.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to ensure category account")
	}
	if err := h.accountingRepo.UpsertTransactionEntry(ctx, userID, transaction, assetAccountID, categoryAccountID); err != nil {
		return nil, status.Error(codes.Internal, "failed to post updated transaction journal")
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

	transaction, err := h.transactionRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, status.Error(codes.NotFound, "transaction not found")
		}
		return nil, status.Error(codes.Internal, "failed to get transaction")
	}

	if err := h.transactionRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, status.Error(codes.NotFound, "transaction not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete transaction")
	}

	if err := h.accountingRepo.DeleteTransactionEntry(ctx, userID, transaction.ID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete transaction journal")
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

func sourceAssetIDFromMetadata(ctx context.Context) (uuid.UUID, error) {
	sourceAssetID, err := sourceAssetIDFromMetadataOptional(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if sourceAssetID == nil {
		return uuid.Nil, errors.New("source_asset_id is required")
	}
	return *sourceAssetID, nil
}

func sourceAssetIDFromMetadataOptional(ctx context.Context) (*uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil
	}

	keys := []string{"source-asset-id", "x-source-asset-id", "grpcgateway-source-asset-id"}
	for _, key := range keys {
		values := md.Get(key)
		if len(values) == 0 || values[0] == "" {
			continue
		}
		id, err := uuid.Parse(values[0])
		if err != nil {
			return nil, errors.New("invalid source_asset_id")
		}
		return &id, nil
	}

	return nil, nil
}

func searchKeywordFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	keys := []string{"search-keyword", "x-search-keyword", "grpcgateway-search-keyword"}
	for _, key := range keys {
		values := md.Get(key)
		if len(values) == 0 {
			continue
		}
		keyword := strings.TrimSpace(values[0])
		if keyword == "" {
			continue
		}
		return keyword
	}

	return ""
}

func isUndefinedTableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01"
	}
	return false
}

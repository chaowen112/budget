package handler

import (
	"context"
	"errors"
	"time"

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

type BudgetHandler struct {
	pb.UnimplementedBudgetServiceServer
	budgetRepo   *repository.BudgetRepository
	currencyRepo *repository.CurrencyRepository
	userRepo     *repository.UserRepository
}

func NewBudgetHandler(budgetRepo *repository.BudgetRepository, currencyRepo *repository.CurrencyRepository, userRepo *repository.UserRepository) *BudgetHandler {
	return &BudgetHandler{
		budgetRepo:   budgetRepo,
		currencyRepo: currencyRepo,
		userRepo:     userRepo,
	}
}

func (h *BudgetHandler) computeBudgetStatus(ctx context.Context, budget *model.Budget) (*model.BudgetStatus, error) {
	spentByCurrency, err := h.budgetRepo.GetSpentAmountByCurrency(ctx, budget.UserID, budget.CategoryID, budget.PeriodType, budget.StartDate)
	if err != nil {
		return nil, err
	}

	spent := decimal.Zero
	for _, ca := range spentByCurrency {
		if ca.Currency == budget.Currency {
			spent = spent.Add(ca.Amount)
		} else {
			converted, err := h.currencyRepo.ConvertAmount(ctx, ca.Amount, ca.Currency, budget.Currency)
			if err != nil {
				return nil, err
			}
			spent = spent.Add(converted)
		}
	}

	remaining := budget.Amount.Sub(spent)
	percentUsed := float64(0)
	if !budget.Amount.IsZero() {
		percentUsed = spent.Div(budget.Amount).InexactFloat64() * 100
	}

	return &model.BudgetStatus{
		Budget:       *budget,
		Spent:        spent,
		Remaining:    remaining,
		PercentUsed:  percentUsed,
		IsOverBudget: remaining.IsNegative(),
	}, nil
}

func (h *BudgetHandler) CreateBudget(ctx context.Context, req *pb.CreateBudgetRequest) (*pb.CreateBudgetResponse, error) {
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
	if req.PeriodType == pb.PeriodType_PERIOD_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "period_type is required")
	}

	categoryID, err := uuid.Parse(req.CategoryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category_id")
	}

	amount, err := decimal.NewFromString(req.Amount.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount")
	}

	currency := req.Amount.Currency
	if currency == "" {
		currency = "SGD"
	}

	startDate := normalizeDateOnly(req.StartDate.AsTime())

	budget := &model.Budget{
		UserID:     userID,
		CategoryID: categoryID,
		Amount:     amount,
		Currency:   currency,
		PeriodType: protoToPeriodType(req.PeriodType),
		StartDate:  startDate,
	}

	if err := h.budgetRepo.Create(ctx, budget); err != nil {
		return nil, status.Error(codes.Internal, "failed to create budget")
	}

	// Refetch to get category name
	budget, _ = h.budgetRepo.GetByID(ctx, budget.ID, userID)

	return &pb.CreateBudgetResponse{
		Budget: budgetToProto(budget),
	}, nil
}

func (h *BudgetHandler) GetBudget(ctx context.Context, req *pb.GetBudgetRequest) (*pb.GetBudgetResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid budget ID")
	}

	budget, err := h.budgetRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrBudgetNotFound) {
			return nil, status.Error(codes.NotFound, "budget not found")
		}
		return nil, status.Error(codes.Internal, "failed to get budget")
	}

	return &pb.GetBudgetResponse{
		Budget: budgetToProto(budget),
	}, nil
}

func (h *BudgetHandler) ListBudgets(ctx context.Context, req *pb.ListBudgetsRequest) (*pb.ListBudgetsResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	var periodType *model.PeriodType
	if req.PeriodType != pb.PeriodType_PERIOD_TYPE_UNSPECIFIED {
		pt := protoToPeriodType(req.PeriodType)
		periodType = &pt
	}

	budgets, err := h.budgetRepo.List(ctx, userID, periodType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list budgets")
	}

	pbBudgets := make([]*pb.Budget, len(budgets))
	for i, b := range budgets {
		pbBudgets[i] = budgetToProto(&b)
	}

	return &pb.ListBudgetsResponse{
		Budgets: pbBudgets,
	}, nil
}

func (h *BudgetHandler) UpdateBudget(ctx context.Context, req *pb.UpdateBudgetRequest) (*pb.UpdateBudgetResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid budget ID")
	}

	budget, err := h.budgetRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrBudgetNotFound) {
			return nil, status.Error(codes.NotFound, "budget not found")
		}
		return nil, status.Error(codes.Internal, "failed to get budget")
	}

	if req.Amount != nil {
		amount, err := decimal.NewFromString(req.Amount.Amount)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid amount")
		}
		budget.Amount = amount
	}

	if req.PeriodType != pb.PeriodType_PERIOD_TYPE_UNSPECIFIED {
		budget.PeriodType = protoToPeriodType(req.PeriodType)
	}

	if req.StartDate != nil {
		budget.StartDate = normalizeDateOnly(req.StartDate.AsTime())
	}

	if err := h.budgetRepo.Update(ctx, budget); err != nil {
		return nil, status.Error(codes.Internal, "failed to update budget")
	}

	return &pb.UpdateBudgetResponse{
		Budget: budgetToProto(budget),
	}, nil
}

func (h *BudgetHandler) DeleteBudget(ctx context.Context, req *pb.DeleteBudgetRequest) (*pb.DeleteBudgetResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid budget ID")
	}

	if err := h.budgetRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrBudgetNotFound) {
			return nil, status.Error(codes.NotFound, "budget not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete budget")
	}

	return &pb.DeleteBudgetResponse{}, nil
}

func (h *BudgetHandler) GetBudgetStatus(ctx context.Context, req *pb.GetBudgetStatusRequest) (*pb.GetBudgetStatusResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.BudgetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid budget ID")
	}

	budget, err := h.budgetRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrBudgetNotFound) {
			return nil, status.Error(codes.NotFound, "budget not found")
		}
		return nil, status.Error(codes.Internal, "failed to get budget")
	}

	budgetStatus, err := h.computeBudgetStatus(ctx, budget)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get budget status")
	}

	return &pb.GetBudgetStatusResponse{
		Status: budgetStatusToProto(budgetStatus),
	}, nil
}

func (h *BudgetHandler) GetAllBudgetStatuses(ctx context.Context, req *pb.GetAllBudgetStatusesRequest) (*pb.GetAllBudgetStatusesResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}
	baseCurrency := user.BaseCurrency

	var periodType *model.PeriodType
	if req.PeriodType != pb.PeriodType_PERIOD_TYPE_UNSPECIFIED {
		pt := protoToPeriodType(req.PeriodType)
		periodType = &pt
	}

	budgets, err := h.budgetRepo.List(ctx, userID, periodType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list budgets")
	}

	totalBudgeted := decimal.Zero
	totalSpent := decimal.Zero
	statuses := make([]*pb.BudgetStatus, 0, len(budgets))

	for _, b := range budgets {
		budgetStatus, err := h.computeBudgetStatus(ctx, &b)
		if err != nil {
			continue
		}
		statuses = append(statuses, budgetStatusToProto(budgetStatus))

		budgetedInBase, err := h.currencyRepo.ConvertAmount(ctx, b.Amount, b.Currency, baseCurrency)
		if err != nil {
			budgetedInBase = b.Amount
		}
		spentInBase, err := h.currencyRepo.ConvertAmount(ctx, budgetStatus.Spent, b.Currency, baseCurrency)
		if err != nil {
			spentInBase = budgetStatus.Spent
		}
		totalBudgeted = totalBudgeted.Add(budgetedInBase)
		totalSpent = totalSpent.Add(spentInBase)
	}

	return &pb.GetAllBudgetStatusesResponse{
		Statuses: statuses,
		TotalBudgeted: &pb.Money{
			Amount:   totalBudgeted.String(),
			Currency: baseCurrency,
		},
		TotalSpent: &pb.Money{
			Amount:   totalSpent.String(),
			Currency: baseCurrency,
		},
	}, nil
}

func budgetToProto(b *model.Budget) *pb.Budget {
	return &pb.Budget{
		Id:           b.ID.String(),
		CategoryId:   b.CategoryID.String(),
		CategoryName: b.CategoryName,
		Amount: &pb.Money{
			Amount:   b.Amount.String(),
			Currency: b.Currency,
		},
		PeriodType: periodTypeToProto(b.PeriodType),
		StartDate:  timestamppb.New(b.StartDate),
		CreatedAt:  timestamppb.New(b.CreatedAt),
	}
}

func budgetStatusToProto(s *model.BudgetStatus) *pb.BudgetStatus {
	return &pb.BudgetStatus{
		Budget: budgetToProto(&s.Budget),
		Spent: &pb.Money{
			Amount:   s.Spent.String(),
			Currency: s.Budget.Currency,
		},
		Remaining: &pb.Money{
			Amount:   s.Remaining.String(),
			Currency: s.Budget.Currency,
		},
		PercentageUsed: s.PercentUsed,
		IsOverBudget:   s.IsOverBudget,
	}
}

func periodTypeToProto(pt model.PeriodType) pb.PeriodType {
	switch pt {
	case model.PeriodTypeDaily:
		return pb.PeriodType_PERIOD_TYPE_DAILY
	case model.PeriodTypeWeekly:
		return pb.PeriodType_PERIOD_TYPE_WEEKLY
	case model.PeriodTypeMonthly:
		return pb.PeriodType_PERIOD_TYPE_MONTHLY
	case model.PeriodTypeYearly:
		return pb.PeriodType_PERIOD_TYPE_YEARLY
	default:
		return pb.PeriodType_PERIOD_TYPE_UNSPECIFIED
	}
}

func protoToPeriodType(pt pb.PeriodType) model.PeriodType {
	switch pt {
	case pb.PeriodType_PERIOD_TYPE_DAILY:
		return model.PeriodTypeDaily
	case pb.PeriodType_PERIOD_TYPE_WEEKLY:
		return model.PeriodTypeWeekly
	case pb.PeriodType_PERIOD_TYPE_MONTHLY:
		return model.PeriodTypeMonthly
	case pb.PeriodType_PERIOD_TYPE_YEARLY:
		return model.PeriodTypeYearly
	default:
		return model.PeriodTypeMonthly
	}
}

func normalizeDateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

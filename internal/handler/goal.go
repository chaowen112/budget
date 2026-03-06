package handler

import (
	"context"
	"errors"
	"math"
	"time"

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

type GoalHandler struct {
	pb.UnimplementedSavingGoalServiceServer
	goalRepo        *repository.GoalRepository
	assetRepo       *repository.AssetRepository
	transactionRepo *repository.TransactionRepository
}

func NewGoalHandler(goalRepo *repository.GoalRepository, assetRepo *repository.AssetRepository, transactionRepo *repository.TransactionRepository) *GoalHandler {
	return &GoalHandler{
		goalRepo:        goalRepo,
		assetRepo:       assetRepo,
		transactionRepo: transactionRepo,
	}
}

func (h *GoalHandler) CreateSavingGoal(ctx context.Context, req *pb.CreateSavingGoalRequest) (*pb.CreateSavingGoalResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.TargetAmount == nil {
		return nil, status.Error(codes.InvalidArgument, "target_amount is required")
	}

	targetAmount, err := decimal.NewFromString(req.TargetAmount.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_amount")
	}

	currency := req.TargetAmount.Currency
	if currency == "" {
		currency = "SGD"
	}

	var deadline *time.Time
	if req.Deadline != nil {
		t := req.Deadline.AsTime()
		deadline = &t
	}

	linkedAssetIDs := make([]uuid.UUID, 0, len(req.LinkedAssetIds))
	for _, idStr := range req.LinkedAssetIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid linked_asset_id: "+idStr)
		}
		linkedAssetIDs = append(linkedAssetIDs, id)
	}

	goal := &model.SavingGoal{
		UserID:         userID,
		Name:           req.Name,
		TargetAmount:   targetAmount,
		CurrentAmount:  decimal.Zero,
		Currency:       currency,
		Deadline:       deadline,
		LinkedAssetIDs: linkedAssetIDs,
		Notes:          req.Notes,
	}

	if err := h.goalRepo.Create(ctx, goal); err != nil {
		return nil, status.Error(codes.Internal, "failed to create saving goal")
	}

	return &pb.CreateSavingGoalResponse{
		Goal: goalToProto(goal),
	}, nil
}

func (h *GoalHandler) GetSavingGoal(ctx context.Context, req *pb.GetSavingGoalRequest) (*pb.GetSavingGoalResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid goal ID")
	}

	goal, err := h.goalRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			return nil, status.Error(codes.NotFound, "saving goal not found")
		}
		return nil, status.Error(codes.Internal, "failed to get saving goal")
	}

	return &pb.GetSavingGoalResponse{
		Goal: goalToProto(goal),
	}, nil
}

func (h *GoalHandler) ListSavingGoals(ctx context.Context, req *pb.ListSavingGoalsRequest) (*pb.ListSavingGoalsResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	goals, err := h.goalRepo.List(ctx, userID, req.IncludeCompleted)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list saving goals")
	}

	pbGoals := make([]*pb.SavingGoal, len(goals))
	for i, g := range goals {
		pbGoals[i] = goalToProto(&g)
	}

	return &pb.ListSavingGoalsResponse{
		Goals: pbGoals,
	}, nil
}

func (h *GoalHandler) UpdateSavingGoal(ctx context.Context, req *pb.UpdateSavingGoalRequest) (*pb.UpdateSavingGoalResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid goal ID")
	}

	goal, err := h.goalRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			return nil, status.Error(codes.NotFound, "saving goal not found")
		}
		return nil, status.Error(codes.Internal, "failed to get saving goal")
	}

	if req.Name != "" {
		goal.Name = req.Name
	}

	if req.TargetAmount != nil {
		targetAmount, err := decimal.NewFromString(req.TargetAmount.Amount)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid target_amount")
		}
		goal.TargetAmount = targetAmount
	}

	if req.Deadline != nil {
		t := req.Deadline.AsTime()
		goal.Deadline = &t
	}

	if len(req.LinkedAssetIds) > 0 {
		linkedAssetIDs := make([]uuid.UUID, 0, len(req.LinkedAssetIds))
		for _, idStr := range req.LinkedAssetIds {
			assetID, err := uuid.Parse(idStr)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid linked_asset_id: "+idStr)
			}
			linkedAssetIDs = append(linkedAssetIDs, assetID)
		}
		goal.LinkedAssetIDs = linkedAssetIDs
	}

	if req.Notes != "" {
		goal.Notes = req.Notes
	}

	if err := h.goalRepo.Update(ctx, goal); err != nil {
		return nil, status.Error(codes.Internal, "failed to update saving goal")
	}

	return &pb.UpdateSavingGoalResponse{
		Goal: goalToProto(goal),
	}, nil
}

func (h *GoalHandler) UpdateGoalProgress(ctx context.Context, req *pb.UpdateGoalProgressRequest) (*pb.UpdateGoalProgressResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid goal ID")
	}

	if req.CurrentAmount == nil {
		return nil, status.Error(codes.InvalidArgument, "current_amount is required")
	}

	currentAmount, err := decimal.NewFromString(req.CurrentAmount.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid current_amount")
	}

	goal, err := h.goalRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			return nil, status.Error(codes.NotFound, "saving goal not found")
		}
		return nil, status.Error(codes.Internal, "failed to get saving goal")
	}

	previousAmount := goal.CurrentAmount
	goal.CurrentAmount = currentAmount

	if err := h.goalRepo.UpdateProgress(ctx, goal); err != nil {
		return nil, status.Error(codes.Internal, "failed to update goal progress")
	}

	changeSource := "manual"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if sources := md.Get("x-goal-change-source"); len(sources) > 0 && sources[0] != "" {
			changeSource = sources[0]
		}
	}
	delta := currentAmount.Sub(previousAmount)
	if !delta.IsZero() {
		_ = h.goalRepo.RecordContribution(ctx, goal.ID, delta, goal.CurrentAmount, changeSource, time.Now())
	}

	_ = h.goalRepo.RecordProgressSnapshot(ctx, goal.ID, goal.CurrentAmount, time.Now())

	return &pb.UpdateGoalProgressResponse{
		Goal: goalToProto(goal),
	}, nil
}

// GetProgressHistory returns goal + snapshot history for custom HTTP endpoints.
func (h *GoalHandler) GetProgressHistory(ctx context.Context, userID, goalID uuid.UUID) (*model.SavingGoal, []model.GoalProgressSnapshot, []model.GoalContribution, error) {
	goal, err := h.goalRepo.GetByID(ctx, goalID, userID)
	if err != nil {
		return nil, nil, nil, err
	}

	snapshots, err := h.goalRepo.GetProgressSnapshots(ctx, goalID)
	if err != nil {
		return nil, nil, nil, err
	}

	contributions, err := h.goalRepo.GetContributions(ctx, goalID)
	if err != nil {
		return nil, nil, nil, err
	}

	return goal, snapshots, contributions, nil
}

func (h *GoalHandler) DeleteSavingGoal(ctx context.Context, req *pb.DeleteSavingGoalRequest) (*pb.DeleteSavingGoalResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid goal ID")
	}

	if err := h.goalRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			return nil, status.Error(codes.NotFound, "saving goal not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete saving goal")
	}

	return &pb.DeleteSavingGoalResponse{}, nil
}

func (h *GoalHandler) GetGoalProgress(ctx context.Context, req *pb.GetGoalProgressRequest) (*pb.GetGoalProgressResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid goal ID")
	}

	goal, err := h.goalRepo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			return nil, status.Error(codes.NotFound, "saving goal not found")
		}
		return nil, status.Error(codes.Internal, "failed to get saving goal")
	}

	return &pb.GetGoalProgressResponse{
		Progress: goalProgressToProto(goal),
	}, nil
}

func (h *GoalHandler) GetAllGoalsProgress(ctx context.Context, req *pb.GetAllGoalsProgressRequest) (*pb.GetAllGoalsProgressResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	goals, err := h.goalRepo.List(ctx, userID, req.IncludeCompleted)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list saving goals")
	}

	progress := make([]*pb.GoalProgress, len(goals))
	for i, g := range goals {
		progress[i] = goalProgressToProto(&g)
	}

	return &pb.GetAllGoalsProgressResponse{
		Progress: progress,
	}, nil
}

// Helper functions

func goalToProto(g *model.SavingGoal) *pb.SavingGoal {
	linkedAssetIDs := make([]string, len(g.LinkedAssetIDs))
	for i, id := range g.LinkedAssetIDs {
		linkedAssetIDs[i] = id.String()
	}

	pbGoal := &pb.SavingGoal{
		Id:   g.ID.String(),
		Name: g.Name,
		TargetAmount: &pb.Money{
			Amount:   g.TargetAmount.String(),
			Currency: g.Currency,
		},
		CurrentAmount: &pb.Money{
			Amount:   g.CurrentAmount.String(),
			Currency: g.Currency,
		},
		LinkedAssetIds: linkedAssetIDs,
		Notes:          g.Notes,
		CreatedAt:      timestamppb.New(g.CreatedAt),
	}

	if g.Deadline != nil {
		pbGoal.Deadline = timestamppb.New(*g.Deadline)
	}

	return pbGoal
}

func goalProgressToProto(g *model.SavingGoal) *pb.GoalProgress {
	daysRemaining := calculateDaysRemaining(g.Deadline)
	requiredMonthlySaving := calculateRequiredMonthlySaving(g, daysRemaining)
	isOnTrack, statusMessage := calculateOnTrackStatus(g, daysRemaining, requiredMonthlySaving)

	return &pb.GoalProgress{
		Goal:               goalToProto(g),
		PercentageComplete: g.PercentageComplete(),
		AmountRemaining: &pb.Money{
			Amount:   g.AmountRemaining().String(),
			Currency: g.Currency,
		},
		DaysRemaining: int32(daysRemaining),
		RequiredMonthlySaving: &pb.Money{
			Amount:   requiredMonthlySaving.String(),
			Currency: g.Currency,
		},
		IsOnTrack:     isOnTrack,
		StatusMessage: statusMessage,
	}
}

func calculateDaysRemaining(deadline *time.Time) int {
	if deadline == nil {
		return -1 // No deadline set
	}

	now := time.Now()
	if deadline.Before(now) {
		return 0 // Deadline has passed
	}

	return int(deadline.Sub(now).Hours() / 24)
}

func calculateRequiredMonthlySaving(g *model.SavingGoal, daysRemaining int) decimal.Decimal {
	remaining := g.AmountRemaining()
	if remaining.IsZero() || remaining.IsNegative() {
		return decimal.Zero
	}

	if daysRemaining <= 0 {
		// Deadline passed or no deadline - return full remaining amount
		return remaining
	}

	// Calculate months remaining (approximate: 30.44 days per month)
	monthsRemaining := float64(daysRemaining) / 30.44
	if monthsRemaining < 1 {
		monthsRemaining = 1 // At least 1 month to avoid division issues
	}

	return remaining.Div(decimal.NewFromFloat(monthsRemaining)).Round(2)
}

func calculateOnTrackStatus(g *model.SavingGoal, daysRemaining int, requiredMonthlySaving decimal.Decimal) (bool, string) {
	percentComplete := g.PercentageComplete()

	// Goal completed
	if percentComplete >= 100 {
		return true, "Goal completed!"
	}

	// No deadline set
	if g.Deadline == nil {
		return true, "No deadline set"
	}

	// Deadline passed
	if daysRemaining <= 0 {
		return false, "Deadline passed"
	}

	// Calculate expected progress based on time elapsed
	totalDays := g.Deadline.Sub(g.CreatedAt).Hours() / 24
	if totalDays <= 0 {
		return true, "On track"
	}

	daysElapsed := totalDays - float64(daysRemaining)
	expectedPercent := (daysElapsed / totalDays) * 100

	// Allow 10% variance
	if percentComplete >= expectedPercent-10 {
		return true, "On track"
	}

	// Calculate how behind
	behindPercent := math.Round(expectedPercent - percentComplete)
	return false, "Behind schedule (" + decimal.NewFromFloat(behindPercent).String() + "% behind)"
}

// ========== Net Worth Goal Methods ==========

// SetNetWorthGoal creates or updates the user's net worth goal
func (h *GoalHandler) SetNetWorthGoal(ctx context.Context, req *pb.SetNetWorthGoalRequest) (*pb.SetNetWorthGoalResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.TargetAmount == nil {
		return nil, status.Error(codes.InvalidArgument, "target_amount is required")
	}

	targetAmount, err := decimal.NewFromString(req.TargetAmount.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_amount")
	}

	if targetAmount.IsNegative() || targetAmount.IsZero() {
		return nil, status.Error(codes.InvalidArgument, "target_amount must be positive")
	}

	currency := req.TargetAmount.Currency
	if currency == "" {
		currency = "SGD"
	}

	goal := &model.NetWorthGoal{
		UserID:       userID,
		Name:         req.Name,
		TargetAmount: targetAmount,
		Currency:     currency,
		Notes:        req.Notes,
	}

	if err := h.goalRepo.SetNetWorthGoal(ctx, goal); err != nil {
		return nil, status.Error(codes.Internal, "failed to set net worth goal")
	}

	return &pb.SetNetWorthGoalResponse{
		Goal: netWorthGoalToProto(goal),
	}, nil
}

// GetNetWorthGoal retrieves the user's net worth goal
func (h *GoalHandler) GetNetWorthGoal(ctx context.Context, req *pb.GetNetWorthGoalRequest) (*pb.GetNetWorthGoalResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	goal, err := h.goalRepo.GetNetWorthGoal(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNetWorthGoalNotFound) {
			// Return empty response if no goal set
			return &pb.GetNetWorthGoalResponse{Goal: nil}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get net worth goal")
	}

	return &pb.GetNetWorthGoalResponse{
		Goal: netWorthGoalToProto(goal),
	}, nil
}

// DeleteNetWorthGoal deletes the user's net worth goal
func (h *GoalHandler) DeleteNetWorthGoal(ctx context.Context, req *pb.DeleteNetWorthGoalRequest) (*pb.DeleteNetWorthGoalResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.goalRepo.DeleteNetWorthGoal(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNetWorthGoalNotFound) {
			return nil, status.Error(codes.NotFound, "net worth goal not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete net worth goal")
	}

	return &pb.DeleteNetWorthGoalResponse{}, nil
}

// GetNetWorthGoalProgress retrieves progress toward the net worth goal
func (h *GoalHandler) GetNetWorthGoalProgress(ctx context.Context, req *pb.GetNetWorthGoalProgressRequest) (*pb.GetNetWorthGoalProgressResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	goal, err := h.goalRepo.GetNetWorthGoal(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNetWorthGoalNotFound) {
			// Return empty response if no goal set
			return &pb.GetNetWorthGoalProgressResponse{Progress: nil}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get net worth goal")
	}

	// Get current net worth
	totalAssets, err := h.assetRepo.GetTotalAssets(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total assets")
	}

	totalLiabilities, err := h.assetRepo.GetTotalLiabilities(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total liabilities")
	}

	currentNetWorth := totalAssets.Sub(totalLiabilities)

	// Calculate estimated months to goal based on average monthly savings
	estimatedMonths := calculateEstimatedMonthsToGoal(ctx, h.transactionRepo, userID, goal, currentNetWorth)

	return &pb.GetNetWorthGoalProgressResponse{
		Progress: netWorthGoalProgressToProto(goal, currentNetWorth, estimatedMonths),
	}, nil
}

// Helper functions for NetWorthGoal

func netWorthGoalToProto(g *model.NetWorthGoal) *pb.NetWorthGoal {
	return &pb.NetWorthGoal{
		Id:   g.ID.String(),
		Name: g.Name,
		TargetAmount: &pb.Money{
			Amount:   g.TargetAmount.String(),
			Currency: g.Currency,
		},
		Notes:     g.Notes,
		CreatedAt: timestamppb.New(g.CreatedAt),
		UpdatedAt: timestamppb.New(g.UpdatedAt),
	}
}

func netWorthGoalProgressToProto(g *model.NetWorthGoal, currentNetWorth decimal.Decimal, estimatedMonths int32) *pb.NetWorthGoalProgress {
	return &pb.NetWorthGoalProgress{
		Goal: netWorthGoalToProto(g),
		CurrentNetWorth: &pb.Money{
			Amount:   currentNetWorth.String(),
			Currency: g.Currency,
		},
		PercentageComplete: g.PercentageComplete(currentNetWorth),
		AmountRemaining: &pb.Money{
			Amount:   g.AmountRemaining(currentNetWorth).String(),
			Currency: g.Currency,
		},
		EstimatedMonthsToGoal: estimatedMonths,
	}
}

func calculateEstimatedMonthsToGoal(ctx context.Context, transactionRepo *repository.TransactionRepository, userID uuid.UUID, goal *model.NetWorthGoal, currentNetWorth decimal.Decimal) int32 {
	remaining := goal.AmountRemaining(currentNetWorth)
	if remaining.IsZero() || remaining.IsNegative() {
		return 0 // Goal reached
	}

	// Calculate average monthly savings from last 3 months
	now := time.Now()
	threeMonthsAgo := now.AddDate(0, -3, 0)

	totalIncome, err := transactionRepo.GetTotalByType(ctx, userID, threeMonthsAgo, now, model.CategoryTypeIncome)
	if err != nil {
		return -1 // Unable to calculate
	}

	totalExpenses, err := transactionRepo.GetTotalByType(ctx, userID, threeMonthsAgo, now, model.CategoryTypeExpense)
	if err != nil {
		return -1 // Unable to calculate
	}

	totalSavings := totalIncome.Sub(totalExpenses)
	monthlyAvgSavings := totalSavings.Div(decimal.NewFromInt(3))

	if monthlyAvgSavings.IsNegative() || monthlyAvgSavings.IsZero() {
		return -1 // Can't reach goal with negative or zero savings
	}

	months := remaining.Div(monthlyAvgSavings).IntPart()
	if months > math.MaxInt32 {
		return -1
	}

	return int32(months)
}

package handler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

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

type CPFHandler struct {
	pb.UnimplementedCPFServiceServer
	cpfRepo *repository.CPFRepository
}

func NewCPFHandler(cpfRepo *repository.CPFRepository) *CPFHandler {
	return &CPFHandler{
		cpfRepo: cpfRepo,
	}
}

func (h *CPFHandler) GetCPFAccount(ctx context.Context, req *pb.GetCPFAccountRequest) (*pb.GetCPFAccountResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	account, err := h.cpfRepo.GetAccount(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCPFAccountNotFound) {
			// Return empty account
			return &pb.GetCPFAccountResponse{
				Account: &pb.CPFAccount{
					UserId:       userID.String(),
					OaBalance:    "0",
					SaBalance:    "0",
					MaBalance:    "0",
					TotalBalance: "0",
				},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get CPF account")
	}

	return &pb.GetCPFAccountResponse{
		Account: cpfAccountToProto(account),
	}, nil
}

func (h *CPFHandler) UpdateCPFBalances(ctx context.Context, req *pb.UpdateCPFBalancesRequest) (*pb.UpdateCPFBalancesResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	oaBalance, err := decimal.NewFromString(req.OaBalance)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid oa_balance")
	}

	saBalance, err := decimal.NewFromString(req.SaBalance)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sa_balance")
	}

	maBalance, err := decimal.NewFromString(req.MaBalance)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid ma_balance")
	}

	account := &model.CPFAccount{
		UserID:    userID,
		OABalance: oaBalance,
		SABalance: saBalance,
		MABalance: maBalance,
	}

	if err := h.cpfRepo.CreateOrUpdateAccount(ctx, account); err != nil {
		return nil, status.Error(codes.Internal, "failed to update CPF balances")
	}

	return &pb.UpdateCPFBalancesResponse{
		Account: cpfAccountToProto(account),
	}, nil
}

func (h *CPFHandler) RecordCPFContribution(ctx context.Context, req *pb.RecordCPFContributionRequest) (*pb.RecordCPFContributionResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.ContributionMonth == "" {
		return nil, status.Error(codes.InvalidArgument, "contribution_month is required")
	}

	employeeAmount, err := decimal.NewFromString(req.EmployeeAmount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid employee_amount")
	}

	employerAmount, err := decimal.NewFromString(req.EmployerAmount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid employer_amount")
	}

	oaAmount, err := decimal.NewFromString(req.OaAmount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid oa_amount")
	}

	saAmount, err := decimal.NewFromString(req.SaAmount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sa_amount")
	}

	maAmount, err := decimal.NewFromString(req.MaAmount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid ma_amount")
	}

	contribution := &model.CPFContribution{
		UserID:            userID,
		ContributionMonth: req.ContributionMonth,
		EmployeeAmount:    employeeAmount,
		EmployerAmount:    employerAmount,
		OAAmount:          oaAmount,
		SAAmount:          saAmount,
		MAAmount:          maAmount,
	}

	if err := h.cpfRepo.RecordContribution(ctx, contribution); err != nil {
		return nil, status.Error(codes.Internal, "failed to record CPF contribution")
	}

	return &pb.RecordCPFContributionResponse{
		Contribution: cpfContributionToProto(contribution),
	}, nil
}

func (h *CPFHandler) ListCPFContributions(ctx context.Context, req *pb.ListCPFContributionsRequest) (*pb.ListCPFContributionsResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	var year *int
	if req.Year > 0 {
		y := int(req.Year)
		year = &y
	}

	contributions, err := h.cpfRepo.ListContributions(ctx, userID, year)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list CPF contributions")
	}

	employee, employer, total, err := h.cpfRepo.GetContributionTotals(ctx, userID, year)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get contribution totals")
	}

	pbContributions := make([]*pb.CPFContribution, len(contributions))
	for i, c := range contributions {
		pbContributions[i] = cpfContributionToProto(&c)
	}

	return &pb.ListCPFContributionsResponse{
		Contributions:      pbContributions,
		TotalEmployee:      employee.String(),
		TotalEmployer:      employer.String(),
		TotalContributions: total.String(),
	}, nil
}

func (h *CPFHandler) DeleteCPFContribution(ctx context.Context, req *pb.DeleteCPFContributionRequest) (*pb.DeleteCPFContributionResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contribution ID")
	}

	if err := h.cpfRepo.DeleteContribution(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrCPFContributionNotFound) {
			return nil, status.Error(codes.NotFound, "CPF contribution not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete CPF contribution")
	}

	return &pb.DeleteCPFContributionResponse{}, nil
}

func (h *CPFHandler) GetCPFProjection(ctx context.Context, req *pb.GetCPFProjectionRequest) (*pb.GetCPFProjectionResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	account, err := h.cpfRepo.GetAccount(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrCPFAccountNotFound) {
		return nil, status.Error(codes.Internal, "failed to get CPF account")
	}

	if account == nil {
		account = &model.CPFAccount{}
	}

	currentAge := int(req.CurrentAge)
	targetAge := int(req.TargetAge)
	if targetAge <= currentAge {
		return nil, status.Error(codes.InvalidArgument, "target_age must be greater than current_age")
	}

	yearsToTarget := targetAge - currentAge

	// CPF interest rates
	oaRate := decimal.NewFromFloat(0.025)  // 2.5% for OA
	saMaRate := decimal.NewFromFloat(0.04) // 4% for SA and MA

	// Monthly contribution (if provided)
	monthlyContribution := decimal.Zero
	if req.MonthlyContribution != "" {
		monthlyContribution, _ = decimal.NewFromString(req.MonthlyContribution)
	}

	// Calculate projections using compound interest
	// FV = PV * (1 + r)^n + PMT * ((1 + r)^n - 1) / r
	oaFV := calculateFutureValue(account.OABalance, oaRate, monthlyContribution.Mul(decimal.NewFromFloat(0.23)), yearsToTarget)
	saFV := calculateFutureValue(account.SABalance, saMaRate, monthlyContribution.Mul(decimal.NewFromFloat(0.06)), yearsToTarget)
	maFV := calculateFutureValue(account.MABalance, saMaRate, monthlyContribution.Mul(decimal.NewFromFloat(0.08)), yearsToTarget)

	totalFV := oaFV.Add(saFV).Add(maFV)

	return &pb.GetCPFProjectionResponse{
		Projection: &pb.CPFProjection{
			TargetAge:        strconv.Itoa(targetAge),
			ProjectedOa:      oaFV.Round(2).String(),
			ProjectedSa:      saFV.Round(2).String(),
			ProjectedMa:      maFV.Round(2).String(),
			ProjectedTotal:   totalFV.Round(2).String(),
			OaInterestRate:   "2.5%",
			SaMaInterestRate: "4.0%",
		},
	}, nil
}

func calculateFutureValue(presentValue, annualRate, monthlyContribution decimal.Decimal, years int) decimal.Decimal {
	// Simple calculation with annual compounding
	rate := annualRate.InexactFloat64()
	pv := presentValue.InexactFloat64()
	pmt := monthlyContribution.InexactFloat64() * 12 // Convert to annual

	// FV = PV * (1 + r)^n
	fvPV := pv * math.Pow(1+rate, float64(years))

	// FV of annuity = PMT * ((1 + r)^n - 1) / r
	fvPMT := float64(0)
	if rate > 0 {
		fvPMT = pmt * (math.Pow(1+rate, float64(years)) - 1) / rate
	}

	return decimal.NewFromFloat(fvPV + fvPMT)
}

func cpfAccountToProto(a *model.CPFAccount) *pb.CPFAccount {
	return &pb.CPFAccount{
		Id:           a.ID.String(),
		UserId:       a.UserID.String(),
		OaBalance:    a.OABalance.String(),
		SaBalance:    a.SABalance.String(),
		MaBalance:    a.MABalance.String(),
		TotalBalance: a.TotalBalance().String(),
		UpdatedAt:    timestamppb.New(a.UpdatedAt),
	}
}

func cpfContributionToProto(c *model.CPFContribution) *pb.CPFContribution {
	return &pb.CPFContribution{
		Id:                c.ID.String(),
		ContributionMonth: c.ContributionMonth,
		EmployeeAmount:    c.EmployeeAmount.String(),
		EmployerAmount:    c.EmployerAmount.String(),
		TotalAmount:       c.TotalAmount().String(),
		OaAmount:          c.OAAmount.String(),
		SaAmount:          c.SAAmount.String(),
		MaAmount:          c.MAAmount.String(),
		CreatedAt:         timestamppb.New(c.CreatedAt),
	}
}

// Suppress unused warning
var _ = fmt.Sprintf

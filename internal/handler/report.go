package handler

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/middleware"
	"github.com/chaowen/budget/internal/model"
	"github.com/chaowen/budget/internal/repository"
)

// ReportHandler implements the ReportService gRPC service
type ReportHandler struct {
	pb.UnimplementedReportServiceServer
	transactionRepo *repository.TransactionRepository
	budgetRepo      *repository.BudgetRepository
	assetRepo       *repository.AssetRepository
	goalRepo        *repository.GoalRepository
	userRepo        *repository.UserRepository
	currencyRepo    *repository.CurrencyRepository
}

// NewReportHandler creates a new ReportHandler
func NewReportHandler(
	transactionRepo *repository.TransactionRepository,
	budgetRepo *repository.BudgetRepository,
	assetRepo *repository.AssetRepository,
	goalRepo *repository.GoalRepository,
	userRepo *repository.UserRepository,
	currencyRepo *repository.CurrencyRepository,
) *ReportHandler {
	return &ReportHandler{
		transactionRepo: transactionRepo,
		budgetRepo:      budgetRepo,
		assetRepo:       assetRepo,
		goalRepo:        goalRepo,
		userRepo:        userRepo,
		currencyRepo:    currencyRepo,
	}
}

// GetWeeklyReport returns a weekly spending report
func (h *ReportHandler) GetWeeklyReport(ctx context.Context, req *pb.GetWeeklyReportRequest) (*pb.GetWeeklyReportResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Determine the week boundaries
	var weekOf time.Time
	if req.GetWeekOf() != nil {
		weekOf = req.GetWeekOf().AsTime()
	} else {
		weekOf = time.Now()
	}

	weekStart, weekEnd := getWeekBounds(weekOf)

	// Get spending by category
	spendingByCategory, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, weekStart, weekEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get spending by category")
	}

	// Get total income and expenses
	totalIncome, err := h.transactionRepo.GetTotalByType(ctx, userID, weekStart, weekEnd, model.CategoryTypeIncome)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total income")
	}

	totalExpenses, err := h.transactionRepo.GetTotalByType(ctx, userID, weekStart, weekEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total expenses")
	}

	netSavings := totalIncome.Sub(totalExpenses)

	// Calculate daily average spending
	days := int(weekEnd.Sub(weekStart).Hours()/24) + 1
	dailyAverage := totalExpenses.Div(decimal.NewFromInt(int64(days)))

	// Get budget summaries
	budgets, err := h.budgetRepo.List(ctx, userID, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get budgets")
	}

	var budgetSummaries []*pb.BudgetSummary
	for _, budget := range budgets {
		if budget.PeriodType == model.PeriodTypeWeekly {
			budgetStatus, err := h.budgetRepo.GetBudgetStatus(ctx, &budget)
			if err != nil {
				continue
			}
			budgetSummaries = append(budgetSummaries, reportBudgetStatusToSummary(budgetStatus))
		}
	}

	// Build spending by category response
	pbSpending := spendingByCategoryToProto(spendingByCategory, totalExpenses)

	return &pb.GetWeeklyReportResponse{
		Report: &pb.WeeklyReport{
			WeekStart:            timestamppb.New(weekStart),
			WeekEnd:              timestamppb.New(weekEnd),
			TotalIncome:          reportDecimalToMoney(totalIncome, "SGD"),
			TotalExpenses:        reportDecimalToMoney(totalExpenses, "SGD"),
			NetSavings:           reportDecimalToMoney(netSavings, "SGD"),
			SpendingByCategory:   pbSpending,
			BudgetSummaries:      budgetSummaries,
			DailyAverageSpending: reportDecimalToMoney(dailyAverage, "SGD"),
		},
	}, nil
}

// GetMonthlyReport returns a monthly spending report
func (h *ReportHandler) GetMonthlyReport(ctx context.Context, req *pb.GetMonthlyReportRequest) (*pb.GetMonthlyReportResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Parse month or use current month
	var year, month int
	if req.GetMonth() != "" {
		parsedTime, err := time.Parse("2006-01", req.GetMonth())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid month format, expected YYYY-MM")
		}
		year = parsedTime.Year()
		month = int(parsedTime.Month())
	} else {
		now := time.Now()
		year = now.Year()
		month = int(now.Month())
	}

	monthStart, monthEnd := getMonthBounds(year, month)

	// Get spending by category
	spendingByCategory, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get spending by category")
	}

	// Get total income and expenses
	totalIncome, err := h.transactionRepo.GetTotalByType(ctx, userID, monthStart, monthEnd, model.CategoryTypeIncome)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total income")
	}

	totalExpenses, err := h.transactionRepo.GetTotalByType(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total expenses")
	}

	netSavings := totalIncome.Sub(totalExpenses)

	// Calculate savings rate
	var savingsRate float64
	if !totalIncome.IsZero() {
		savingsRate = netSavings.Div(totalIncome).InexactFloat64() * 100
	}

	// Calculate daily average
	days := monthEnd.Day()
	dailyAverage := totalExpenses.Div(decimal.NewFromInt(int64(days)))

	// Get previous month data for comparison
	prevMonthStart, prevMonthEnd := getMonthBounds(year, month-1)
	prevIncome, _ := h.transactionRepo.GetTotalByType(ctx, userID, prevMonthStart, prevMonthEnd, model.CategoryTypeIncome)
	prevExpenses, _ := h.transactionRepo.GetTotalByType(ctx, userID, prevMonthStart, prevMonthEnd, model.CategoryTypeExpense)

	incomeChange := totalIncome.Sub(prevIncome)
	expenseChange := totalExpenses.Sub(prevExpenses)

	var incomeChangePercentage, expenseChangePercentage float64
	if !prevIncome.IsZero() {
		incomeChangePercentage = incomeChange.Div(prevIncome).InexactFloat64() * 100
	}
	if !prevExpenses.IsZero() {
		expenseChangePercentage = expenseChange.Div(prevExpenses).InexactFloat64() * 100
	}

	// Get budget summaries
	budgets, err := h.budgetRepo.List(ctx, userID, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get budgets")
	}

	var budgetSummaries []*pb.BudgetSummary
	for _, budget := range budgets {
		if budget.PeriodType == model.PeriodTypeMonthly {
			budgetStatus, err := h.budgetRepo.GetBudgetStatus(ctx, &budget)
			if err != nil {
				continue
			}
			budgetSummaries = append(budgetSummaries, reportBudgetStatusToSummary(budgetStatus))
		}
	}

	pbSpending := spendingByCategoryToProto(spendingByCategory, totalExpenses)

	return &pb.GetMonthlyReportResponse{
		Report: &pb.MonthlyReport{
			Month:                   time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).Format("2006-01"),
			TotalIncome:             reportDecimalToMoney(totalIncome, "SGD"),
			TotalExpenses:           reportDecimalToMoney(totalExpenses, "SGD"),
			NetSavings:              reportDecimalToMoney(netSavings, "SGD"),
			SavingsRate:             savingsRate,
			SpendingByCategory:      pbSpending,
			BudgetSummaries:         budgetSummaries,
			DailyAverageSpending:    reportDecimalToMoney(dailyAverage, "SGD"),
			IncomeChange:            reportDecimalToMoney(incomeChange, "SGD"),
			ExpenseChange:           reportDecimalToMoney(expenseChange, "SGD"),
			IncomeChangePercentage:  incomeChangePercentage,
			ExpenseChangePercentage: expenseChangePercentage,
		},
	}, nil
}

// GetNetWorthReport returns a net worth report
func (h *ReportHandler) GetNetWorthReport(ctx context.Context, req *pb.GetNetWorthReportRequest) (*pb.GetNetWorthReportResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	asOf := time.Now()
	if req.GetAsOf() != nil {
		asOf = req.GetAsOf().AsTime()
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}

	assets, err := h.assetRepo.List(ctx, userID, nil, true)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list assets")
	}

	categoryTotals := make(map[model.AssetCategory]decimal.Decimal)
	categoryCounts := make(map[model.AssetCategory]int)
	totalAssets := decimal.Zero
	totalLiabilities := decimal.Zero

	for _, asset := range assets {
		converted, convErr := h.currencyRepo.ConvertAmount(ctx, asset.CurrentValue, asset.Currency, user.BaseCurrency)
		if convErr != nil {
			if errors.Is(convErr, repository.ErrExchangeRateNotFound) {
				return nil, status.Error(codes.FailedPrecondition, "missing exchange rate for net worth conversion")
			}
			return nil, status.Error(codes.Internal, "failed to convert asset value")
		}

		if asset.IsLiability {
			totalLiabilities = totalLiabilities.Add(converted)
			continue
		}

		totalAssets = totalAssets.Add(converted)
		categoryTotals[asset.Category] = categoryTotals[asset.Category].Add(converted)
		categoryCounts[asset.Category]++
	}

	netWorth := totalAssets.Sub(totalLiabilities)

	var assetBreakdown []*pb.AssetCategorySummary
	for category, total := range categoryTotals {
		var percentage float64
		if !totalAssets.IsZero() {
			percentage = total.Div(totalAssets).InexactFloat64() * 100
		}
		assetBreakdown = append(assetBreakdown, &pb.AssetCategorySummary{
			Category:          string(category),
			TotalValue:        reportDecimalToMoney(total, user.BaseCurrency),
			PercentageOfTotal: percentage,
			AssetCount:        int32(categoryCounts[category]),
		})
	}

	return &pb.GetNetWorthReportResponse{
		Report: &pb.NetWorthReport{
			AsOf:             timestamppb.New(asOf),
			TotalAssets:      reportDecimalToMoney(totalAssets, user.BaseCurrency),
			TotalLiabilities: reportDecimalToMoney(totalLiabilities, user.BaseCurrency),
			NetWorth:         reportDecimalToMoney(netWorth, user.BaseCurrency),
			BaseCurrency:     user.BaseCurrency,
			AssetBreakdown:   assetBreakdown,
		},
	}, nil
}

// GetSavingGoalsReport returns a report on all saving goals
func (h *ReportHandler) GetSavingGoalsReport(ctx context.Context, req *pb.GetSavingGoalsReportRequest) (*pb.GetSavingGoalsReportResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	goals, err := h.goalRepo.List(ctx, userID, req.GetIncludeCompleted())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list saving goals")
	}

	var goalReports []*pb.SavingGoalReport
	var onTrackCount, behindScheduleCount int32

	for _, goal := range goals {
		report := savingGoalToReport(&goal)
		goalReports = append(goalReports, report)

		if report.IsOnTrack {
			onTrackCount++
		} else {
			behindScheduleCount++
		}
	}

	return &pb.GetSavingGoalsReportResponse{
		Goals:               goalReports,
		TotalGoals:          int32(len(goals)),
		OnTrackCount:        onTrackCount,
		BehindScheduleCount: behindScheduleCount,
	}, nil
}

// GetBudgetTrackingReport returns a budget tracking report
func (h *ReportHandler) GetBudgetTrackingReport(ctx context.Context, req *pb.GetBudgetTrackingReportRequest) (*pb.GetBudgetTrackingReportResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	periodType := req.GetPeriodType()
	if periodType == pb.PeriodType_PERIOD_TYPE_UNSPECIFIED {
		periodType = pb.PeriodType_PERIOD_TYPE_MONTHLY
	}

	modelPeriodType := reportProtoToPeriodType(periodType)

	// Get period bounds
	now := time.Now()
	periodStart, periodEnd := repository.GetPeriodBounds(modelPeriodType, now, now)

	// Calculate progress
	totalDays := int(periodEnd.Sub(periodStart).Hours()/24) + 1
	daysElapsed := int(now.Sub(periodStart).Hours()/24) + 1
	if daysElapsed > totalDays {
		daysElapsed = totalDays
	}
	daysRemaining := totalDays - daysElapsed
	periodProgress := float64(daysElapsed) / float64(totalDays) * 100

	// Get budgets for this period type
	budgets, err := h.budgetRepo.List(ctx, userID, &modelPeriodType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list budgets")
	}

	var totalBudgeted, totalSpent decimal.Decimal
	var categoryDetails []*pb.BudgetSummary

	for _, budget := range budgets {
		budgetStatus, err := h.budgetRepo.GetBudgetStatus(ctx, &budget)
		if err != nil {
			continue
		}
		totalBudgeted = totalBudgeted.Add(budget.Amount)
		totalSpent = totalSpent.Add(budgetStatus.Spent)
		categoryDetails = append(categoryDetails, reportBudgetStatusToSummary(budgetStatus))
	}

	// Calculate expected spending based on period progress
	expectedSpent := totalBudgeted.Mul(decimal.NewFromFloat(periodProgress / 100))

	// Calculate budget utilization
	var budgetUtilization float64
	if !expectedSpent.IsZero() {
		budgetUtilization = totalSpent.Div(expectedSpent).InexactFloat64() * 100
	}

	isOnTrack := totalSpent.LessThanOrEqual(expectedSpent)

	// Generate status message
	var statusMessage string
	if isOnTrack {
		statusMessage = "You are on track with your budget"
	} else {
		overBy := totalSpent.Sub(expectedSpent)
		statusMessage = "You are over budget by " + overBy.StringFixed(2)
	}

	// Project end of period spending
	var projectedSpending decimal.Decimal
	if daysElapsed > 0 {
		dailyRate := totalSpent.Div(decimal.NewFromInt(int64(daysElapsed)))
		projectedSpending = dailyRate.Mul(decimal.NewFromInt(int64(totalDays)))
	}

	return &pb.GetBudgetTrackingReportResponse{
		Report: &pb.BudgetTrackingReport{
			PeriodType:                   periodType,
			PeriodStart:                  timestamppb.New(periodStart),
			PeriodEnd:                    timestamppb.New(periodEnd),
			DaysElapsed:                  int32(daysElapsed),
			DaysRemaining:                int32(daysRemaining),
			PeriodProgressPercentage:     periodProgress,
			TotalBudgeted:                reportDecimalToMoney(totalBudgeted, "SGD"),
			TotalSpent:                   reportDecimalToMoney(totalSpent, "SGD"),
			ExpectedSpent:                reportDecimalToMoney(expectedSpent, "SGD"),
			BudgetUtilization:            budgetUtilization,
			IsOnTrack:                    isOnTrack,
			StatusMessage:                statusMessage,
			ProjectedEndOfPeriodSpending: reportDecimalToMoney(projectedSpending, "SGD"),
			CategoryDetails:              categoryDetails,
		},
	}, nil
}

// GetSpendingTrend returns spending trend over time
func (h *ReportHandler) GetSpendingTrend(ctx context.Context, req *pb.GetSpendingTrendRequest) (*pb.GetSpendingTrendResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	months := int(req.GetMonths())
	if months <= 0 {
		months = 6
	}

	var categoryID *uuid.UUID
	if req.GetCategoryId() != "" {
		id, err := uuid.Parse(req.GetCategoryId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid category ID")
		}
		categoryID = &id
	}

	var trend []*pb.SpendingTrendPoint
	var total, min, max decimal.Decimal
	min = decimal.NewFromInt(math.MaxInt64)

	now := time.Now()
	for i := months - 1; i >= 0; i-- {
		monthStart, monthEnd := getMonthBounds(now.Year(), int(now.Month())-i)

		var monthTotal decimal.Decimal
		if categoryID != nil {
			// Get spending for specific category
			spending, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
			if err != nil {
				continue
			}
			for _, s := range spending {
				if s.CategoryID == *categoryID {
					monthTotal = s.Total
					break
				}
			}
		} else {
			monthTotal, err = h.transactionRepo.GetTotalByType(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
			if err != nil {
				continue
			}
		}

		trend = append(trend, &pb.SpendingTrendPoint{
			Month:  monthStart.Format("2006-01"),
			Amount: reportDecimalToMoney(monthTotal, "SGD"),
		})

		total = total.Add(monthTotal)
		if monthTotal.LessThan(min) {
			min = monthTotal
		}
		if monthTotal.GreaterThan(max) {
			max = monthTotal
		}
	}

	average := decimal.Zero
	if len(trend) > 0 {
		average = total.Div(decimal.NewFromInt(int64(len(trend))))
	}

	// Determine trend direction
	var trendDirection string
	if len(trend) >= 2 {
		first := reportMoneyToDecimal(trend[0].Amount)
		last := reportMoneyToDecimal(trend[len(trend)-1].Amount)
		diff := last.Sub(first)
		threshold := first.Mul(decimal.NewFromFloat(0.1)) // 10% threshold

		if diff.GreaterThan(threshold) {
			trendDirection = "increasing"
		} else if diff.LessThan(threshold.Neg()) {
			trendDirection = "decreasing"
		} else {
			trendDirection = "stable"
		}
	} else {
		trendDirection = "stable"
	}

	// Handle case where min was never set
	if min.Equal(decimal.NewFromInt(math.MaxInt64)) {
		min = decimal.Zero
	}

	return &pb.GetSpendingTrendResponse{
		Trend:          trend,
		Average:        reportDecimalToMoney(average, "SGD"),
		Min:            reportDecimalToMoney(min, "SGD"),
		Max:            reportDecimalToMoney(max, "SGD"),
		TrendDirection: trendDirection,
	}, nil
}

// GetNetWorthTrend returns net worth trend over time
func (h *ReportHandler) GetNetWorthTrend(ctx context.Context, req *pb.GetNetWorthTrendRequest) (*pb.GetNetWorthTrendResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	months := int(req.GetMonths())
	if months <= 0 {
		months = 12
	}

	var trend []*pb.NetWorthTrendPoint
	now := time.Now()
	for i := months - 1; i >= 0; i-- {
		_, monthEnd := getMonthBounds(now.Year(), int(now.Month())-i)

		totalAssets, totalLiabilities, err := h.assetRepo.GetTotalsAsOf(ctx, userID, monthEnd)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to get historical asset totals")
		}

		netWorth := totalAssets.Sub(totalLiabilities)
		trend = append(trend, &pb.NetWorthTrendPoint{
			Month:       monthEnd.Format("2006-01"),
			NetWorth:    reportDecimalToMoney(netWorth, "SGD"),
			Assets:      reportDecimalToMoney(totalAssets, "SGD"),
			Liabilities: reportDecimalToMoney(totalLiabilities, "SGD"),
		})
	}

	totalChange := decimal.Zero
	totalChangePercentage := 0.0
	if len(trend) >= 2 {
		first := reportMoneyToDecimal(trend[0].NetWorth)
		last := reportMoneyToDecimal(trend[len(trend)-1].NetWorth)
		totalChange = last.Sub(first)
		if !first.IsZero() {
			totalChangePercentage = totalChange.Div(first).InexactFloat64() * 100
		}
	}

	return &pb.GetNetWorthTrendResponse{
		Trend:                 trend,
		TotalChange:           reportDecimalToMoney(totalChange, "SGD"),
		TotalChangePercentage: totalChangePercentage,
	}, nil
}

// Helper functions

func getWeekBounds(t time.Time) (time.Time, time.Time) {
	// Week starts on Monday
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
	end := start.AddDate(0, 0, 6)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, t.Location())
	return start, end
}

func getMonthBounds(year, month int) (time.Time, time.Time) {
	// Handle negative months
	for month <= 0 {
		month += 12
		year--
	}
	for month > 12 {
		month -= 12
		year++
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return start, end
}

func reportDecimalToMoney(d decimal.Decimal, currency string) *pb.Money {
	return &pb.Money{
		Amount:   d.StringFixed(2),
		Currency: currency,
	}
}

func reportMoneyToDecimal(m *pb.Money) decimal.Decimal {
	if m == nil {
		return decimal.Zero
	}
	d, _ := decimal.NewFromString(m.Amount)
	return d
}

func spendingByCategoryToProto(spending []repository.CategorySpending, total decimal.Decimal) []*pb.SpendingByCategory {
	var result []*pb.SpendingByCategory
	for _, s := range spending {
		var percentage float64
		if !total.IsZero() {
			percentage = s.Total.Div(total).InexactFloat64() * 100
		}
		result = append(result, &pb.SpendingByCategory{
			CategoryId:       s.CategoryID.String(),
			CategoryName:     s.CategoryName,
			Amount:           reportDecimalToMoney(s.Total, "SGD"),
			Percentage:       percentage,
			TransactionCount: int32(s.Count),
		})
	}
	return result
}

func reportBudgetStatusToSummary(status *model.BudgetStatus) *pb.BudgetSummary {
	return &pb.BudgetSummary{
		CategoryId:     status.Budget.CategoryID.String(),
		CategoryName:   status.Budget.CategoryName,
		Budgeted:       reportDecimalToMoney(status.Budget.Amount, status.Budget.Currency),
		Spent:          reportDecimalToMoney(status.Spent, status.Budget.Currency),
		Remaining:      reportDecimalToMoney(status.Remaining, status.Budget.Currency),
		PercentageUsed: status.PercentUsed,
		IsOverBudget:   status.IsOverBudget,
	}
}

func savingGoalToReport(goal *model.SavingGoal) *pb.SavingGoalReport {
	report := &pb.SavingGoalReport{
		GoalId:             goal.ID.String(),
		GoalName:           goal.Name,
		TargetAmount:       reportDecimalToMoney(goal.TargetAmount, goal.Currency),
		CurrentAmount:      reportDecimalToMoney(goal.CurrentAmount, goal.Currency),
		PercentageComplete: goal.PercentageComplete(),
	}

	if goal.Deadline != nil {
		report.Deadline = timestamppb.New(*goal.Deadline)

		// Calculate days remaining
		daysRemaining := int32(goal.Deadline.Sub(time.Now()).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}
		report.DaysRemaining = daysRemaining

		// Calculate required monthly saving
		remaining := goal.AmountRemaining()
		monthsRemaining := float64(daysRemaining) / 30.0
		var requiredMonthly decimal.Decimal
		if monthsRemaining > 0 {
			requiredMonthly = remaining.Div(decimal.NewFromFloat(monthsRemaining))
		}
		report.RequiredMonthlySaving = reportDecimalToMoney(requiredMonthly, goal.Currency)

		// Simplified: assume current monthly saving equals required for on-track
		// In real implementation, this would be calculated from transaction history
		report.CurrentMonthlySaving = reportDecimalToMoney(requiredMonthly, goal.Currency)
		report.IsOnTrack = true
	} else {
		report.IsOnTrack = true
	}

	return report
}

func reportProtoToPeriodType(pt pb.PeriodType) model.PeriodType {
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

package handler

import (
	"context"
	"errors"
	"math"
	"strings"
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

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}
	baseCurrency := user.BaseCurrency

	weekStart, weekEnd := getWeekBounds(weekOf)

	// Get spending by category
	rawSpendingByCategory, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, weekStart, weekEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get spending by category")
	}
	spendingByCategory, _, err := h.sumCategorySpending(ctx, rawSpendingByCategory, baseCurrency)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to convert currencies")
	}

	// Get total income and expenses
	rawIncome, err := h.transactionRepo.GetTotalByType(ctx, userID, weekStart, weekEnd, model.CategoryTypeIncome)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total income")
	}
	totalIncome, _ := h.sumCurrencyAmounts(ctx, rawIncome, baseCurrency)

	rawExpenses, err := h.transactionRepo.GetTotalByType(ctx, userID, weekStart, weekEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total expenses")
	}
	totalExpenses, _ := h.sumCurrencyAmounts(ctx, rawExpenses, baseCurrency)

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
			TotalIncome:          reportDecimalToMoney(totalIncome, baseCurrency),
			TotalExpenses:        reportDecimalToMoney(totalExpenses, baseCurrency),
			NetSavings:           reportDecimalToMoney(netSavings, baseCurrency),
			SpendingByCategory:   pbSpending,
			BudgetSummaries:      budgetSummaries,
			DailyAverageSpending: reportDecimalToMoney(dailyAverage, baseCurrency),
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

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}
	baseCurrency := user.BaseCurrency

	monthStart, monthEnd := getMonthBounds(year, month)

	// Get spending by category
	rawSpendingByCategory, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get spending by category")
	}
	spendingByCategory, _, _ := h.sumCategorySpending(ctx, rawSpendingByCategory, baseCurrency)

	// Get total income and expenses
	rawTotalIncome, err := h.transactionRepo.GetTotalByType(ctx, userID, monthStart, monthEnd, model.CategoryTypeIncome)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total income")
	}
	totalIncome, _ := h.sumCurrencyAmounts(ctx, rawTotalIncome, baseCurrency)

	rawTotalExpenses, err := h.transactionRepo.GetTotalByType(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get total expenses")
	}
	totalExpenses, _ := h.sumCurrencyAmounts(ctx, rawTotalExpenses, baseCurrency)

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
	rawPrevIncome, _ := h.transactionRepo.GetTotalByType(ctx, userID, prevMonthStart, prevMonthEnd, model.CategoryTypeIncome)
	prevIncome, _ := h.sumCurrencyAmounts(ctx, rawPrevIncome, baseCurrency)
	rawPrevExpenses, _ := h.transactionRepo.GetTotalByType(ctx, userID, prevMonthStart, prevMonthEnd, model.CategoryTypeExpense)
	prevExpenses, _ := h.sumCurrencyAmounts(ctx, rawPrevExpenses, baseCurrency)

	incomeChange := totalIncome.Sub(prevIncome)
	expenseChange := totalExpenses.Sub(prevExpenses)

	var incomeChangePercentage, expenseChangePercentage float64
	if !prevIncome.IsZero() {
		incomeChangePercentage = incomeChange.Div(prevIncome).InexactFloat64() * 100
	}
	if !prevExpenses.IsZero() {
		expenseChangePercentage = expenseChange.Div(prevExpenses).InexactFloat64() * 100
	}

	// Get budget summaries for monthly + weekly budgets in this month range
	budgets, err := h.budgetRepo.List(ctx, userID, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get budgets")
	}

	rawSpendingForBudgets, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get budget spending by category")
	}
	spendingForBudgets, _, _ := h.sumCategorySpending(ctx, rawSpendingForBudgets, baseCurrency)

	spentByCategory := make(map[uuid.UUID]decimal.Decimal, len(spendingForBudgets))
	for _, s := range spendingForBudgets {
		spentByCategory[s.CategoryID] = s.Total
	}

	agg := buildBudgetSummariesForRange(
		budgets,
		spentByCategory,
		monthStart,
		monthEnd,
		func(periodType model.PeriodType) bool {
			return periodType == model.PeriodTypeMonthly || periodType == model.PeriodTypeWeekly
		},
		calculateBudgetAmountForMonthlyRange,
	)

	pbSpending := spendingByCategoryToProto(spendingByCategory, totalExpenses)

	return &pb.GetMonthlyReportResponse{
		Report: &pb.MonthlyReport{
			Month:                   time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).Format("2006-01"),
			TotalIncome:             reportDecimalToMoney(totalIncome, baseCurrency),
			TotalExpenses:           reportDecimalToMoney(totalExpenses, baseCurrency),
			NetSavings:              reportDecimalToMoney(netSavings, baseCurrency),
			SavingsRate:             savingsRate,
			SpendingByCategory:      pbSpending,
			BudgetSummaries:         agg.summaries,
			DailyAverageSpending:    reportDecimalToMoney(dailyAverage, baseCurrency),
			IncomeChange:            reportDecimalToMoney(incomeChange, baseCurrency),
			ExpenseChange:           reportDecimalToMoney(expenseChange, baseCurrency),
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

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}
	baseCurrency := user.BaseCurrency

	now := time.Now()

	selectedYear, selectedMonth, err := budgetTrackingContextFromRequest(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Get period bounds and as-of (yearly uses full-year budget vs YTD spent)
	var periodStart, periodEnd, asOf time.Time
	switch periodType {
	case pb.PeriodType_PERIOD_TYPE_MONTHLY:
		year := now.Year()
		month := int(now.Month())
		if selectedYear > 0 {
			year = selectedYear
		}
		if selectedMonth > 0 {
			month = selectedMonth
		}
		periodStart, periodEnd = getMonthBounds(year, month)
		asOf = minTime(now, periodEnd)

	case pb.PeriodType_PERIOD_TYPE_YEARLY:
		year := now.Year()
		if selectedYear > 0 {
			year = selectedYear
		}
		periodStart = time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
		periodEnd = time.Date(year, time.December, 31, 23, 59, 59, 999999999, time.Local)
		asOf = minTime(now, periodEnd)

	default:
		periodStart, periodEnd = repository.GetPeriodBounds(modelPeriodType, now, now)
		asOf = now
	}

	// Calculate progress
	totalDays := int(periodEnd.Sub(periodStart).Hours()/24) + 1
	daysElapsed := 0
	if !asOf.Before(periodStart) {
		daysElapsed = int(asOf.Sub(periodStart).Hours()/24) + 1
		if daysElapsed > totalDays {
			daysElapsed = totalDays
		}
	}
	daysRemaining := totalDays - daysElapsed
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	periodProgress := float64(daysElapsed) / float64(totalDays) * 100

	// Get budgets
	var budgetPeriodFilter *model.PeriodType
	if periodType == pb.PeriodType_PERIOD_TYPE_WEEKLY || periodType == pb.PeriodType_PERIOD_TYPE_DAILY {
		budgetPeriodFilter = &modelPeriodType
	}

	budgets, err := h.budgetRepo.List(ctx, userID, budgetPeriodFilter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list budgets")
	}

	rawSpendingByCategory, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, periodStart, asOf, model.CategoryTypeExpense)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get spending by category")
	}
	spendingByCategory, _, _ := h.sumCategorySpending(ctx, rawSpendingByCategory, baseCurrency)

	spentByCategory := make(map[uuid.UUID]decimal.Decimal, len(spendingByCategory))
	for _, s := range spendingByCategory {
		spentByCategory[s.CategoryID] = s.Total
	}

	includeBudget := func(period model.PeriodType) bool {
		switch periodType {
		case pb.PeriodType_PERIOD_TYPE_YEARLY:
			return true
		case pb.PeriodType_PERIOD_TYPE_MONTHLY:
			return period == model.PeriodTypeMonthly || period == model.PeriodTypeWeekly
		case pb.PeriodType_PERIOD_TYPE_WEEKLY:
			return period == model.PeriodTypeWeekly
		case pb.PeriodType_PERIOD_TYPE_DAILY:
			return period == model.PeriodTypeDaily
		default:
			return period == modelPeriodType
		}
	}

	budgetAmountCalculator := calculateBudgetAmountByCycles
	if periodType == pb.PeriodType_PERIOD_TYPE_YEARLY {
		budgetAmountCalculator = calculateBudgetAmountForYearlyRange
	} else if periodType == pb.PeriodType_PERIOD_TYPE_MONTHLY {
		budgetAmountCalculator = calculateBudgetAmountForMonthlyRange
	}

	agg := buildBudgetSummariesForRange(budgets, spentByCategory, periodStart, periodEnd, includeBudget, budgetAmountCalculator)

	totalBudgeted := agg.totalBudgeted
	totalSpent := agg.totalSpentUnique

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
			TotalBudgeted:                reportDecimalToMoney(totalBudgeted, baseCurrency),
			TotalSpent:                   reportDecimalToMoney(totalSpent, baseCurrency),
			ExpectedSpent:                reportDecimalToMoney(expectedSpent, baseCurrency),
			BudgetUtilization:            budgetUtilization,
			IsOnTrack:                    isOnTrack,
			StatusMessage:                statusMessage,
			ProjectedEndOfPeriodSpending: reportDecimalToMoney(projectedSpending, baseCurrency),
			CategoryDetails:              agg.summaries,
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

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}
	baseCurrency := user.BaseCurrency

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
			rawSpending, err := h.transactionRepo.GetSpendingByCategory(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
			if err != nil {
				continue
			}
			spending, _, _ := h.sumCategorySpending(ctx, rawSpending, baseCurrency)
			for _, s := range spending {
				if s.CategoryID == *categoryID {
					monthTotal = s.Total
					break
				}
			}
		} else {
			rawMonthTotal, err := h.transactionRepo.GetTotalByType(ctx, userID, monthStart, monthEnd, model.CategoryTypeExpense)
			if err != nil {
				continue
			}
			monthTotal, _ = h.sumCurrencyAmounts(ctx, rawMonthTotal, baseCurrency)
		}

		trend = append(trend, &pb.SpendingTrendPoint{
			Month:  monthStart.Format("2006-01"),
			Amount: reportDecimalToMoney(monthTotal, baseCurrency),
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
		Average:        reportDecimalToMoney(average, baseCurrency),
		Min:            reportDecimalToMoney(min, baseCurrency),
		Max:            reportDecimalToMoney(max, baseCurrency),
		TrendDirection: trendDirection,
	}, nil
}

// GetNetWorthTrend returns net worth trend over time
func (h *ReportHandler) GetNetWorthTrend(ctx context.Context, req *pb.GetNetWorthTrendRequest) (*pb.GetNetWorthTrendResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load user profile")
	}

	interval, selectedYear, selectedMonth, err := netWorthTrendOptionsFromRequest(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var trend []*pb.NetWorthTrendPoint
	rateCache := make(map[string]decimal.Decimal)
	now := time.Now()

	switch interval {
	case netWorthTrendIntervalDaily:
		targetYear := now.Year()
		targetMonth := int(now.Month())
		if selectedMonth != "" {
			parsedMonth, _ := time.Parse("2006-01", selectedMonth)
			targetYear = parsedMonth.Year()
			targetMonth = int(parsedMonth.Month())
		}

		monthStart, monthEnd := getMonthBounds(targetYear, targetMonth)
		for day := monthStart; !day.After(monthEnd); day = day.AddDate(0, 0, 1) {
			dayEnd := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, day.Location())
			totalAssets, totalLiabilities, totalsErr := h.getConvertedNetWorthTotalsAsOf(ctx, userID, user.BaseCurrency, dayEnd, rateCache)
			if totalsErr != nil {
				if errors.Is(totalsErr, repository.ErrExchangeRateNotFound) {
					return nil, status.Error(codes.FailedPrecondition, "missing exchange rate for net worth trend conversion")
				}
				return nil, status.Error(codes.Internal, "failed to get historical net worth totals")
			}

			netWorth := totalAssets.Sub(totalLiabilities)
			trend = append(trend, &pb.NetWorthTrendPoint{
				Month:       day.Format("2006-01-02"),
				NetWorth:    reportDecimalToMoney(netWorth, user.BaseCurrency),
				Assets:      reportDecimalToMoney(totalAssets, user.BaseCurrency),
				Liabilities: reportDecimalToMoney(totalLiabilities, user.BaseCurrency),
			})
		}

	case netWorthTrendIntervalMonthly:
		if selectedYear > 0 {
			if selectedYear > now.Year() {
				trend = []*pb.NetWorthTrendPoint{}
				break
			}

			endMonth := 12
			if selectedYear == now.Year() {
				endMonth = int(now.Month())
			}

			for month := 1; month <= endMonth; month++ {
				_, monthEnd := getMonthBounds(selectedYear, month)
				totalAssets, totalLiabilities, totalsErr := h.getConvertedNetWorthTotalsAsOf(ctx, userID, user.BaseCurrency, monthEnd, rateCache)
				if totalsErr != nil {
					if errors.Is(totalsErr, repository.ErrExchangeRateNotFound) {
						return nil, status.Error(codes.FailedPrecondition, "missing exchange rate for net worth trend conversion")
					}
					return nil, status.Error(codes.Internal, "failed to get historical net worth totals")
				}

				netWorth := totalAssets.Sub(totalLiabilities)
				trend = append(trend, &pb.NetWorthTrendPoint{
					Month:       monthEnd.Format("2006-01"),
					NetWorth:    reportDecimalToMoney(netWorth, user.BaseCurrency),
					Assets:      reportDecimalToMoney(totalAssets, user.BaseCurrency),
					Liabilities: reportDecimalToMoney(totalLiabilities, user.BaseCurrency),
				})
			}
			break
		}

		months := int(req.GetMonths())
		if months <= 0 {
			months = 12
		}

		for i := months - 1; i >= 0; i-- {
			_, monthEnd := getMonthBounds(now.Year(), int(now.Month())-i)
			totalAssets, totalLiabilities, totalsErr := h.getConvertedNetWorthTotalsAsOf(ctx, userID, user.BaseCurrency, monthEnd, rateCache)
			if totalsErr != nil {
				if errors.Is(totalsErr, repository.ErrExchangeRateNotFound) {
					return nil, status.Error(codes.FailedPrecondition, "missing exchange rate for net worth trend conversion")
				}
				return nil, status.Error(codes.Internal, "failed to get historical net worth totals")
			}

			netWorth := totalAssets.Sub(totalLiabilities)
			trend = append(trend, &pb.NetWorthTrendPoint{
				Month:       monthEnd.Format("2006-01"),
				NetWorth:    reportDecimalToMoney(netWorth, user.BaseCurrency),
				Assets:      reportDecimalToMoney(totalAssets, user.BaseCurrency),
				Liabilities: reportDecimalToMoney(totalLiabilities, user.BaseCurrency),
			})
		}

	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported trend interval")
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
		TotalChange:           reportDecimalToMoney(totalChange, user.BaseCurrency),
		TotalChangePercentage: totalChangePercentage,
	}, nil
}

func (h *ReportHandler) getConvertedNetWorthTotalsAsOf(
	ctx context.Context,
	userID uuid.UUID,
	baseCurrency string,
	asOf time.Time,
	rateCache map[string]decimal.Decimal,
) (decimal.Decimal, decimal.Decimal, error) {
	balances, err := h.assetRepo.ListBalancesAsOf(ctx, userID, asOf)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	totalAssets := decimal.Zero
	totalLiabilities := decimal.Zero

	for _, item := range balances {
		converted, convErr := convertAmountAsOfWithCache(ctx, h.currencyRepo, rateCache, item.Balance, item.Currency, baseCurrency, asOf)
		if convErr != nil {
			return decimal.Zero, decimal.Zero, convErr
		}

		if item.IsLiability {
			totalLiabilities = totalLiabilities.Add(converted)
			continue
		}

		totalAssets = totalAssets.Add(converted)
	}

	return totalAssets, totalLiabilities, nil
}

func convertAmountAsOfWithCache(
	ctx context.Context,
	currencyRepo *repository.CurrencyRepository,
	rateCache map[string]decimal.Decimal,
	amount decimal.Decimal,
	fromCurrency string,
	toCurrency string,
	asOf time.Time,
) (decimal.Decimal, error) {
	if fromCurrency == toCurrency {
		return amount.Round(2), nil
	}

	directKey := fromCurrency + ":" + toCurrency
	if rate, ok := rateCache[directKey]; ok {
		return amount.Mul(rate).Round(2), nil
	}

	inverseKey := toCurrency + ":" + fromCurrency
	if rate, ok := rateCache[inverseKey]; ok {
		if rate.IsZero() {
			return decimal.Zero, errors.New("invalid zero exchange rate")
		}
		return amount.Div(rate).Round(2), nil
	}

	rate, err := currencyRepo.GetExchangeRateAsOf(ctx, fromCurrency, toCurrency, asOf)
	if err == nil {
		rateCache[directKey] = rate.Rate
		return amount.Mul(rate.Rate).Round(2), nil
	}
	if !errors.Is(err, repository.ErrExchangeRateNotFound) {
		return decimal.Zero, err
	}

	inverseRate, invErr := currencyRepo.GetExchangeRateAsOf(ctx, toCurrency, fromCurrency, asOf)
	if invErr != nil {
		if errors.Is(invErr, repository.ErrExchangeRateNotFound) {
			return decimal.Zero, repository.ErrExchangeRateNotFound
		}
		return decimal.Zero, invErr
	}

	if inverseRate.Rate.IsZero() {
		return decimal.Zero, errors.New("invalid zero exchange rate")
	}

	rateCache[inverseKey] = inverseRate.Rate
	return amount.Div(inverseRate.Rate).Round(2), nil
}

const (
	netWorthTrendIntervalMonthly = "monthly"
	netWorthTrendIntervalDaily   = "daily"
)

func netWorthTrendOptionsFromRequest(req *pb.GetNetWorthTrendRequest) (string, int, string, error) {
	interval := netWorthTrendIntervalMonthly
	selectedYear := 0
	selectedMonth := ""

	if req.Interval != "" {
		normalized := strings.ToLower(strings.TrimSpace(req.Interval))
		switch normalized {
		case netWorthTrendIntervalMonthly, netWorthTrendIntervalDaily:
			interval = normalized
		default:
			return "", 0, "", errors.New("interval must be monthly or daily")
		}
	}

	if req.Year != 0 {
		if req.Year < 1970 || req.Year > 9999 {
			return "", 0, "", errors.New("year must be between 1970 and 9999")
		}
		selectedYear = int(req.Year)
	}

	if req.Month != "" {
		month := strings.TrimSpace(req.Month)
		if _, err := time.Parse("2006-01", month); err != nil {
			return "", 0, "", errors.New("month must use YYYY-MM format")
		}
		selectedMonth = month
	}

	return interval, selectedYear, selectedMonth, nil
}

type budgetSummaryAggregation struct {
	summaries        []*pb.BudgetSummary
	totalBudgeted    decimal.Decimal
	totalSpentUnique decimal.Decimal
}

func buildBudgetSummariesForRange(
	budgets []model.Budget,
	spentByCategory map[uuid.UUID]decimal.Decimal,
	periodStart time.Time,
	periodEnd time.Time,
	includeBudget func(model.PeriodType) bool,
	budgetAmountCalculator func(*model.Budget, time.Time, time.Time) decimal.Decimal,
) budgetSummaryAggregation {
	agg := budgetSummaryAggregation{
		summaries: make([]*pb.BudgetSummary, 0, len(budgets)),
	}
	seenCategories := make(map[uuid.UUID]bool)

	for _, budget := range budgets {
		if !includeBudget(budget.PeriodType) {
			continue
		}

		budgeted := budgetAmountCalculator(&budget, periodStart, periodEnd)
		spent := spentByCategory[budget.CategoryID]
		remaining := budgeted.Sub(spent)

		percentageUsed := 0.0
		if !budgeted.IsZero() {
			percentageUsed = spent.Div(budgeted).InexactFloat64() * 100
		}

		agg.summaries = append(agg.summaries, &pb.BudgetSummary{
			CategoryId:     budget.CategoryID.String(),
			CategoryName:   budget.CategoryName,
			Budgeted:       reportDecimalToMoney(budgeted, budget.Currency),
			Spent:          reportDecimalToMoney(spent, budget.Currency),
			Remaining:      reportDecimalToMoney(remaining, budget.Currency),
			PercentageUsed: percentageUsed,
			IsOverBudget:   remaining.IsNegative(),
		})

		agg.totalBudgeted = agg.totalBudgeted.Add(budgeted)
		if !seenCategories[budget.CategoryID] {
			agg.totalSpentUnique = agg.totalSpentUnique.Add(spent)
			seenCategories[budget.CategoryID] = true
		}
	}

	return agg
}

func calculateBudgetAmountForMonthlyRange(budget *model.Budget, periodStart, periodEnd time.Time) decimal.Decimal {
	if budget == nil || periodEnd.Before(periodStart) {
		return decimal.Zero
	}

	if budget.StartDate.After(periodEnd) {
		return decimal.Zero
	}

	rangeStart := startOfDay(periodStart)
	rangeEnd := endOfDay(periodEnd)
	budgetStart := startOfDay(budget.StartDate)

	activeStart := maxTime(rangeStart, budgetStart)
	activeDays := inclusiveDays(activeStart, rangeEnd)
	if activeDays <= 0 {
		return decimal.Zero
	}

	switch budget.PeriodType {
	case model.PeriodTypeWeekly:
		return budget.Amount.Div(decimal.NewFromInt(7)).Mul(decimal.NewFromInt(int64(activeDays)))
	case model.PeriodTypeMonthly:
		return budget.Amount
	default:
		return calculateBudgetAmountByCycles(budget, periodStart, periodEnd)
	}
}

func calculateBudgetAmountForYearlyRange(budget *model.Budget, periodStart, periodEnd time.Time) decimal.Decimal {
	if budget == nil || periodEnd.Before(periodStart) {
		return decimal.Zero
	}
	if budget.StartDate.After(periodEnd) {
		return decimal.Zero
	}

	switch budget.PeriodType {
	case model.PeriodTypeDaily:
		return budget.Amount.Mul(decimal.NewFromInt(int64(inclusiveDays(startOfDay(periodStart), endOfDay(periodEnd)))))
	case model.PeriodTypeWeekly:
		return budget.Amount.Mul(decimal.NewFromInt(52))
	case model.PeriodTypeMonthly:
		return budget.Amount.Mul(decimal.NewFromInt(12))
	case model.PeriodTypeYearly:
		return budget.Amount
	default:
		return decimal.Zero
	}
}

func calculateBudgetAmountByCycles(budget *model.Budget, periodStart, periodEnd time.Time) decimal.Decimal {
	if budget == nil || periodEnd.Before(periodStart) {
		return decimal.Zero
	}

	cycles := countBudgetCyclesInRange(budget.PeriodType, budget.StartDate, periodStart, periodEnd)
	if cycles <= 0 {
		return decimal.Zero
	}

	return budget.Amount.Mul(decimal.NewFromInt(int64(cycles)))
}

func inclusiveDays(start, end time.Time) int {
	if end.Before(start) {
		return 0
	}
	return int(end.Sub(start).Hours()/24) + 1
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func endOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 23, 59, 59, 999999999, value.Location())
}

func countBudgetCyclesInRange(periodType model.PeriodType, startDate, periodStart, periodEnd time.Time) int {
	rangeStart := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, periodStart.Location())
	rangeEnd := time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), 23, 59, 59, 999999999, periodEnd.Location())
	anchor := time.Date(startDate.In(rangeStart.Location()).Year(), startDate.In(rangeStart.Location()).Month(), startDate.In(rangeStart.Location()).Day(), 0, 0, 0, 0, rangeStart.Location())

	if rangeEnd.Before(anchor) {
		return 0
	}

	if periodType == model.PeriodTypeDaily {
		effectiveStart := rangeStart
		if anchor.After(effectiveStart) {
			effectiveStart = anchor
		}
		if rangeEnd.Before(effectiveStart) {
			return 0
		}
		return int(rangeEnd.Sub(effectiveStart).Hours()/24) + 1
	}

	cycleStart := anchor
	for cycleStart.Before(rangeStart) {
		next := nextBudgetCycleStart(periodType, cycleStart)
		if !next.After(cycleStart) {
			break
		}
		cycleStart = next
	}

	count := 0
	for !cycleStart.After(rangeEnd) {
		count++
		next := nextBudgetCycleStart(periodType, cycleStart)
		if !next.After(cycleStart) {
			break
		}
		cycleStart = next
	}

	return count
}

func nextBudgetCycleStart(periodType model.PeriodType, current time.Time) time.Time {
	switch periodType {
	case model.PeriodTypeWeekly:
		return current.AddDate(0, 0, 7)
	case model.PeriodTypeMonthly:
		return addMonthsPreserveDay(current, 1)
	case model.PeriodTypeYearly:
		return addYearsPreserveDay(current, 1)
	case model.PeriodTypeDaily:
		fallthrough
	default:
		return current.AddDate(0, 0, 1)
	}
}

func addMonthsPreserveDay(d time.Time, months int) time.Time {
	base := time.Date(d.Year(), d.Month(), 1, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	targetMonthStart := base.AddDate(0, months, 0)
	lastDay := time.Date(targetMonthStart.Year(), targetMonthStart.Month()+1, 0, 0, 0, 0, 0, d.Location()).Day()
	day := d.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetMonthStart.Year(), targetMonthStart.Month(), day, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
}

func addYearsPreserveDay(d time.Time, years int) time.Time {
	base := time.Date(d.Year()+years, d.Month(), 1, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
	lastDay := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, d.Location()).Day()
	day := d.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(base.Year(), base.Month(), day, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
}

func budgetTrackingContextFromRequest(req *pb.GetBudgetTrackingReportRequest) (int, int, error) {
	selectedYear := int(req.Year)
	if selectedYear != 0 {
		if selectedYear < 1970 || selectedYear > 9999 {
			return 0, 0, errors.New("year must be between 1970 and 9999")
		}
	}

	selectedMonth := int(req.Month)
	if selectedMonth != 0 {
		if selectedMonth < 1 || selectedMonth > 12 {
			return 0, 0, errors.New("month must be between 1 and 12")
		}
	}

	return selectedYear, selectedMonth, nil
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
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

func (h *ReportHandler) sumCurrencyAmounts(ctx context.Context, amounts []repository.CurrencyAmount, targetCurrency string) (decimal.Decimal, error) {
	total := decimal.Zero
	for _, ca := range amounts {
		converted, err := h.currencyRepo.ConvertAmount(ctx, ca.Amount, ca.Currency, targetCurrency)
		if err != nil {
			return decimal.Zero, err
		}
		total = total.Add(converted)
	}
	return total, nil
}

func (h *ReportHandler) sumCategorySpending(ctx context.Context, spending []repository.CategorySpending, targetCurrency string) ([]repository.CategorySpending, decimal.Decimal, error) {
	total := decimal.Zero
	categoryMap := make(map[uuid.UUID]repository.CategorySpending)
	for _, cs := range spending {
		converted, err := h.currencyRepo.ConvertAmount(ctx, cs.Total, cs.Currency, targetCurrency)
		if err != nil {
			return nil, decimal.Zero, err
		}

		if existing, ok := categoryMap[cs.CategoryID]; ok {
			existing.Total = existing.Total.Add(converted)
			existing.Count += cs.Count
			categoryMap[cs.CategoryID] = existing
		} else {
			cs.Total = converted
			cs.Currency = targetCurrency
			categoryMap[cs.CategoryID] = cs
		}
		total = total.Add(converted)
	}

	var result []repository.CategorySpending
	for _, cs := range categoryMap {
		result = append(result, cs)
	}
	return result, total, nil
}

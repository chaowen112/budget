package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/config"
	"github.com/chaowen/budget/internal/handler"
	"github.com/chaowen/budget/internal/metrics"
	"github.com/chaowen/budget/internal/middleware"
	"github.com/chaowen/budget/internal/model"
	"github.com/chaowen/budget/internal/pkg/currency"
	"github.com/chaowen/budget/internal/pkg/jwt"
	logger "github.com/chaowen/budget/internal/pkg/logger"
	"github.com/chaowen/budget/internal/repository"
	"github.com/chaowen/budget/internal/service"
)

type transferPayload struct {
	FromAssetID  string `json:"fromAssetId"`
	ToAssetID    string `json:"toAssetId"`
	FromAmount   string `json:"fromAmount"`
	ToAmount     string `json:"toAmount"`
	FromCurrency string `json:"fromCurrency"`
	ToCurrency   string `json:"toCurrency"`
	TransferDate string `json:"transferDate"`
	Description  string `json:"description"`
}

//go:embed static/*
var staticFiles embed.FS

//go:embed swagger-ui/*
var swaggerUI embed.FS

//go:embed openapi/*
var openAPISpec embed.FS

func main() {
	if err := run(); err != nil {
		log.WithError(err).Fatal("server error")
	}
}

func run() error {
	// Load configuration
	cfg := config.Load()

	// Initialize structured logger
	logger.Init(cfg.LogLevel, cfg.LogDir)
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02T15:04:05.000Z07:00"})
	if lvl, err := log.ParseLevel(cfg.LogLevel); err == nil {
		log.SetLevel(lvl)
	}
	log.SetOutput(logger.Log.Out)

	log.WithFields(log.Fields{"env": cfg.Env, "log_level": cfg.LogLevel, "log_dir": cfg.LogDir}).Info("Configuration loaded")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := repository.NewDB(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	log.Info("Connected to database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	apiKeyRepo := repository.NewApiKeyRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)
	assetRepo := repository.NewAssetRepository(db)
	goalRepo := repository.NewGoalRepository(db)
	accountingRepo := repository.NewAccountingRepository(db)
	transferRepo := repository.NewTransferRepository(db)
	currencyRepo := repository.NewCurrencyRepository(db)
	cpfRepo := repository.NewCPFRepository(db)

	metricsCollector := metrics.NewCollector(prometheus.DefaultRegisterer)
	nodeName, hostErr := os.Hostname()
	if hostErr != nil {
		nodeName = "unknown"
	}
	metricsCollector.StartBusinessMetricsUpdater(ctx, db, nodeName, 30*time.Second)

	// Initialize JWT manager
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	// Initialize services
	userService := service.NewUserService(userRepo, apiKeyRepo, jwtManager)
	transactionAssistantService := service.NewTransactionAssistantService(
		cfg.AI.Provider,
		cfg.AI.APIKey,
		cfg.AI.Model,
		cfg.AI.BaseURL,
	)

	// Initialize external clients
	currencyClient := currency.NewClient(cfg.Exchange.APIKey)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	categoryHandler := handler.NewCategoryHandler(categoryRepo)
	transactionHandler := handler.NewTransactionHandler(transactionRepo, categoryRepo, assetRepo, accountingRepo)
	budgetHandler := handler.NewBudgetHandler(budgetRepo)
	assetHandler := handler.NewAssetHandler(assetRepo, accountingRepo, userRepo, currencyRepo)
	goalHandler := handler.NewGoalHandler(goalRepo, assetRepo, transactionRepo, currencyRepo)
	currencyHandler := handler.NewCurrencyHandler(currencyRepo, currencyClient)
	cpfHandler := handler.NewCPFHandler(cpfRepo)
	reportHandler := handler.NewReportHandler(transactionRepo, budgetRepo, assetRepo, goalRepo, userRepo, currencyRepo)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metricsCollector.GRPCUnaryInterceptor(),
			middleware.AuthInterceptor(jwtManager, apiKeyRepo, metricsCollector, middleware.PublicMethods()),
		),
	)

	// Register gRPC services
	pb.RegisterUserServiceServer(grpcServer, userHandler)
	pb.RegisterCategoryServiceServer(grpcServer, categoryHandler)
	pb.RegisterTransactionServiceServer(grpcServer, transactionHandler)
	pb.RegisterBudgetServiceServer(grpcServer, budgetHandler)
	pb.RegisterAssetServiceServer(grpcServer, assetHandler)
	pb.RegisterSavingGoalServiceServer(grpcServer, goalHandler)
	pb.RegisterCurrencyServiceServer(grpcServer, currencyHandler)
	pb.RegisterCPFServiceServer(grpcServer, cpfHandler)
	pb.RegisterReportServiceServer(grpcServer, reportHandler)

	// Enable reflection for development
	if cfg.Env == "development" {
		reflection.Register(grpcServer)
	}

	// Start gRPC server
	grpcAddr := ":" + cfg.Server.GRPCPort
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port: %w", err)
	}

	go func() {
		log.WithField("addr", grpcAddr).Info("gRPC server listening")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.WithError(err).Error("gRPC server error")
		}
	}()

	// Create gRPC-Gateway mux
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcEndpoint := "localhost" + grpcAddr

	// Register gRPC-Gateway handlers
	if err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register user gateway: %w", err)
	}
	if err := pb.RegisterCategoryServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register category gateway: %w", err)
	}
	if err := pb.RegisterTransactionServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register transaction gateway: %w", err)
	}
	if err := pb.RegisterBudgetServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register budget gateway: %w", err)
	}
	if err := pb.RegisterAssetServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register asset gateway: %w", err)
	}
	if err := pb.RegisterSavingGoalServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register saving goal gateway: %w", err)
	}
	if err := pb.RegisterCurrencyServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register currency gateway: %w", err)
	}
	if err := pb.RegisterCPFServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register CPF gateway: %w", err)
	}
	if err := pb.RegisterReportServiceHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register report gateway: %w", err)
	}

	// Create HTTP server with CORS support
	httpMux := http.NewServeMux()

	// Expose Prometheus metrics
	httpMux.Handle("/metrics", promhttp.Handler())

	// Basic health endpoint for probes
	httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Serve OpenAPI spec
	httpMux.HandleFunc("/api/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		spec, err := openAPISpec.ReadFile("openapi/budget.swagger.json")
		if err != nil {
			http.Error(w, "OpenAPI spec not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(spec)
	})

	// Serve Swagger UI
	swaggerFS, err := fs.Sub(swaggerUI, "swagger-ui")
	if err != nil {
		return fmt.Errorf("failed to create swagger-ui sub filesystem: %w", err)
	}
	httpMux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS))))

	// Serve static UI files (React app)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to create static sub filesystem: %w", err)
	}

	// Handle SPA routing - serve index.html for non-API routes
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API routes go to gRPC gateway
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if serveGoalHistory(w, r, jwtManager, apiKeyRepo, goalHandler) {
				return
			}
			if serveAccounting(w, r, jwtManager, apiKeyRepo, accountingRepo) {
				return
			}
			if serveTransfers(w, r, jwtManager, apiKeyRepo, transferRepo, assetRepo, accountingRepo) {
				return
			}
			if serveTransactionAssistant(w, r, jwtManager, apiKeyRepo, transactionAssistantService) {
				return
			}
			if serveTransactionSourceLinks(w, r, jwtManager, apiKeyRepo, transactionRepo) {
				return
			}
			withCORS(gwMux).ServeHTTP(w, r)
			return
		}

		// Try to serve static file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists in static FS
		f, err := staticFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
			return
		}

		// SPA fallback - serve index.html for client-side routing
		indexHTML, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	httpAddr := ":" + cfg.Server.HTTPPort
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: metricsCollector.HTTPMiddleware(httpMux),
	}

	go func() {
		log.WithField("addr", httpAddr).Info("HTTP server (gRPC-Gateway) listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Error("HTTP server error")
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.WithField("signal", sig.String()).Info("Shutting down servers...")

	// Graceful shutdown
	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("HTTP server shutdown error")
	}

	log.Info("Servers stopped")
	return nil
}

func serveGoalHistory(w http.ResponseWriter, r *http.Request, jwtManager *jwt.Manager, apiKeyRepo *repository.ApiKeyRepository, goalHandler *handler.GoalHandler) bool {
	if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/history") {
		return false
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "goals" || parts[4] != "history" {
		return false
	}

	goalID, err := uuid.Parse(parts[3])
	if err != nil {
		http.Error(w, "invalid goal id", http.StatusBadRequest)
		return true
	}

	auth, err := authenticateHTTP(r, jwtManager, apiKeyRepo)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return true
	}

	goal, snapshots, contributions, err := goalHandler.GetProgressHistory(r.Context(), auth.UserID, goalID)
	if err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			http.Error(w, "goal not found", http.StatusNotFound)
			return true
		}
		http.Error(w, "failed to fetch goal history", http.StatusInternalServerError)
		return true
	}

	maxPoints := 0
	if raw := r.URL.Query().Get("max_points"); raw != "" {
		if v, convErr := strconv.Atoi(raw); convErr == nil && v > 0 {
			maxPoints = v
		}
	}

	history := make([]map[string]string, 0, len(snapshots))
	for _, s := range snapshots {
		history = append(history, map[string]string{
			"id":         s.ID.String(),
			"goalId":     s.GoalID.String(),
			"amount":     s.Amount.String(),
			"recordedAt": s.RecordedAt.Format(time.RFC3339),
		})
	}

	if maxPoints > 0 && len(history) > maxPoints {
		history = history[len(history)-maxPoints:]
	}

	contributionItems := make([]map[string]string, 0, len(contributions))
	for _, c := range contributions {
		contributionItems = append(contributionItems, map[string]string{
			"id":           c.ID.String(),
			"goalId":       c.GoalID.String(),
			"amountDelta":  c.AmountDelta.String(),
			"balanceAfter": c.BalanceAfter.String(),
			"source":       c.Source,
			"recordedAt":   c.RecordedAt.Format(time.RFC3339),
		})
	}
	if maxPoints > 0 && len(contributionItems) > maxPoints {
		contributionItems = contributionItems[len(contributionItems)-maxPoints:]
	}

	resp := map[string]any{
		"goal": map[string]string{
			"id":            goal.ID.String(),
			"name":          goal.Name,
			"targetAmount":  goal.TargetAmount.String(),
			"currentAmount": goal.CurrentAmount.String(),
			"currency":      goal.Currency,
			"createdAt":     goal.CreatedAt.Format(time.RFC3339),
		},
		"history":       history,
		"contributions": contributionItems,
	}
	if goal.Deadline != nil {
		resp["goal"].(map[string]string)["deadline"] = goal.Deadline.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	return true
}

func serveAccounting(w http.ResponseWriter, r *http.Request, jwtManager *jwt.Manager, apiKeyRepo *repository.ApiKeyRepository, accountingRepo *repository.AccountingRepository) bool {
	if r.Method != http.MethodGet {
		return false
	}

	if r.URL.Path != "/api/v1/accounting/accounts" && r.URL.Path != "/api/v1/accounting/journal" {
		return false
	}

	auth, err := authenticateHTTP(r, jwtManager, apiKeyRepo)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return true
	}

	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/api/v1/accounting/accounts" {
		accounts, err := accountingRepo.ListAccountsWithBalances(r.Context(), auth.UserID)
		if err != nil {
			http.Error(w, "failed to fetch accounts", http.StatusInternalServerError)
			return true
		}

		items := make([]map[string]any, 0, len(accounts))
		for _, a := range accounts {
			item := map[string]any{
				"id":             a.ID.String(),
				"name":           a.Name,
				"accountType":    string(a.AccountType),
				"currency":       a.Currency,
				"openingBalance": a.OpeningBalance.String(),
				"balance":        a.Balance.String(),
				"assetTypeName":  a.AssetTypeName,
				"isSystem":       a.IsSystem,
				"createdAt":      a.CreatedAt.Format(time.RFC3339),
				"updatedAt":      a.UpdatedAt.Format(time.RFC3339),
			}
			if a.AssetID != nil {
				item["assetId"] = a.AssetID.String()
			}
			if a.CategoryID != nil {
				item["categoryId"] = a.CategoryID.String()
			}
			items = append(items, item)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"accounts": items})
		return true
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, convErr := strconv.Atoi(raw); convErr == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	entries, err := accountingRepo.ListJournalEntries(r.Context(), auth.UserID, limit)
	if err != nil {
		http.Error(w, "failed to fetch journal", http.StatusInternalServerError)
		return true
	}

	entryItems := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		entryItem := map[string]any{
			"id":            e.ID.String(),
			"entryDate":     e.EntryDate.Format(time.RFC3339),
			"description":   e.Description,
			"source":        e.Source,
			"referenceType": e.ReferenceType,
			"baseCurrency":  e.BaseCurrency,
			"createdAt":     e.CreatedAt.Format(time.RFC3339),
		}
		if e.ReferenceID != nil {
			entryItem["referenceId"] = e.ReferenceID.String()
		}

		lineItems := make([]map[string]any, 0, len(e.Lines))
		for _, l := range e.Lines {
			lineItems = append(lineItems, map[string]any{
				"id":          l.ID.String(),
				"accountId":   l.AccountID.String(),
				"accountName": l.AccountName,
				"accountType": string(l.AccountType),
				"debit":       l.Debit.String(),
				"credit":      l.Credit.String(),
				"baseDebit":   l.BaseDebit.String(),
				"baseCredit":  l.BaseCredit.String(),
				"description": l.Description,
			})
		}
		entryItem["lines"] = lineItems
		entryItems = append(entryItems, entryItem)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"entries": entryItems})
	return true
}

func serveTransfers(
	w http.ResponseWriter,
	r *http.Request,
	jwtManager *jwt.Manager,
	apiKeyRepo *repository.ApiKeyRepository,
	transferRepo *repository.TransferRepository,
	assetRepo *repository.AssetRepository,
	accountingRepo *repository.AccountingRepository,
) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/v1/transfers") {
		return false
	}

	auth, err := authenticateHTTP(r, jwtManager, apiKeyRepo)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return true
	}

	writeTransfer := func(statusCode int, t *model.Transfer) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"transfer": map[string]any{
				"id":            t.ID.String(),
				"fromAssetId":   t.FromAssetID.String(),
				"toAssetId":     t.ToAssetID.String(),
				"fromAmount":    t.FromAmount.String(),
				"toAmount":      t.ToAmount.String(),
				"fromCurrency":  t.FromCurrency,
				"toCurrency":    t.ToCurrency,
				"exchangeRate":  t.ExchangeRate.String(),
				"transferDate":  t.TransferDate.Format(time.RFC3339),
				"description":   t.Description,
				"createdAt":     t.CreatedAt.Format(time.RFC3339),
				"updatedAt":     t.UpdatedAt.Format(time.RFC3339),
				"fromAssetName": t.FromAssetName,
				"toAssetName":   t.ToAssetName,
			},
		})
	}

	if r.Method == http.MethodGet && r.URL.Path == "/api/v1/transfers" {
		items, err := transferRepo.List(r.Context(), auth.UserID, 200)
		if err != nil {
			http.Error(w, "failed to fetch transfers", http.StatusInternalServerError)
			return true
		}
		resp := make([]map[string]any, 0, len(items))
		for _, t := range items {
			resp = append(resp, map[string]any{
				"id":            t.ID.String(),
				"fromAssetId":   t.FromAssetID.String(),
				"toAssetId":     t.ToAssetID.String(),
				"fromAmount":    t.FromAmount.String(),
				"toAmount":      t.ToAmount.String(),
				"fromCurrency":  t.FromCurrency,
				"toCurrency":    t.ToCurrency,
				"exchangeRate":  t.ExchangeRate.String(),
				"transferDate":  t.TransferDate.Format(time.RFC3339),
				"description":   t.Description,
				"createdAt":     t.CreatedAt.Format(time.RFC3339),
				"updatedAt":     t.UpdatedAt.Format(time.RFC3339),
				"fromAssetName": t.FromAssetName,
				"toAssetName":   t.ToAssetName,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"transfers": resp})
		return true
	}

	if r.Method == http.MethodPost && r.URL.Path == "/api/v1/transfers" {
		var payload transferPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return true
		}
		t, err := parseTransferPayload(payload, auth.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return true
		}
		fromAsset, err := assetRepo.GetByID(r.Context(), t.FromAssetID, auth.UserID)
		if err != nil {
			http.Error(w, "from asset not found", http.StatusBadRequest)
			return true
		}
		toAsset, err := assetRepo.GetByID(r.Context(), t.ToAssetID, auth.UserID)
		if err != nil {
			http.Error(w, "to asset not found", http.StatusBadRequest)
			return true
		}
		if err := finalizeTransferPayload(t, payload, fromAsset.Currency, toAsset.Currency); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return true
		}
		if err := transferRepo.Create(r.Context(), t); err != nil {
			http.Error(w, "failed to create transfer", http.StatusInternalServerError)
			return true
		}
		fromAcc, err := accountingRepo.EnsureAssetAccount(r.Context(), fromAsset)
		if err != nil {
			http.Error(w, "failed to ensure source account", http.StatusInternalServerError)
			return true
		}
		toAcc, err := accountingRepo.EnsureAssetAccount(r.Context(), toAsset)
		if err != nil {
			http.Error(w, "failed to ensure destination account", http.StatusInternalServerError)
			return true
		}
		if err := accountingRepo.UpsertTransferEntry(r.Context(), auth.UserID, t, fromAcc, toAcc); err != nil {
			http.Error(w, "failed to post transfer journal", http.StatusInternalServerError)
			return true
		}
		t.FromAssetName = fromAsset.Name
		t.ToAssetName = toAsset.Name
		writeTransfer(http.StatusCreated, t)
		return true
	}

	if strings.HasPrefix(r.URL.Path, "/api/v1/transfers/") {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/transfers/")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid transfer id", http.StatusBadRequest)
			return true
		}

		switch r.Method {
		case http.MethodPatch:
			transfer, err := transferRepo.GetByID(r.Context(), id, auth.UserID)
			if err != nil {
				http.Error(w, "transfer not found", http.StatusNotFound)
				return true
			}

			var payload transferPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return true
			}

			next, err := parseTransferPayload(payload, auth.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return true
			}
			next.ID = transfer.ID

			fromAsset, err := assetRepo.GetByID(r.Context(), next.FromAssetID, auth.UserID)
			if err != nil {
				http.Error(w, "from asset not found", http.StatusBadRequest)
				return true
			}
			toAsset, err := assetRepo.GetByID(r.Context(), next.ToAssetID, auth.UserID)
			if err != nil {
				http.Error(w, "to asset not found", http.StatusBadRequest)
				return true
			}
			if err := finalizeTransferPayload(next, payload, fromAsset.Currency, toAsset.Currency); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return true
			}

			if err := transferRepo.Update(r.Context(), next); err != nil {
				http.Error(w, "failed to update transfer", http.StatusInternalServerError)
				return true
			}
			fromAcc, err := accountingRepo.EnsureAssetAccount(r.Context(), fromAsset)
			if err != nil {
				http.Error(w, "failed to ensure source account", http.StatusInternalServerError)
				return true
			}
			toAcc, err := accountingRepo.EnsureAssetAccount(r.Context(), toAsset)
			if err != nil {
				http.Error(w, "failed to ensure destination account", http.StatusInternalServerError)
				return true
			}
			if err := accountingRepo.UpsertTransferEntry(r.Context(), auth.UserID, next, fromAcc, toAcc); err != nil {
				http.Error(w, "failed to post transfer journal", http.StatusInternalServerError)
				return true
			}
			next.FromAssetName = fromAsset.Name
			next.ToAssetName = toAsset.Name
			writeTransfer(http.StatusOK, next)
			return true

		case http.MethodDelete:
			if err := transferRepo.Delete(r.Context(), id, auth.UserID); err != nil {
				http.Error(w, "transfer not found", http.StatusNotFound)
				return true
			}
			if err := accountingRepo.DeleteTransferEntry(r.Context(), auth.UserID, id); err != nil {
				http.Error(w, "failed to delete transfer journal", http.StatusInternalServerError)
				return true
			}
			w.WriteHeader(http.StatusNoContent)
			return true
		}
	}

	return false
}

func serveTransactionAssistant(
	w http.ResponseWriter,
	r *http.Request,
	jwtManager *jwt.Manager,
	apiKeyRepo *repository.ApiKeyRepository,
	assistantService *service.TransactionAssistantService,
) bool {
	if r.URL.Path != "/api/v1/transactions/assistant/parse" {
		return false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return true
	}

	if _, err := authenticateHTTP(r, jwtManager, apiKeyRepo); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return true
	}

	var payload service.AssistantParseRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return true
	}

	parsed, err := assistantService.Parse(r.Context(), payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return true
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(parsed)
	return true
}

func serveTransactionSourceLinks(
	w http.ResponseWriter,
	r *http.Request,
	jwtManager *jwt.Manager,
	apiKeyRepo *repository.ApiKeyRepository,
	transactionRepo *repository.TransactionRepository,
) bool {
	if r.URL.Path != "/api/v1/transactions/source-links" {
		return false
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return true
	}

	auth, err := authenticateHTTP(r, jwtManager, apiKeyRepo)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return true
	}

	links, err := transactionRepo.ListSourceAssetLinks(r.Context(), auth.UserID)
	if err != nil {
		http.Error(w, "failed to fetch transaction source links", http.StatusInternalServerError)
		return true
	}

	items := make([]map[string]string, 0, len(links))
	for _, item := range links {
		items = append(items, map[string]string{
			"transactionId": item.TransactionID.String(),
			"assetId":       item.AssetID.String(),
			"assetName":     item.AssetName,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"links": items})
	return true
}

func parseTransferPayload(payload transferPayload, userID uuid.UUID) (*model.Transfer, error) {
	fromAssetID, err := uuid.Parse(payload.FromAssetID)
	if err != nil {
		return nil, errors.New("invalid fromAssetId")
	}
	toAssetID, err := uuid.Parse(payload.ToAssetID)
	if err != nil {
		return nil, errors.New("invalid toAssetId")
	}
	if fromAssetID == toAssetID {
		return nil, errors.New("from and to assets must be different")
	}

	fromAmount, err := decimal.NewFromString(payload.FromAmount)
	if err != nil || fromAmount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("invalid fromAmount")
	}

	transferDate, err := time.Parse(time.RFC3339, payload.TransferDate)
	if err != nil {
		return nil, errors.New("invalid transferDate")
	}

	return &model.Transfer{
		UserID:       userID,
		FromAssetID:  fromAssetID,
		ToAssetID:    toAssetID,
		FromAmount:   fromAmount.Round(2),
		ToAmount:     decimal.Zero,
		FromCurrency: "",
		ToCurrency:   "",
		ExchangeRate: decimal.Zero,
		TransferDate: transferDate,
		Description:  payload.Description,
	}, nil
}

func finalizeTransferPayload(t *model.Transfer, payload transferPayload, fromAssetCurrency, toAssetCurrency string) error {
	fromCurrency := strings.ToUpper(strings.TrimSpace(fromAssetCurrency))
	toCurrency := strings.ToUpper(strings.TrimSpace(toAssetCurrency))
	if fromCurrency == "" || toCurrency == "" {
		return errors.New("asset currencies are required")
	}

	t.FromCurrency = fromCurrency
	t.ToCurrency = toCurrency

	toAmountRaw := strings.TrimSpace(payload.ToAmount)
	toCurrencyInput := strings.ToUpper(strings.TrimSpace(payload.ToCurrency))

	if fromCurrency == toCurrency {
		if toCurrencyInput != "" && toCurrencyInput != toCurrency {
			return errors.New("toCurrency must match destination asset currency")
		}
		if toAmountRaw == "" {
			t.ToAmount = t.FromAmount
			t.ExchangeRate = decimal.NewFromInt(1)
			return nil
		}
		toAmount, err := decimal.NewFromString(toAmountRaw)
		if err != nil || toAmount.LessThanOrEqual(decimal.Zero) {
			return errors.New("invalid toAmount")
		}
		t.ToAmount = toAmount.Round(2)
		if t.FromAmount.IsZero() {
			return errors.New("invalid fromAmount")
		}
		t.ExchangeRate = t.ToAmount.Div(t.FromAmount)
		return nil
	}

	if toAmountRaw == "" {
		return errors.New("toAmount is required for cross-currency transfer")
	}
	if toCurrencyInput == "" {
		return errors.New("toCurrency is required for cross-currency transfer")
	}
	if toCurrencyInput != toCurrency {
		return errors.New("toCurrency must match destination asset currency")
	}

	toAmount, err := decimal.NewFromString(toAmountRaw)
	if err != nil || toAmount.LessThanOrEqual(decimal.Zero) {
		return errors.New("invalid toAmount")
	}
	t.ToAmount = toAmount.Round(2)
	if t.FromAmount.IsZero() {
		return errors.New("invalid fromAmount")
	}
	t.ExchangeRate = t.ToAmount.Div(t.FromAmount)
	return nil
}

type httpAuthResult struct {
	UserID uuid.UUID
}

func authenticateHTTP(r *http.Request, jwtManager *jwt.Manager, apiKeyRepo *repository.ApiKeyRepository) (*httpAuthResult, error) {
	authz := r.Header.Get("Authorization")
	if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
		return nil, errors.New("unauthorized")
	}

	token := strings.TrimPrefix(authz, "Bearer ")

	if strings.HasPrefix(token, "api_") {
		apiKey, err := apiKeyRepo.GetByKey(r.Context(), token)
		if err != nil {
			return nil, errors.New("unauthorized")
		}
		go func() {
			_ = apiKeyRepo.UpdateLastUsed(context.Background(), apiKey.ID)
		}()
		return &httpAuthResult{UserID: apiKey.UserID}, nil
	}

	claims, err := jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	return &httpAuthResult{UserID: claims.UserID}, nil
}

// withCORS adds CORS headers for development
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h.ServeHTTP(w, r)
	})
}

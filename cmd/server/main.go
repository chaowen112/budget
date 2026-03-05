package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/config"
	"github.com/chaowen/budget/internal/handler"
	"github.com/chaowen/budget/internal/middleware"
	"github.com/chaowen/budget/internal/pkg/currency"
	"github.com/chaowen/budget/internal/pkg/jwt"
	"github.com/chaowen/budget/internal/repository"
	"github.com/chaowen/budget/internal/service"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed swagger-ui/*
var swaggerUI embed.FS

//go:embed openapi/*
var openAPISpec embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func run() error {
	// Load configuration
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := repository.NewDB(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	log.Println("Connected to database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)
	assetRepo := repository.NewAssetRepository(db)
	goalRepo := repository.NewGoalRepository(db)
	currencyRepo := repository.NewCurrencyRepository(db)
	cpfRepo := repository.NewCPFRepository(db)

	// Initialize JWT manager
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	// Initialize services
	userService := service.NewUserService(userRepo, jwtManager)

	// Initialize external clients
	currencyClient := currency.NewClient(cfg.Exchange.APIKey)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	categoryHandler := handler.NewCategoryHandler(categoryRepo)
	transactionHandler := handler.NewTransactionHandler(transactionRepo, categoryRepo)
	budgetHandler := handler.NewBudgetHandler(budgetRepo)
	assetHandler := handler.NewAssetHandler(assetRepo)
	goalHandler := handler.NewGoalHandler(goalRepo, assetRepo, transactionRepo)
	currencyHandler := handler.NewCurrencyHandler(currencyRepo, currencyClient)
	cpfHandler := handler.NewCPFHandler(cpfRepo)
	reportHandler := handler.NewReportHandler(transactionRepo, budgetRepo, assetRepo, goalRepo)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.AuthInterceptor(jwtManager, middleware.PublicMethods()),
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
		log.Printf("gRPC server listening on %s", grpcAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("gRPC server error: %v", err)
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
		Handler: httpMux,
	}

	go func() {
		log.Printf("HTTP server (gRPC-Gateway) listening on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down servers...")

	// Graceful shutdown
	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Servers stopped")
	return nil
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

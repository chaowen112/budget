package handler

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/shopspring/decimal"

	pb "github.com/chaowen/budget/gen/budget/v1"
	"github.com/chaowen/budget/internal/pkg/currency"
	"github.com/chaowen/budget/internal/repository"
)

type CurrencyHandler struct {
	pb.UnimplementedCurrencyServiceServer
	currencyRepo   *repository.CurrencyRepository
	currencyClient *currency.Client
}

func NewCurrencyHandler(currencyRepo *repository.CurrencyRepository, currencyClient *currency.Client) *CurrencyHandler {
	return &CurrencyHandler{
		currencyRepo:   currencyRepo,
		currencyClient: currencyClient,
	}
}

func (h *CurrencyHandler) ListCurrencies(ctx context.Context, req *pb.ListCurrenciesRequest) (*pb.ListCurrenciesResponse, error) {
	currencies, err := h.currencyRepo.ListCurrencies(ctx, req.ActiveOnly)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list currencies")
	}

	pbCurrencies := make([]*pb.Currency, len(currencies))
	for i, c := range currencies {
		pbCurrencies[i] = &pb.Currency{
			Code:     c.Code,
			Name:     c.Name,
			Symbol:   c.Symbol,
			IsActive: c.IsActive,
		}
	}

	return &pb.ListCurrenciesResponse{
		Currencies: pbCurrencies,
	}, nil
}

func (h *CurrencyHandler) GetExchangeRate(ctx context.Context, req *pb.GetExchangeRateRequest) (*pb.GetExchangeRateResponse, error) {
	if req.FromCurrency == "" || req.ToCurrency == "" {
		return nil, status.Error(codes.InvalidArgument, "from_currency and to_currency are required")
	}

	// Try to get from database first
	rate, err := h.currencyRepo.GetExchangeRate(ctx, req.FromCurrency, req.ToCurrency)
	if err != nil {
		if err == repository.ErrExchangeRateNotFound {
			// Try to fetch from external API
			rates, fetchErr := h.currencyClient.GetRates(ctx, req.FromCurrency)
			if fetchErr != nil {
				return nil, status.Error(codes.NotFound, "exchange rate not found")
			}

			rateValue, ok := rates[req.ToCurrency]
			if !ok {
				return nil, status.Error(codes.NotFound, "exchange rate not found")
			}

			// Store for future use
			_ = h.currencyRepo.UpsertExchangeRate(ctx, req.FromCurrency, req.ToCurrency, rateValue)

			return &pb.GetExchangeRateResponse{
				Rate: &pb.ExchangeRate{
					FromCurrency: req.FromCurrency,
					ToCurrency:   req.ToCurrency,
					Rate:         rateValue.String(),
					FetchedAt:    timestamppb.Now(),
				},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get exchange rate")
	}

	return &pb.GetExchangeRateResponse{
		Rate: &pb.ExchangeRate{
			FromCurrency: rate.FromCurrency,
			ToCurrency:   rate.ToCurrency,
			Rate:         rate.Rate.String(),
			FetchedAt:    timestamppb.New(rate.FetchedAt),
		},
	}, nil
}

func (h *CurrencyHandler) ConvertCurrency(ctx context.Context, req *pb.ConvertCurrencyRequest) (*pb.ConvertCurrencyResponse, error) {
	if req.Amount == "" {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}
	if req.FromCurrency == "" || req.ToCurrency == "" {
		return nil, status.Error(codes.InvalidArgument, "from_currency and to_currency are required")
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount")
	}

	// Same currency, no conversion needed
	if req.FromCurrency == req.ToCurrency {
		return &pb.ConvertCurrencyResponse{
			OriginalAmount:  req.Amount,
			ConvertedAmount: req.Amount,
			FromCurrency:    req.FromCurrency,
			ToCurrency:      req.ToCurrency,
			Rate:            "1",
		}, nil
	}

	// Try to use the external API client
	converted, rate, err := h.currencyClient.Convert(ctx, amount, req.FromCurrency, req.ToCurrency)
	if err != nil {
		// Fallback to database rate
		dbRate, dbErr := h.currencyRepo.GetExchangeRate(ctx, req.FromCurrency, req.ToCurrency)
		if dbErr != nil {
			return nil, status.Error(codes.Internal, "failed to convert currency: "+err.Error())
		}
		converted = amount.Mul(dbRate.Rate)
		rate = dbRate.Rate
	}

	return &pb.ConvertCurrencyResponse{
		OriginalAmount:  req.Amount,
		ConvertedAmount: converted.String(),
		FromCurrency:    req.FromCurrency,
		ToCurrency:      req.ToCurrency,
		Rate:            rate.String(),
	}, nil
}

func (h *CurrencyHandler) SyncExchangeRates(ctx context.Context, req *pb.SyncExchangeRatesRequest) (*pb.SyncExchangeRatesResponse, error) {
	baseCurrency := req.BaseCurrency
	if baseCurrency == "" {
		baseCurrency = "SGD"
	}

	rates, err := h.currencyClient.GetRates(ctx, baseCurrency)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch exchange rates: "+err.Error())
	}

	if err := h.currencyRepo.BulkUpsertExchangeRates(ctx, baseCurrency, rates); err != nil {
		return nil, status.Error(codes.Internal, "failed to save exchange rates")
	}

	return &pb.SyncExchangeRatesResponse{
		RatesUpdated: int32(len(rates)),
		SyncedAt:     timestamppb.New(time.Now()),
	}, nil
}

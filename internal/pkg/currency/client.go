package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// Client handles exchange rate API calls
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// ExchangeRateResponse represents the API response
type ExchangeRateResponse struct {
	Result          string             `json:"result"`
	BaseCode        string             `json:"base_code"`
	ConversionRates map[string]float64 `json:"conversion_rates"`
	TimeLastUpdate  int64              `json:"time_last_update_unix"`
}

// NewClient creates a new exchange rate API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://v6.exchangerate-api.com/v6",
	}
}

// GetRates fetches exchange rates for a base currency
func (c *Client) GetRates(ctx context.Context, baseCurrency string) (map[string]decimal.Decimal, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("exchange rate API key not configured")
	}

	url := fmt.Sprintf("%s/%s/latest/%s", c.baseURL, c.apiKey, baseCurrency)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange rate API returned status %d", resp.StatusCode)
	}

	var result ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Result != "success" {
		return nil, fmt.Errorf("exchange rate API returned error: %s", result.Result)
	}

	rates := make(map[string]decimal.Decimal)
	for currency, rate := range result.ConversionRates {
		rates[currency] = decimal.NewFromFloat(rate)
	}

	return rates, nil
}

// Convert converts an amount from one currency to another
func (c *Client) Convert(ctx context.Context, amount decimal.Decimal, from, to string) (decimal.Decimal, decimal.Decimal, error) {
	rates, err := c.GetRates(ctx, from)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	rate, ok := rates[to]
	if !ok {
		return decimal.Zero, decimal.Zero, fmt.Errorf("exchange rate not found for %s to %s", from, to)
	}

	converted := amount.Mul(rate)
	return converted, rate, nil
}

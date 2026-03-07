package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TransactionAssistantService struct {
	provider string
	apiKey   string
	model    string
	baseURL  string
	client   *http.Client
}

type AssistantParseRequest struct {
	Message      string `json:"message"`
	ImageDataURL string `json:"imageDataUrl"`
}

type AssistantSuggestion struct {
	EntryType      string   `json:"entryType"`
	Description    string   `json:"description"`
	TransactionDate string  `json:"transactionDate"`
	Amount         string   `json:"amount"`
	Currency       string   `json:"currency"`
	CategoryType   string   `json:"categoryType"`
	CategoryName   string   `json:"categoryName"`
	SourceAsset    string   `json:"sourceAsset"`
	FromAsset      string   `json:"fromAsset"`
	ToAsset        string   `json:"toAsset"`
	FromAmount     string   `json:"fromAmount"`
	ToAmount       string   `json:"toAmount"`
	FromCurrency   string   `json:"fromCurrency"`
	ToCurrency     string   `json:"toCurrency"`
	Confidence     float64  `json:"confidence"`
	MissingFields  []string `json:"missingFields"`
}

type AssistantParseResponse struct {
	Suggestion AssistantSuggestion `json:"suggestion"`
	RawText    string              `json:"rawText"`
	Provider   string              `json:"provider"`
}

func NewTransactionAssistantService(provider, apiKey, model, baseURL string) *TransactionAssistantService {
	return &TransactionAssistantService{
		provider: provider,
		apiKey:   apiKey,
		model:    model,
		baseURL:  strings.TrimRight(baseURL, "/"),
		client:   &http.Client{Timeout: 45 * time.Second},
	}
}

func (s *TransactionAssistantService) Parse(ctx context.Context, req AssistantParseRequest) (*AssistantParseResponse, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" && req.ImageDataURL == "" {
		return nil, errors.New("message or imageDataUrl is required")
	}

	if s.apiKey != "" && strings.EqualFold(s.provider, "openai") {
		if parsed, err := s.parseWithOpenAI(ctx, req); err == nil {
			parsed.Provider = "openai"
			return parsed, nil
		}
	}

	suggestion := heuristicParse(message)
	if req.ImageDataURL != "" {
		suggestion.MissingFields = appendMissing(suggestion.MissingFields, "image_ai_unavailable")
	}
	return &AssistantParseResponse{
		Suggestion: suggestion,
		RawText:    message,
		Provider:   "heuristic",
	}, nil
}

func (s *TransactionAssistantService) parseWithOpenAI(ctx context.Context, req AssistantParseRequest) (*AssistantParseResponse, error) {
	type contentPart map[string]any

	userParts := []contentPart{}
	if strings.TrimSpace(req.Message) != "" {
		userParts = append(userParts, contentPart{"type": "text", "text": req.Message})
	}
	if strings.TrimSpace(req.ImageDataURL) != "" {
		userParts = append(userParts, contentPart{"type": "image_url", "image_url": map[string]any{"url": req.ImageDataURL}})
	}

	payload := map[string]any{
		"model": s.model,
		"messages": []map[string]any{
			{
				"role": "system",
				"content": "Extract a single personal finance entry from user input. Rules: (1) If purchase was paid/charged by credit card (e.g. 'paid by credit card', 'charged to card'), classify as entryType=transaction with sourceAsset hint as credit card account. (2) If debt repayment between accounts (e.g. 'pay credit card bill from bank', 'repay loan from savings'), classify as entryType=transfer with fromAsset and toAsset hints. Return strict JSON only with keys: entryType(transaction|transfer), description, transactionDate(ISO8601), amount, currency, categoryType(TRANSACTION_TYPE_EXPENSE|TRANSACTION_TYPE_INCOME), categoryName, sourceAsset, fromAsset, toAsset, fromAmount, toAmount, fromCurrency, toCurrency, confidence(0..1), missingFields(array of strings).",
			},
			{
				"role":    "user",
				"content": userParts,
			},
		},
		"temperature": 0.1,
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai provider status %d", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("no ai choices")
	}

	raw := strings.TrimSpace(parsed.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var suggestion AssistantSuggestion
	if err := json.Unmarshal([]byte(raw), &suggestion); err != nil {
		return nil, err
	}
	if suggestion.TransactionDate == "" {
		suggestion.TransactionDate = time.Now().Format(time.RFC3339)
	}
	return &AssistantParseResponse{Suggestion: suggestion, RawText: raw}, nil
}

func heuristicParse(message string) AssistantSuggestion {
	msg := strings.TrimSpace(message)
	lower := strings.ToLower(msg)
	res := AssistantSuggestion{
		EntryType:       "transaction",
		Description:     msg,
		TransactionDate: time.Now().Format(time.RFC3339),
		Currency:        detectCurrency(lower),
		CategoryType:    "TRANSACTION_TYPE_EXPENSE",
		Confidence:      0.45,
		MissingFields:   []string{},
	}

	isDebtRepayment := strings.Contains(lower, "paid off") ||
		strings.Contains(lower, "repay") ||
		strings.Contains(lower, "settle") ||
		strings.Contains(lower, "pay loan") ||
		strings.Contains(lower, "loan repayment") ||
		strings.Contains(lower, "credit card bill") ||
		(strings.Contains(lower, "pay") && strings.Contains(lower, "credit card") && strings.Contains(lower, "from "))

	isCardChargePurchase := strings.Contains(lower, "paid by credit card") ||
		strings.Contains(lower, "charged to") ||
		strings.Contains(lower, "using credit card") ||
		strings.Contains(lower, "via credit card") ||
		strings.Contains(lower, "on credit card")

	if strings.Contains(lower, "transfer") || isDebtRepayment {
		res.EntryType = "transfer"
	}

	if isCardChargePurchase {
		res.EntryType = "transaction"
		res.SourceAsset = "Credit Card"
		res.Confidence += 0.15
	}
	if strings.Contains(lower, "salary") || strings.Contains(lower, "bonus") || strings.Contains(lower, "income") || strings.Contains(lower, "refund") {
		res.CategoryType = "TRANSACTION_TYPE_INCOME"
		res.CategoryName = "Salary"
	}
	if res.CategoryName == "" {
		res.CategoryName = guessCategory(lower, res.CategoryType)
	}

	if amt := extractAmount(lower); amt != "" {
		res.Amount = amt
		res.FromAmount = amt
		if res.EntryType == "transfer" {
			res.ToAmount = amt
		}
		res.Confidence += 0.2
	} else {
		res.MissingFields = appendMissing(res.MissingFields, "amount")
	}

	if from := extractAfter(lower, "from "); from != "" {
		if res.EntryType == "transfer" {
			res.FromAsset = from
		} else {
			res.SourceAsset = from
		}
		res.Confidence += 0.15
	}
	if to := extractAfter(lower, "to "); to != "" && res.EntryType == "transfer" {
		res.ToAsset = to
		res.Confidence += 0.1
	}
	if res.EntryType == "transfer" && res.ToAsset == "" {
		if strings.Contains(lower, "credit card") {
			res.ToAsset = "Credit Card"
		}
		if strings.Contains(lower, "loan") {
			res.ToAsset = "Loan"
		}
	}

	if d := extractDate(lower); d != "" {
		res.TransactionDate = d
		res.Confidence += 0.1
	}

	if res.EntryType == "transaction" {
		if res.Amount == "" {
			res.MissingFields = appendMissing(res.MissingFields, "amount")
		}
		if res.SourceAsset == "" {
			res.MissingFields = appendMissing(res.MissingFields, "sourceAsset")
		}
		if res.CategoryName == "" {
			res.MissingFields = appendMissing(res.MissingFields, "category")
		}
	} else {
		res.FromCurrency = res.Currency
		res.ToCurrency = res.Currency
		if res.FromAsset == "" {
			res.MissingFields = appendMissing(res.MissingFields, "fromAsset")
		}
		if res.ToAsset == "" {
			res.MissingFields = appendMissing(res.MissingFields, "toAsset")
		}
		if res.FromAmount == "" {
			res.MissingFields = appendMissing(res.MissingFields, "fromAmount")
		}
	}

	if res.Confidence > 0.95 {
		res.Confidence = 0.95
	}
	return res
}

func appendMissing(items []string, value string) []string {
	for _, v := range items {
		if v == value {
			return items
		}
	}
	return append(items, value)
}

func detectCurrency(text string) string {
	for _, c := range []string{"sgd", "usd", "eur", "gbp", "jpy", "myr"} {
		if strings.Contains(text, c) {
			return strings.ToUpper(c)
		}
	}
	if strings.Contains(text, "$") {
		return "SGD"
	}
	return "SGD"
}

func guessCategory(text, categoryType string) string {
	if categoryType == "TRANSACTION_TYPE_INCOME" {
		return "Salary"
	}
	switch {
	case strings.Contains(text, "grocery"), strings.Contains(text, "supermarket"):
		return "Groceries"
	case strings.Contains(text, "transport"), strings.Contains(text, "grab"), strings.Contains(text, "taxi"):
		return "Transportation"
	case strings.Contains(text, "rent"), strings.Contains(text, "mortgage"):
		return "Housing"
	case strings.Contains(text, "food"), strings.Contains(text, "dinner"), strings.Contains(text, "lunch"):
		return "Food & Dining"
	default:
		return "Other Expense"
	}
}

func extractAmount(text string) string {
	re := regexp.MustCompile(`([0-9]+(?:\.[0-9]{1,2})?)`)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%.2f", v)
}

func extractAfter(text, marker string) string {
	idx := strings.Index(text, marker)
	if idx < 0 {
		return ""
	}
	part := strings.TrimSpace(text[idx+len(marker):])
	if part == "" {
		return ""
	}
	for _, sep := range []string{" for ", " on ", " yesterday", " today", " tomorrow", ","} {
		if i := strings.Index(part, sep); i > 0 {
			part = strings.TrimSpace(part[:i])
			break
		}
	}
	if part == "" {
		return ""
	}
	return strings.Title(strings.TrimSpace(part))
}

func extractDate(text string) string {
	now := time.Now()
	switch {
	case strings.Contains(text, "yesterday"):
		return now.AddDate(0, 0, -1).Format(time.RFC3339)
	case strings.Contains(text, "today"):
		return now.Format(time.RFC3339)
	case strings.Contains(text, "tomorrow"):
		return now.AddDate(0, 0, 1).Format(time.RFC3339)
	default:
		return ""
	}
}

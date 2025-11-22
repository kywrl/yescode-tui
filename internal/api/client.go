package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"yescode-tui/internal/logger"
	"yescode-tui/internal/version"
)

const (
	defaultBaseURL        = "https://co.yes.vg"
	defaultTimeout        = 5 * time.Second
	defaultRequestTimeout = 10 * time.Second
)

// userAgent returns the User-Agent header value.
func userAgent() string {
	return fmt.Sprintf("yescode-tui/%s", version.Version)
}

// Client wraps HTTP access to the YesCode API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient allows providing a custom http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL overrides the default API base URL (useful for testing).
func WithBaseURL(base string) Option {
	return func(c *Client) {
		c.baseURL = base
	}
}

// NewClient builds a Client with the provided API key.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("api key is required")
	}

	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Profile aggregates the /auth/profile payload.
type Profile struct {
	Email               string   `json:"email"`
	Username            string   `json:"username"`
	Balance             float64  `json:"balance"`
	SubscriptionBalance float64  `json:"subscription_balance"`
	PayAsYouGoBalance   float64  `json:"pay_as_you_go_balance"`
	BalancePreference   string   `json:"balance_preference"`
	SubscriptionExpiry  string   `json:"subscription_expiry"`
	CurrentWeekSpend    float64  `json:"current_week_spend"`
	CurrentMonthSpend   float64  `json:"current_month_spend"`
	SubscriptionPlan    PlanInfo `json:"subscription_plan"`
}

// PlanInfo details the current subscription plan.
type PlanInfo struct {
	Name              string  `json:"name"`
	Price             float64 `json:"price"`
	IsActive          bool    `json:"is_active"`
	DailyBalance      float64 `json:"daily_balance"`
	WeeklyLimit       float64 `json:"weekly_limit"`
	MonthlySpendLimit float64 `json:"monthly_spend_limit"`
}

// ProvidersResponse represents /user/available-providers.
type ProvidersResponse struct {
	HasPaygBalance  bool             `json:"has_payg_balance"`
	HasSubscription bool             `json:"has_subscription"`
	IsTeamMember    bool             `json:"is_team_member"`
	Providers       []ProviderBucket `json:"providers"`
}

// ProviderBucket tracks a provider grouping with its alternatives.
type ProviderBucket struct {
	Provider              ProviderInfo         `json:"provider"`
	RateMultiplier        float64              `json:"rate_multiplier"`
	IsDefault             bool                 `json:"is_default"`
	Source                string               `json:"source"`
	Alternatives          []AlternativeMapping `json:"alternatives"`
	SelectedAlternativeID int                  `json:"selected_alternative_id"`
}

// ProviderInfo contains metadata about a provider group.
type ProviderInfo struct {
	ID            int      `json:"id"`
	DisplayName   string   `json:"display_name"`
	Type          string   `json:"type"`
	Description   string   `json:"description"`
	AllowedModels []string `json:"allowed_models"`
}

// AlternativeMapping represents a provider alternative mapping.
type AlternativeMapping struct {
	ID             int          `json:"id"`
	ProviderID     int          `json:"provider_id"`
	AlternativeID  int          `json:"alternative_id"`
	Alternative    ProviderInfo `json:"alternative"`
	DisplayName    string       `json:"display_name"`
	Priority       int          `json:"priority"`
	IsSelf         bool         `json:"is_self"`
	RateMultiplier float64      `json:"rate_multiplier"`
}

// AlternativeResponse is returned by provider-alternatives endpoints (deprecated).
type AlternativeResponse struct {
	Data []AlternativeOption `json:"data"`
}

// AlternativeOption describes one selectable alternative (deprecated).
type AlternativeOption struct {
	IsSelf      bool                `json:"is_self"`
	Alternative ProviderAlternative `json:"alternative"`
}

// ProviderAlternative holds display info for an alternative (deprecated).
type ProviderAlternative struct {
	ID             int     `json:"id"`
	DisplayName    string  `json:"display_name"`
	Type           string  `json:"type"`
	RateMultiplier float64 `json:"rate_multiplier"`
	Description    string  `json:"description"`
}

// ProviderSelection wraps the current or updated selection.
type ProviderSelection struct {
	ProviderID            int                 `json:"provider_id"`
	SelectedAlternativeID int                 `json:"selected_alternative_id"`
	SelectedAlternative   ProviderAlternative `json:"selected_alternative"`
}

// BalancePreferenceResponse represents updates to balance usage preference.
type BalancePreferenceResponse struct {
	BalancePreference string `json:"balance_preference"`
	UpdatedAt         string `json:"updated_at"`
}

// selectionEnvelope mirrors the API shape { "data": { ... } }.
type selectionEnvelope struct {
	Data ProviderSelection `json:"data"`
}

// APIError represents an HTTP error with optional server message.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("yescode api error: status=%d message=%s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("yescode api error: status=%d body=%s", e.StatusCode, e.Body)
}

type errorPayload struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetProfile fetches /api/v1/auth/profile.
func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var profile Profile
	if err := c.get(ctx, "/api/v1/auth/profile", &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetAvailableProviders fetches /api/v1/user/available-providers.
func (c *Client) GetAvailableProviders(ctx context.Context) (*ProvidersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var resp ProvidersResponse
	if err := c.get(ctx, "/api/v1/user/available-providers", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetProviderAlternatives fetches /api/v1/user/provider-alternatives/{providerID}.
func (c *Client) GetProviderAlternatives(ctx context.Context, providerID int) ([]AlternativeOption, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	path := fmt.Sprintf("/api/v1/user/provider-alternatives/%d", providerID)
	var resp AlternativeResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetProviderSelection fetches /api/v1/user/provider-alternatives/{providerID}/selection.
func (c *Client) GetProviderSelection(ctx context.Context, providerID int) (*ProviderSelection, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	path := fmt.Sprintf("/api/v1/user/provider-alternatives/%d/selection", providerID)
	var env selectionEnvelope
	if err := c.get(ctx, path, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// SwitchProvider updates the selection for the provider group.
func (c *Client) SwitchProvider(ctx context.Context, providerID int, alternativeID int) (*ProviderSelection, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	path := fmt.Sprintf("/api/v1/user/provider-alternatives/%d/selection", providerID)
	payload := map[string]int{"selected_alternative_id": alternativeID}
	var env selectionEnvelope
	if err := c.put(ctx, path, payload, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateBalancePreference sets the user's balance preference.
func (c *Client) UpdateBalancePreference(ctx context.Context, preference string) (*BalancePreferenceResponse, error) {
	if preference == "" {
		return nil, errors.New("preference is required")
	}

	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	payload := map[string]string{"balance_preference": preference}
	var resp BalancePreferenceResponse
	if err := c.put(ctx, "/api/v1/user/balance-preference", payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return err
		}

		// 记录GET请求
		logger.Debug("→ GET %s", req.URL.String())

		err = c.do(req, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < 1 {
			logger.Debug("Request failed, retrying... (attempt %d/2)", attempt+1)
		}
	}
	return lastErr
}

func (c *Client) put(ctx context.Context, path string, body any, out any) error {
	var buf *bytes.Buffer
	var bodyJSON []byte
	if body != nil {
		buf = &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return fmt.Errorf("encode body: %w", err)
		}
		bodyJSON = buf.Bytes()
	}

	// 重新创建 buffer（因为上面的 Encode 已经消费了）
	var bodyReader io.Reader
	if bodyJSON != nil {
		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, err := c.newRequest(ctx, http.MethodPut, path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// 记录PUT请求
	logger.Debug("→ PUT %s", req.URL.String())
	if bodyJSON != nil {
		logger.Debug("  Request Body: %s", string(bodyJSON))
	}

	return c.do(req, out)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent())
	return req, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("HTTP request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body: %v", err)
		return err
	}

	// 记录响应信息
	logger.Debug("← Status: %d", resp.StatusCode)
	if len(bodyBytes) > 0 {
		// 格式化 JSON 输出（如果是有效的 JSON）
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err == nil {
			logger.Debug("  Response Body:\n%s", prettyJSON.String())
		} else {
			logger.Debug("  Response Body: %s", string(bodyBytes))
		}
	}

	if resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
		var payload errorPayload
		if err := json.Unmarshal(bodyBytes, &payload); err == nil {
			if payload.Message != "" {
				apiErr.Message = payload.Message
			} else if payload.Error != "" {
				apiErr.Message = payload.Error
			}
		}
		logger.Error("API Error: %v", apiErr)
		return apiErr
	}

	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			logger.Error("Failed to decode response: %v", err)
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

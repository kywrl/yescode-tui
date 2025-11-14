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
)

const (
	defaultBaseURL        = "https://co.yes.vg"
	defaultTimeout        = 5 * time.Second
	defaultUserAgent      = "yescode-tui/0.1"
	defaultRequestTimeout = 10 * time.Second
)

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
	Providers       []ProviderBucket `json:"providers"`
}

// ProviderBucket tracks a provider grouping.
type ProviderBucket struct {
	Provider       ProviderInfo `json:"provider"`
	RateMultiplier float64      `json:"rate_multiplier"`
	IsDefault      bool         `json:"is_default"`
	Source         string       `json:"source"`
}

// ProviderInfo contains metadata about a provider group.
type ProviderInfo struct {
	ID          int    `json:"id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// AlternativeResponse is returned by provider-alternatives endpoints.
type AlternativeResponse struct {
	Data []AlternativeOption `json:"data"`
}

// AlternativeOption describes one selectable alternative.
type AlternativeOption struct {
	IsSelf      bool                `json:"is_self"`
	Alternative ProviderAlternative `json:"alternative"`
}

// ProviderAlternative holds display info for an alternative.
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
		err = c.do(req, out)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func (c *Client) put(ctx context.Context, path string, body any, out any) error {
	var buf *bytes.Buffer
	if body != nil {
		buf = &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return fmt.Errorf("encode body: %w", err)
		}
	}

	req, err := c.newRequest(ctx, http.MethodPut, path, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
	req.Header.Set("User-Agent", defaultUserAgent)
	return req, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
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
		return apiErr
	}

	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

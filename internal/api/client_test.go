package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, err := NewClient("test-key")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
		if client.apiKey != "test-key" {
			t.Errorf("expected apiKey 'test-key', got %q", client.apiKey)
		}
		if client.baseURL != defaultBaseURL {
			t.Errorf("expected baseURL %q, got %q", defaultBaseURL, client.baseURL)
		}
	})

	t.Run("empty API key", func(t *testing.T) {
		_, err := NewClient("")
		if err == nil {
			t.Fatal("expected error for empty API key")
		}
	})

	t.Run("with custom base URL", func(t *testing.T) {
		customURL := "https://custom.api.url"
		client, err := NewClient("test-key", WithBaseURL(customURL))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if client.baseURL != customURL {
			t.Errorf("expected baseURL %q, got %q", customURL, client.baseURL)
		}
	})

	t.Run("with custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 1 * time.Second}
		client, err := NewClient("test-key", WithHTTPClient(customClient))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if client.httpClient != customClient {
			t.Error("expected custom HTTP client")
		}
	})
}

func TestUserAgent(t *testing.T) {
	ua := userAgent()
	if ua == "" {
		t.Fatal("userAgent() returned empty string")
	}
	// Should contain version
	if len(ua) < len("yescode-tui/") {
		t.Errorf("userAgent() = %q, seems too short", ua)
	}
}

func TestGetProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check authentication header
			if r.Header.Get("X-API-Key") != "test-key" {
				t.Errorf("expected X-API-Key header 'test-key', got %q", r.Header.Get("X-API-Key"))
			}

			// Check User-Agent
			ua := r.Header.Get("User-Agent")
			if ua == "" {
				t.Error("User-Agent header is empty")
			}

			// Return mock profile
			profile := Profile{
				Username:            "testuser",
				Email:               "test@example.com",
				SubscriptionBalance: 10.0,
				PayAsYouGoBalance:   5.0,
				Balance:             15.0,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(profile)
		}))
		defer server.Close()

		client, _ := NewClient("test-key", WithBaseURL(server.URL))
		profile, err := client.GetProfile(context.Background())
		if err != nil {
			t.Fatalf("GetProfile() error = %v", err)
		}
		if profile.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %q", profile.Username)
		}
		if profile.Balance != 15.0 {
			t.Errorf("expected Balance 15.0, got %f", profile.Balance)
		}
	})

	t.Run("401 unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid API key",
			})
		}))
		defer server.Close()

		client, _ := NewClient("bad-key", WithBaseURL(server.URL))
		_, err := client.GetProfile(context.Background())
		if err == nil {
			t.Fatal("expected error for 401 response")
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected APIError, got %T", err)
		}
		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", apiErr.StatusCode)
		}
	})

	t.Run("network error", func(t *testing.T) {
		client, _ := NewClient("test-key", WithBaseURL("http://invalid-host-that-does-not-exist.local"))
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := client.GetProfile(ctx)
		if err == nil {
			t.Fatal("expected error for network failure")
		}
	})
}

func TestGetAvailableProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user/available-providers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := ProvidersResponse{
			HasPaygBalance:  true,
			HasSubscription: true,
			Providers: []ProviderBucket{
				{
					Provider: ProviderInfo{
						ID:          1,
						DisplayName: "GPT-4",
					},
					RateMultiplier: 1.0,
				},
				{
					Provider: ProviderInfo{
						ID:          2,
						DisplayName: "Claude",
					},
					RateMultiplier: 1.2,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := NewClient("test-key", WithBaseURL(server.URL))
	providers, err := client.GetAvailableProviders(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableProviders() error = %v", err)
	}
	if len(providers.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers.Providers))
	}
	if providers.Providers[0].Provider.DisplayName != "GPT-4" {
		t.Errorf("expected first provider name 'GPT-4', got %q", providers.Providers[0].Provider.DisplayName)
	}
}

func TestUpdateBalancePreference(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT method, got %s", r.Method)
			}
			if r.URL.Path != "/api/v1/user/balance-preference" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			var req struct {
				BalancePreference string `json:"balance_preference"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			if req.BalancePreference != "subscription_first" {
				t.Errorf("expected preference 'subscription_first', got %q", req.BalancePreference)
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		}))
		defer server.Close()

		client, _ := NewClient("test-key", WithBaseURL(server.URL))
		resp, err := client.UpdateBalancePreference(context.Background(), "subscription_first")
		if err != nil {
			t.Fatalf("UpdateBalancePreference() error = %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "Not found",
	}

	expected := "yescode api error: status=404 message=Not found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

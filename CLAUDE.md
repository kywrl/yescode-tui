# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**yescode-tui** is a Terminal User Interface (TUI) application for managing YesCode API services. Built with Go and Charm's Bubble Tea framework, it provides an interactive interface for viewing profiles, switching API providers, and managing balance preferences.

**Tech Stack:**
- Language: Go 1.24.2
- TUI Framework: Bubble Tea (Elm-inspired)
- Styling: Lipgloss
- Module: yescode-tui

## Build and Run Commands

```bash
# Build the binary
go build -o yc ./cmd/yc

# Run directly
go run ./cmd/yc --api-key YOUR_KEY

# Run with custom API endpoint
go run ./cmd/yc --api-key YOUR_KEY --base-url https://custom.api.url

# Install to GOPATH/bin (installs as 'yc' command)
go install ./cmd/yc

# Format code
go fmt ./...

# Vet code
go vet ./...

# Update dependencies
go mod tidy

# Run tests (when added)
go test ./...
go test -cover ./...
go test -race ./...
```

## Architecture

The codebase follows a clean, layered architecture with three main components:

### 1. Entry Point (`cmd/yc/main.go`)
- Parses CLI flags (`--api-key`, `--base-url`)
- Reads API key from CLI flag or `YESCODE_API_KEY` environment variable
- Initializes API client with optional configuration
- Launches Bubble Tea program in alternate screen mode

### 2. API Client Layer (`internal/api/client.go`)
**Purpose:** RESTful HTTP client for YesCode API

**Key Configuration:**
- Base URL: `https://co.yes.vg` (configurable via `WithBaseURL` option)
- Authentication: `X-API-Key` header
- Default timeout: 5 seconds
- Retry logic: GET requests retry once on failure

**API Methods:**
- `GetProfile(ctx)` - User profile and balance info
- `GetAvailableProviders(ctx)` - List available API providers
- `GetProviderAlternatives(ctx, providerID)` - Get alternative providers for a group
- `GetProviderSelection(ctx, providerID)` - Get current provider selection
- `SwitchProvider(ctx, providerID, alternativeID)` - Switch active provider
- `UpdateBalancePreference(ctx, preference)` - Set balance usage preference

**Error Handling:**
- Custom `APIError` type with status code and message
- Attempts to parse JSON error payload from server
- Returns structured errors for proper UI error display

### 3. TUI Layer (`internal/tui/model.go`)
**Purpose:** Bubble Tea Model implementing the interactive UI

**Architecture Pattern:**
- Implements `tea.Model` interface (`Init`, `Update`, `View`)
- Message-driven state updates (Elm Architecture)
- Command-based async operations

**Three-Tab Interface:**

1. **Profile Tab (Tab 1)**
   - User account info (username, email)
   - Balance details (subscription, pay-as-you-go, total)
   - Subscription plan information
   - Weekly/monthly spend with percentage indicators
   - Scrollable viewport for long content

2. **Providers Tab (Tab 2)**
   - **Two-panel design:**
     - Left panel: List of provider groups
     - Right panel: Alternatives for selected provider
   - Per-provider state caching in `providerData map[int]*providerState`
   - Visual indicators for current selection (✓ checkmark)
   - Rate multiplier display (e.g., "×1.0", "×1.2")
   - Loading spinners for async operations

3. **Balance Preference Tab (Tab 3)**
   - Toggle between `subscription_first` and `payg_only`
   - Real-time API updates
   - Explanatory descriptions for each option

**State Management:**
- `providerData` map caches alternatives/selection per provider ID
- Loading states tracked independently per provider
- Focus state for two-panel navigation (Providers tab)
- Status messages with auto-clear timers (2-3 seconds)

**Keyboard Navigation:**
```
Tab / 1-3      Switch tabs
↑↓ / k/j       Navigate lists
←→ / h/l       Switch focus (providers panel)
Enter          Select/confirm action
r              Refresh current view
Esc / Ctrl+C   Quit application
```

## Key Design Patterns

### Provider Groups vs Alternatives
- A **provider group** (identified by `provider_id`) represents a category like "GPT-4 Turbo"
- Each group contains multiple **alternatives** (different API sources: official, Cloudflare, Azure, etc.)
- Users can switch between alternatives within a group
- Each alternative has a `rate_multiplier` (pricing relative to baseline)

### Async Operations via Commands
The TUI uses Bubble Tea's command pattern for async API calls:

```go
// Example pattern in Update()
case tea.KeyMsg:
    if key.String() == "enter" {
        return m, fetchDataCmd()  // Returns tea.Cmd
    }

// Separate function returns tea.Cmd
func fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        // API call happens here
        data, err := client.GetSomething(context.Background())
        return dataMsg{data: data, err: err}
    }
}
```

### State Caching
- Provider alternatives are fetched once and cached in `providerData`
- Prevents redundant API calls when switching between providers
- Cache invalidated on refresh (r key)

## Material Design Styling

The UI uses Material Design principles with Lipgloss:

- **Primary Color:** Blue (#2196F3)
- **Accent Color:** Light Blue (#64B5F6)
- **Surface Colors:** White on primary, gray borders
- **Typography:** Bold headers, regular body text
- **Spacing:** Consistent padding (1-2 units)
- **Borders:** Rounded corners with subtle shadows

## Common Development Tasks

### Adding a New API Endpoint

1. **Add method to `internal/api/client.go`:**
   ```go
   func (c *Client) GetNewData(ctx context.Context) (*NewDataType, error) {
       var resp NewDataType
       if err := c.get(ctx, "/api/v1/new/endpoint", &resp); err != nil {
           return nil, err
       }
       return &resp, nil
   }
   ```

2. **Define response type:**
   ```go
   type NewDataType struct {
       Field1 string `json:"field1"`
       Field2 int    `json:"field2"`
   }
   ```

3. **Use in TUI via command:**
   ```go
   func fetchNewDataCmd(client *api.Client) tea.Cmd {
       return func() tea.Msg {
           data, err := client.GetNewData(context.Background())
           return newDataMsg{data: data, err: err}
       }
   }
   ```

### Adding a New Tab

1. **Update tab constants in `model.go`:**
   ```go
   const (
       tabProfile = iota
       tabProviders
       tabBalance
       tabNewTab  // Add new tab
   )
   ```

2. **Add rendering logic in `View()` method:**
   ```go
   switch m.currentTab {
   case tabNewTab:
       return m.renderNewTab()
   }
   ```

3. **Handle tab-specific key bindings in `Update()`**

4. **Implement `renderNewTab()` method with Lipgloss styling**

## Testing Strategy

**Current Status:** No tests exist yet

**Recommended Test Coverage:**

1. **API Client Tests (`internal/api/client_test.go`):**
   - Mock HTTP server for each endpoint
   - Test error handling (401, 404, 500, network errors)
   - Test retry logic for GET requests
   - Test custom options (WithBaseURL, WithHTTPClient)

2. **TUI Model Tests (`internal/tui/model_test.go`):**
   - Test state transitions (tab switching, navigation)
   - Test message handling (API success/error responses)
   - Test keyboard input handling
   - Mock API client for controlled testing

3. **Integration Tests:**
   - End-to-end flows with test API server
   - Profile loading → Provider switching → Preference update

**Test Execution:**
```bash
go test ./...                    # Run all tests
go test -v ./internal/api        # Verbose output for specific package
go test -cover ./...             # With coverage
go test -race ./...              # Race detection
go test -short ./...             # Skip long-running tests
```

## Configuration

### Environment Variables
- `YESCODE_API_KEY` - API authentication key (required if not provided via flag)

### CLI Flags
- `--api-key` - YesCode API Key (overrides environment variable)
- `--base-url` - Custom API Base URL (defaults to https://co.yes.vg)

### API Client Configuration
The client supports functional options pattern:

```go
client, err := api.NewClient(
    apiKey,
    api.WithBaseURL("https://custom.api.url"),
    api.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
)
```

## Code Style Guidelines

### Go Conventions
- Follow standard Go formatting (enforced by `go fmt`)
- Use descriptive variable names (avoid single-letter except in short scopes)
- Export only necessary types/functions
- Document exported types and functions with comments

### Bubble Tea Patterns
- **Commands** for async operations (API calls)
- **Messages** for state updates (define custom message types)
- **Model** holds all UI state (avoid global state)
- **View** is pure function (no side effects)

### Lipgloss Styling
- Define reusable styles as package-level variables
- Use `lipgloss.NewStyle()` for component styles
- Consistent spacing: `.Padding(1, 2)` or `.Margin(1)`
- Color constants for Material Design palette

## Chinese Language Support

The application uses Chinese (简体中文) for:
- Error messages
- UI labels and descriptions
- User-facing text

When adding new UI elements:
- Use Chinese for user-facing strings
- Keep code comments in English for international collaboration
- Consider i18n if multi-language support is needed in the future

## Important Implementation Details

### Retry Logic
GET requests retry once on failure (network errors or non-2xx status). PUT requests do not retry to avoid duplicate operations.

### Context Usage
All API methods accept `context.Context` for:
- Request cancellation
- Timeout control
- Tracing (future enhancement)

### Error Display in UI
API errors are shown as status messages with:
- Red error styling
- HTTP status code when available
- User-friendly message extraction from JSON error payload
- Auto-clear after 3 seconds (or until next action)

### State Initialization
On tab load:
- Profile tab: Fetches profile immediately
- Providers tab: Fetches provider list, alternatives loaded on selection
- Balance tab: Uses cached profile data (no separate API call)

## Git Workflow

**Main Branch:** `main` (active development)
**Remote:** `git@github.com:kywrl/yescode-tui.git`

Recent commits focus on UI polish (Material Design, spacing, title adjustments).

## Performance Considerations

- **Binary Size:** ~10MB (includes debug symbols)
- **HTTP Timeout:** 5 seconds default (prevents hanging on network issues)
- **Memory:** Minimal - caches provider state in-memory (bounded by provider count)
- **Concurrency:** API calls run in Bubble Tea goroutines (non-blocking UI)

## Potential Enhancements

- Add comprehensive unit tests
- Implement logging framework (structured logs)
- Support custom themes/color schemes
- Add version flag and semantic versioning
- Implement client-side rate limiting
- Export usage reports (CSV/JSON)
- Add interactive API key setup wizard

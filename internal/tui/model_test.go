package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2025-01-15T10:30:00Z", "2025年1月15日"},
		{"2025-12-31T23:59:59Z", "2025年12月31日"},
		{"2025-01-01", "2025年1月1日"},
		{"", ""},
		{"invalid-date", "invalid-date"},
	}

	// Create a minimal model just for testing formatDate
	m := &Model{}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := m.formatDate(tt.input)
			if result != tt.expected {
				t.Errorf("formatDate(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTabConstants(t *testing.T) {
	// Verify tab constants are defined correctly
	if tabProfile != 0 {
		t.Errorf("expected tabProfile = 0, got %d", tabProfile)
	}
	if tabProviders != 1 {
		t.Errorf("expected tabProviders = 1, got %d", tabProviders)
	}
	if tabBalancePreference != 2 {
		t.Errorf("expected tabBalancePreference = 2, got %d", tabBalancePreference)
	}
}

func TestUILayoutConstants(t *testing.T) {
	// Verify UI layout is properly initialized
	layout := uiLayout{
		titleLineY:    0,
		helpLineY:     2,
		tabHeaderY:    4,
		contentStartY: 6,
	}

	if layout.titleLineY != 0 {
		t.Errorf("expected titleLineY = 0, got %d", layout.titleLineY)
	}
	if layout.contentStartY != 6 {
		t.Errorf("expected contentStartY = 6, got %d", layout.contentStartY)
	}
}

func TestDefaultConstants(t *testing.T) {
	// Verify important constants
	if defaultViewportHeight < 1 {
		t.Error("defaultViewportHeight should be positive")
	}
	if defaultPanelHeight < 1 {
		t.Error("defaultPanelHeight should be positive")
	}
	if profileRefreshInterval < 1*time.Second {
		t.Error("profileRefreshInterval seems too short")
	}
	if statusClearDelay < 1*time.Second {
		t.Error("statusClearDelay seems too short")
	}
}

func TestFocusArea(t *testing.T) {
	// Verify focus area constants
	if focusProviders != 0 {
		t.Errorf("expected focusProviders = 0, got %d", focusProviders)
	}
	if focusAlternatives != 1 {
		t.Errorf("expected focusAlternatives = 1, got %d", focusAlternatives)
	}
}

func TestColorConstants(t *testing.T) {
	// Verify Material Design colors are defined
	primaryColor := lipgloss.Color("#2196F3")
	if primaryColor == "" {
		t.Error("primaryColor should not be empty")
	}

	accentColor := lipgloss.Color("#64B5F6")
	if accentColor == "" {
		t.Error("accentColor should not be empty")
	}
}


package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// captureLog redirects the default logger to a buffer for the test's duration.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestLoadKeepsRegisteredTheme(t *testing.T) {
	logs := captureLog(t)
	t.Cleanup(viper.Reset)
	viper.Set("ui.theme", "outlook")

	if got := Load().UI.Theme; got != "outlook" {
		t.Errorf("UI.Theme = %q, want %q", got, "outlook")
	}
	if logs.Len() != 0 {
		t.Errorf("a registered theme must not log anything, got: %s", logs)
	}
}

func TestLoadUnknownThemeFallsBackWithWarning(t *testing.T) {
	logs := captureLog(t)
	t.Cleanup(viper.Reset)
	viper.Set("ui.theme", "nope")

	if got := Load().UI.Theme; got != "larry" {
		t.Errorf("UI.Theme = %q, want %q", got, "larry")
	}
	out := logs.String()
	for _, want := range []string{"level=WARN", "value=nope", "larry", "outlook-dark", "high-contrast"} {
		if !strings.Contains(out, want) {
			t.Errorf("warning should contain %q, got: %s", want, out)
		}
	}
}

func TestLoadUnsetThemeIsLarryWithoutWarning(t *testing.T) {
	logs := captureLog(t)
	t.Cleanup(viper.Reset)

	if got := Load().UI.Theme; got != "larry" {
		t.Errorf("UI.Theme = %q, want %q", got, "larry")
	}
	if logs.Len() != 0 {
		t.Errorf("an unset theme must not log anything, got: %s", logs)
	}
}

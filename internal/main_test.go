package internal

import (
	"log/slog"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Configure logger for tests
	slog.SetDefault(slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug, // tests usually want verbose logs
		}),
	))

	// Run tests
	code := m.Run()

	// Optional: cleanup here

	os.Exit(code)
}

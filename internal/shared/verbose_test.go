package shared

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// withDefaultLogger swaps slog.Default() for the duration of the test.
func withDefaultLogger(t *testing.T, l *slog.Logger) {
	t.Helper()

	prev := slog.Default()

	t.Cleanup(func() { slog.SetDefault(prev) })
	slog.SetDefault(l)
}

func TestVerbose_SilentByDefault(t *testing.T) {
	var buf bytes.Buffer

	withDefaultLogger(t, slog.New(slog.NewTextHandler(&buf, nil)))

	Verbose("hello %s", "world")

	assert.Empty(t, buf.String(), "Verbose must not emit at the default (info) level")
}

func TestVerbose_EmitsAtDebugLevel(t *testing.T) {
	var buf bytes.Buffer

	withDefaultLogger(t, slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	Verbose("answer=%d", 42)

	assert.Contains(t, buf.String(), "answer=42",
		"Verbose must format and emit when the default logger allows debug")
}

func TestVerbose_DoesNotFormatWhenSilent(t *testing.T) {
	// If Verbose formatted args before checking the level, calling
	// String() on this sentinel would record a call. The contract is
	// that Verbose returns early without invoking the formatter.
	withDefaultLogger(t, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	var s sentinel

	Verbose("%s", &s)

	assert.False(t, s.formatted, "Verbose must not format args when silent")
}

type sentinel struct{ formatted bool }

func (s *sentinel) String() string {
	s.formatted = true
	return "x"
}

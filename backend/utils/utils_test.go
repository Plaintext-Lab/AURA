package utils

import (
	"aura/logging"
	"context"
	"testing"
	"time"
)

func testContext() context.Context {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	return logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))
}

func num(n int) *int { return &n }

func TestConvertDateStringToTime(t *testing.T) {
	want := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	tests := map[string]string{
		"unix seconds":      "1741064767",
		"unix milliseconds": "1741064767000",
		"RFC3339":           "2025-03-04T05:06:07Z",
		"RFC3339 nano":      "2025-03-04T05:06:07.000000000Z",
		"SQL":               "2025-03-04 05:06:07",
		"compact":           "20250304050607",
		"time string":       "2025-03-04 05:06:07 +0000 UTC",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ConvertDateStringToTime(input); !got.Equal(want) {
				t.Errorf("ConvertDateStringToTime(%q) = %v, want %v", input, got, want)
			}
		})
	}

	if got := ConvertDateStringToTime("2025-03-04"); !got.Equal(time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("date only = %v", got)
	}
	// Dates that can't be read are now
	for _, input := range []string{"", "0", "undefined", "not a date"} {
		if got := ConvertDateStringToTime(input); time.Since(got) > time.Minute {
			t.Errorf("ConvertDateStringToTime(%q) = %v, want now", input, got)
		}
	}
}

func TestIsNewerDate(t *testing.T) {
	if !IsNewerDate("2025-03-05", "2025-03-04") {
		t.Error("the 5th should be newer than the 4th")
	}
	if IsNewerDate("2025-03-04", "2025-03-05") || IsNewerDate("2025-03-04", "2025-03-04") {
		t.Error("an older or equal date should not be newer")
	}
}

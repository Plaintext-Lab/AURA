package utils

import (
	"aura/logging"
	"aura/models"
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

func TestFormatting(t *testing.T) {
	for n, want := range map[int]string{0: "00", 7: "07", 10: "10", 123: "123"} {
		if got := FormatIntAsTwoDigitString(n); got != want {
			t.Errorf("FormatIntAsTwoDigitString(%d) = %q, want %q", n, got, want)
		}
	}
	if ParseFileSize("2048") != 2048 || ParseFileSize("big") != 0 {
		t.Error("ParseFileSize should read a number and return 0 for anything else")
	}
}

func TestGetFileDownloadName(t *testing.T) {
	tests := []struct {
		image models.ImageFile
		want  string
	}{
		{models.ImageFile{Type: "poster"}, "Poster"},
		{models.ImageFile{Type: "backdrop"}, "Backdrop"},
		{models.ImageFile{Type: "season_poster", SeasonNumber: num(1)}, "Season 01 Poster"},
		{models.ImageFile{Type: "season_poster", SeasonNumber: num(0)}, "Special Season Poster"},
		{models.ImageFile{Type: "season_poster"}, "Season Poster"},
		{models.ImageFile{Type: "titlecard", SeasonNumber: num(2), EpisodeNumber: num(3)}, "S02E03 Titlecard"},
		{models.ImageFile{Type: "titlecard", SeasonNumber: num(2)}, "S02 Titlecard"},
		{models.ImageFile{Type: "titlecard", EpisodeNumber: num(3)}, "E03 Titlecard"},
		{models.ImageFile{Type: "titlecard", Title: "Pilot"}, "Pilot Titlecard"},
		{models.ImageFile{Type: "titlecard"}, "Titlecard"},
		{models.ImageFile{Type: "logo"}, "Movie - Image"},
	}
	for _, tt := range tests {
		if got := GetFileDownloadName("Movie", tt.image); got != tt.want {
			t.Errorf("GetFileDownloadName(%+v) = %q, want %q", tt.image, got, tt.want)
		}
	}
}

func TestMediaItemInfo(t *testing.T) {
	got := MediaItemInfo(models.MediaItem{Title: "Movie (2020)", LibraryTitle: "Movies", TMDB_ID: "1", RatingKey: "10"})
	if want := "'Movie' (2020) | Movies [TMDB: 1 | Key: 10]"; got != want {
		t.Errorf("MediaItemInfo = %q, want %q", got, want)
	}
	if got := MediaItemInfo(models.MediaItem{}); got != "'<no title>' (-1) | <no library> [TMDB: <no tmdb> | Key: <no key>]" {
		t.Errorf("MediaItemInfo of an empty item = %q", got)
	}
	if got := CollectionItemInfo(models.CollectionItem{Title: "Saga", LibraryTitle: "Movies", RatingKey: "5"}); got != "'Saga' | Movies [Key: 5]" {
		t.Errorf("CollectionItemInfo = %q", got)
	}
}

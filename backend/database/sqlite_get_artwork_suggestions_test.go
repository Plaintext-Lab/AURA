package database

import (
	"aura/models"
	"testing"
)

// TestBuildReasonCodes verifies that the correct reason codes are produced for
// various coverage states.
func TestBuildReasonCodes(t *testing.T) {
	tests := []struct {
		name          string
		item          models.ArtworkSuggestion
		wantCodes     []string
		wantTextEmpty bool
	}{
		{
			name: "no set",
			item: models.ArtworkSuggestion{
				Type:        "movie",
				HasSavedSet: false,
				PosterCount: 0,
				BackdropCount: 0,
			},
			wantCodes: []string{"no_set", "no_poster", "no_backdrop"},
		},
		{
			name: "has set, poster and backdrop selected – complete coverage for movie",
			item: models.ArtworkSuggestion{
				Type:          "movie",
				HasSavedSet:   true,
				PosterCount:   1,
				BackdropCount: 1,
			},
			wantCodes:     []string{},
			wantTextEmpty: false,
		},
		{
			name: "show missing season posters",
			item: models.ArtworkSuggestion{
				Type:              "show",
				HasSavedSet:       true,
				PosterCount:       1,
				BackdropCount:     1,
				SeasonPosterTotal: 3,
				SeasonPosterCount: 1,
				TitleCardTotal:    0,
				TitleCardCount:    0,
			},
			wantCodes: []string{"missing_season_posters"},
		},
		{
			name: "show missing title cards",
			item: models.ArtworkSuggestion{
				Type:              "show",
				HasSavedSet:       true,
				PosterCount:       1,
				BackdropCount:     1,
				SeasonPosterTotal: 2,
				SeasonPosterCount: 2,
				TitleCardTotal:    10,
				TitleCardCount:    7,
			},
			wantCodes: []string{"missing_title_cards"},
		},
		{
			name: "show fully covered",
			item: models.ArtworkSuggestion{
				Type:              "show",
				HasSavedSet:       true,
				PosterCount:       1,
				BackdropCount:     1,
				SeasonPosterTotal: 2,
				SeasonPosterCount: 2,
				TitleCardTotal:    10,
				TitleCardCount:    10,
			},
			wantCodes:     []string{},
			wantTextEmpty: false,
		},
		{
			name: "show no seasons or episodes – only poster/backdrop gaps",
			item: models.ArtworkSuggestion{
				Type:          "show",
				HasSavedSet:   true,
				PosterCount:   0,
				BackdropCount: 0,
			},
			wantCodes: []string{"no_poster", "no_backdrop"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotCodes, gotText := buildReasonCodes(tc.item)

			if len(gotCodes) != len(tc.wantCodes) {
				t.Fatalf("reason codes: got %v, want %v", gotCodes, tc.wantCodes)
			}
			for i, c := range gotCodes {
				if c != tc.wantCodes[i] {
					t.Errorf("reason code[%d]: got %q, want %q", i, c, tc.wantCodes[i])
				}
			}

			if len(gotCodes) > 0 && gotText == "" {
				t.Error("reason_text must not be empty when reason codes are present")
			}
			if len(gotCodes) == 0 && gotText == "" {
				t.Error("reason_text must not be empty even when no gap codes exist")
			}
		})
	}
}

// TestBuildGapFilter verifies that gap type strings are normalised correctly.
func TestBuildGapFilter(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "any"},
		{"any", "any"},
		{"no_set", "no_set"},
		{"NO_SET", "no_set"},
		{"no_poster", "no_poster"},
		{"no_backdrop", "no_backdrop"},
		{"missing_season_posters", "missing_season_posters"},
		{"missing_title_cards", "missing_title_cards"},
		{"unknown_value", "any"},
	}
	for _, tc := range cases {
		got := buildGapFilter(tc.input)
		if got != tc.want {
			t.Errorf("buildGapFilter(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestBuildGapPredicate verifies that the SQL predicates are non-empty and
// cover all expected gap filter values.
func TestBuildGapPredicate(t *testing.T) {
	filters := []string{
		"any",
		"no_set",
		"no_poster",
		"no_backdrop",
		"missing_season_posters",
		"missing_title_cards",
	}
	for _, f := range filters {
		p := buildGapPredicate(f)
		if p == "" {
			t.Errorf("buildGapPredicate(%q) returned an empty predicate", f)
		}
	}
}

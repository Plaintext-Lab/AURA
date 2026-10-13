package models

import "testing"

func TestImageIsSelected(t *testing.T) {
	num := func(n int) *int { return &n }
	show := MediaItem{Series: &MediaItemSeries{Seasons: []MediaItemSeason{
		{SeasonNumber: 0, Episodes: []MediaItemEpisode{{EpisodeNumber: 1}}},
		{SeasonNumber: 1, Episodes: []MediaItemEpisode{{EpisodeNumber: 1}, {EpisodeNumber: 2}}},
	}}}
	all := SelectedTypes{Poster: true, Backdrop: true, SeasonPoster: true, SpecialSeasonPoster: true, Titlecard: true}

	tests := []struct {
		name     string
		item     MediaItem
		selected SelectedTypes
		image    ImageFile
		want     bool
	}{
		{"poster selected", MediaItem{}, all, ImageFile{Type: "poster"}, true},
		{"poster not selected", MediaItem{}, SelectedTypes{Backdrop: true}, ImageFile{Type: "poster"}, false},
		{"backdrop selected", MediaItem{}, all, ImageFile{Type: "backdrop"}, true},
		{"season poster", show, all, ImageFile{Type: "season_poster", SeasonNumber: num(1)}, true},
		{"season poster not selected", show, SelectedTypes{SpecialSeasonPoster: true}, ImageFile{Type: "season_poster", SeasonNumber: num(1)}, false},
		{"special season poster", show, SelectedTypes{SpecialSeasonPoster: true}, ImageFile{Type: "season_poster", SeasonNumber: num(0)}, true},
		{"special season poster not selected", show, SelectedTypes{SeasonPoster: true}, ImageFile{Type: "season_poster", SeasonNumber: num(0)}, false},
		{"season the item doesn't have", show, all, ImageFile{Type: "season_poster", SeasonNumber: num(5)}, false},
		{"season poster without a season", show, all, ImageFile{Type: "season_poster"}, false},
		{"season poster on a movie", MediaItem{}, all, ImageFile{Type: "season_poster", SeasonNumber: num(1)}, false},
		{"titlecard", show, all, ImageFile{Type: "titlecard", SeasonNumber: num(1), EpisodeNumber: num(2)}, true},
		{"titlecard not selected", show, SelectedTypes{Poster: true}, ImageFile{Type: "titlecard", SeasonNumber: num(1), EpisodeNumber: num(2)}, false},
		{"episode the item doesn't have", show, all, ImageFile{Type: "titlecard", SeasonNumber: num(1), EpisodeNumber: num(9)}, false},
		{"titlecard without numbers matches any episode", show, all, ImageFile{Type: "titlecard"}, true},
		{"unknown type", show, all, ImageFile{Type: "logo"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImageIsSelected(tt.item, tt.selected, tt.image); got != tt.want {
				t.Errorf("ImageIsSelected = %v, want %v", got, tt.want)
			}
		})
	}
}

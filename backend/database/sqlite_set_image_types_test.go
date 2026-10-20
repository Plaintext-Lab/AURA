package database

import (
	"aura/models"
	"testing"
)

func TestMovieAndCollectionSetsOnlyKeepPostersAndBackdrops(t *testing.T) {
	db := openTestMediaDB(t)
	ctx := testContext()
	all := models.SelectedTypes{Poster: true, Backdrop: true, SeasonPoster: true, SpecialSeasonPoster: true, Titlecard: true}
	set := func(id, setType string) models.DBPosterSetDetail {
		return models.DBPosterSetDetail{PosterSet: models.PosterSet{BaseSetInfo: models.BaseSetInfo{ID: id, Type: setType, Title: id, UserCreated: "creator"}}, SelectedTypes: all}
	}
	item := models.DBSavedItem{
		MediaItem: models.MediaItem{TMDB_ID: "1", LibraryTitle: "Movies", RatingKey: "1", Type: "movie", Title: "Movie", Year: 2020,
			Movie: &models.MediaItemMovie{File: models.MediaItemFile{Path: "/movies/Movie.mkv"}}},
		PosterSets: []models.DBPosterSetDetail{set("movieSet", "movie"), set("collectionSet", "collection")},
	}
	show := models.DBSavedItem{
		MediaItem: models.MediaItem{TMDB_ID: "2", LibraryTitle: "Series", RatingKey: "2", Type: "show", Title: "Show", Year: 2020,
			Series: &models.MediaItemSeries{}},
		PosterSets: []models.DBPosterSetDetail{set("showSet", "show")},
	}

	// Saved sets
	for _, saved := range []models.DBSavedItem{item, show} {
		if Err := db.UpsertSavedItem(ctx, saved); Err.Message != "" {
			t.Fatal(Err.Message)
		}
	}
	rows, err := db.conn.Query(`SELECT ps.set_id, si.poster_selected, si.backdrop_selected, si.season_poster_selected, si.special_season_poster_selected, si.titlecard_selected
FROM SavedItems si JOIN PosterSets ps ON ps.id = si.poster_set_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var setID string
		var poster, backdrop, season, special, titlecard bool
		if err := rows.Scan(&setID, &poster, &backdrop, &season, &special, &titlecard); err != nil {
			t.Fatal(err)
		}
		showSet := setID == "showSet"
		if !poster || !backdrop || season != showSet || special != showSet || titlecard != showSet {
			t.Errorf("%s saved poster=%v backdrop=%v season=%v special=%v titlecard=%v", setID, poster, backdrop, season, special, titlecard)
		}
	}

	// Download Queue jobs
	if _, Err := db.EnqueueDownloadQueueJob(ctx, item); Err.Message != "" {
		t.Fatal(Err.Message)
	}
	jobs, Err := db.GetAllDownloadQueueJobs(ctx)
	if Err.Message != "" || len(jobs) != 1 {
		t.Fatalf("jobs = %d %s", len(jobs), Err.Message)
	}
	for _, set := range jobs[0].Payload.PosterSets {
		if set.SelectedTypes != (models.SelectedTypes{Poster: true, Backdrop: true}) {
			t.Errorf("queued %s set = %+v, want only poster and backdrop", set.Type, set.SelectedTypes)
		}
	}
	// The caller's item isn't changed
	if !item.PosterSets[0].SelectedTypes.Titlecard {
		t.Error("queueing changed the caller's selected types")
	}
}

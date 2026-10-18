package database

import "testing"

func TestGetSavedSetCreators(t *testing.T) {
	db := openTestMediaDB(t)
	// Poster sets 1 and 2 are "user", see seedMediaItem
	seedMediaItem(t, db.conn, "1", "Movies", "", "100", "movie", 1, 2)
	seedMediaItem(t, db.conn, "2", "Movies", "", "101", "movie", 1, 3)
	exec(t, db.conn, `UPDATE PosterSets SET user = 'other' WHERE id = 3;`)
	exec(t, db.conn, `INSERT INTO PosterSets (id, set_id, type, title, user) VALUES (4, 4, 'movie', 'Not saved', 'nobody');`)

	creators, Err := db.GetSavedSetCreators(testContext())
	if Err.Message != "" {
		t.Fatal(Err.Message)
	}
	// A set saved for two items counts once, and sets that aren't saved don't count
	if len(creators) != 2 || creators[0] != (CreatorSetCount{Username: "user", Sets: 2}) || creators[1] != (CreatorSetCount{Username: "other", Sets: 1}) {
		t.Errorf("creators = %+v", creators)
	}
}

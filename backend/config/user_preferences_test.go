package config

import (
	"aura/logging"
	"context"
	"slices"
	"testing"
)

func TestValidateUserPreferencesFavoriteCreators(t *testing.T) {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	ctx = logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))

	prefs := Config_UserPreferences{FavoriteCreators: []string{" willtong93 ", "", "benno", "WillTong93", "Pejamas"}}
	if !ValidateUserPreferences(ctx, &prefs) {
		t.Fatal("the preferences should be valid")
	}
	// Blank and repeated names go, each keeps its best rank
	if want := []string{"willtong93", "benno", "Pejamas"}; !slices.Equal(prefs.FavoriteCreators, want) {
		t.Errorf("favorite creators = %q, want %q", prefs.FavoriteCreators, want)
	}
}

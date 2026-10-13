package models

// IsKnownImageType reports whether aura knows how to apply an image of this type
func IsKnownImageType(imageType string) bool {
	switch imageType {
	case "poster", "backdrop", "season_poster", "titlecard":
		return true
	}
	return false
}

// ImageIsSelected reports whether an image of a saved set should be applied to the item.
// The image type must be selected, and the item must have the season or episode the image is for.
func ImageIsSelected(item MediaItem, selected SelectedTypes, image ImageFile) bool {
	switch image.Type {
	case "poster":
		return selected.Poster
	case "backdrop":
		return selected.Backdrop
	case "season_poster":
		if image.SeasonNumber == nil || !itemHasSeason(item, *image.SeasonNumber) {
			return false
		}
		if *image.SeasonNumber == 0 {
			return selected.SpecialSeasonPoster
		}
		return selected.SeasonPoster
	case "titlecard":
		return itemHasEpisode(item, image.SeasonNumber, image.EpisodeNumber) && selected.Titlecard
	}
	return false
}

func itemHasSeason(item MediaItem, seasonNumber int) bool {
	if item.Series == nil {
		return false
	}
	for _, season := range item.Series.Seasons {
		if season.SeasonNumber == seasonNumber {
			return true
		}
	}
	return false
}

// itemHasEpisode reports whether the item has the episode. A missing season or episode number matches any
func itemHasEpisode(item MediaItem, seasonNumber, episodeNumber *int) bool {
	if item.Series == nil {
		return false
	}
	for _, season := range item.Series.Seasons {
		if seasonNumber != nil && *seasonNumber != season.SeasonNumber {
			continue
		}
		for _, episode := range season.Episodes {
			if episodeNumber != nil && *episodeNumber != episode.EpisodeNumber {
				continue
			}
			return true
		}
	}
	return false
}

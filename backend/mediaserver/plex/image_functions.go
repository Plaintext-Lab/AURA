package plex

import (
	"aura/config"
	"aura/logging"
	"aura/models"
	"aura/utils"
	"aura/utils/httpx"
	"context"
	"fmt"
	"net/url"
	"path"
	"time"
)

func getAllImages(ctx context.Context, msConfig config.Config_MediaServer, item *models.MediaItem, itemRatingKey string, itr string, imageType string) (images []PlexGetAllImagesMetadata, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf(
		"Plex: Getting %s Images for %s", itr, utils.MediaItemInfo(*item)),
		logging.LevelDebug)
	defer logAction.Complete()

	images = []PlexGetAllImagesMetadata{}
	Err = logging.LogErrorInfo{}

	if imageType == "backdrop" {
		imageType = "arts"
	} else {
		imageType = "posters"
	}

	// Construct the URL for the Plex API request
	u, err := url.Parse(msConfig.URL)
	if err != nil {
		logAction.SetError(logging.Error_BaseUrlParsing(err))
		return images, *logAction.Error
	}
	u.Path = path.Join(u.Path, "library", "metadata", itemRatingKey, imageType)
	URL := u.String()

	// Make the HTTP Request to Plex
	resp, respBody, Err := makeRequest(ctx, msConfig, URL, "GET", nil)
	if Err.Message != "" {
		logAction.SetErrorFromInfo(Err)
		return images, *logAction.Error
	}
	defer resp.Body.Close()

	// Decode the Response
	var respData PlexGetAllImagesWrapper
	Err = httpx.DecodeResponseToJSON(ctx, respBody, &respData, fmt.Sprintf("%s Media Item Images Response", msConfig.Type))
	if Err.Message != "" {
		return images, *logAction.Error
	}

	for _, img := range respData.MediaContainer.Metadata {
		if img.Provider != "local" {
			continue
		}
		images = append(images, img)
	}

	logAction.AppendResult("image_rating_keys", func() []string {
		keys := make([]string, len(images))
		for i, img := range images {
			keys[i] = img.RatingKey
		}
		return keys
	}())
	logAction.AppendResult("current_image_count", len(images))

	return images, Err
}

// findNewImage attempts to retrieve a new image for a Plex item by comparing previous and current images.
// It tries up to 3 times, refreshing the Plex item between attempts if needed.
// Returns the new image's ratingKey (URL or path) if found, or a StandardError if not.
func findNewImage(ctx context.Context, msConfig config.Config_MediaServer, item *models.MediaItem, itemRatingKey string, imageType string, previousImages []PlexGetAllImagesMetadata) (newImageRatingKey string, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf(
		"Plex: Getting Newest '%s' Image for %s",
		imageType,
		utils.MediaItemInfo(*item),
	), logging.LevelDebug)
	defer logAction.Complete()

	newImageRatingKey = ""
	Err = logging.LogErrorInfo{}

	var lastErrorMsg string
	var lastErrorDetail map[string]any

	// Retry logic for the entire process (up to 3 attempts)
	for attempt := 1; attempt <= 3; attempt++ {
		attemptAction := logAction.AddSubAction(fmt.Sprintf("Attempt %d to fetch new image", attempt), logging.LevelDebug)

		// Get all current images from Plex
		currentImages, fetchErr := getAllImages(ctx, msConfig, item, itemRatingKey, "New", imageType)

		if fetchErr.Message != "" {
			attemptAction.AppendWarning(fmt.Sprintf("attempt_%d", attempt), map[string]any{"error": fetchErr.Message})
			lastErrorMsg = fetchErr.Message
			lastErrorDetail = fetchErr.Detail
		} else {
			prevKeys := make(map[string]struct{}, len(previousImages))
			for _, p := range previousImages {
				if p.Provider != "local" {
					continue
				}
				prevKeys[p.RatingKey] = struct{}{}
			}

			for _, c := range currentImages {
				if c.Provider != "local" {
					continue
				}
				if _, exists := prevKeys[c.RatingKey]; !exists {
					attemptAction.AppendResult(fmt.Sprintf("attempt_%d", attempt), map[string]any{
						"new_image_rating_key": c.RatingKey,
					})
					return c.RatingKey, logging.LogErrorInfo{}
				}
			}

			attemptAction.AppendWarning(fmt.Sprintf("attempt_%d", attempt), map[string]any{
				"error": "No local images found after refresh.",
			})
			lastErrorMsg = "No local images found after refresh."
			lastErrorDetail = nil
		}

		if attempt < 3 {
			time.Sleep(1 * time.Second)
			refreshErr := RefreshItemMetadata(ctx, msConfig, item, itemRatingKey, false)
			if refreshErr.Message != "" {
				attemptAction.AppendWarning("refresh_error", map[string]any{"error": refreshErr.Message})
				lastErrorMsg = refreshErr.Message
				lastErrorDetail = refreshErr.Detail
			}
		}
	}

	return "", logging.LogErrorInfo{Message: lastErrorMsg, Detail: lastErrorDetail}
}

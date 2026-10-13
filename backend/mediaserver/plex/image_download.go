package plex

import (
	"aura/config"
	"aura/logging"
	"aura/mediaserver/localimage"
	"aura/mediux"
	"aura/models"
	"aura/utils"
	"context"
	"fmt"
)

func (p *Plex) DownloadApplyImageToMediaItem(ctx context.Context, item *models.MediaItem, imageFile models.ImageFile) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf(
		"Plex: Downloading and Applying %s Image for %s", utils.GetFileDownloadName(item.Title, imageFile), utils.MediaItemInfo(*item)),
		logging.LevelDebug)
	defer logAction.Complete()
	Err = logging.LogErrorInfo{}

	// Determine the Item Rating Key from Plex
	itemRatingKey := getItemRatingKeyFromImageFile(*item, imageFile)
	if itemRatingKey == "" {
		logAction.SetError("Failed to determine Rating Key for Media Item", "Ensure the Media Item and Image File data are correct", nil)
		return *logAction.Error
	}
	logAction.AppendResult("image_rating_key", itemRatingKey)

	// If SaveImageLocally is disabled, skip downloading the image
	// Only the main media server saves images locally
	if !config.Current.Images.SaveImagesLocally.Enabled || !p.Config.IsMain() {
		return applyImageToMediaItemViaMediuxURL(ctx, p.Config, item, itemRatingKey, imageFile)
	}

	// Get the Image from MediUX
	// mediux.GetImage will handle checking the temp folder and caching based on config
	formatDate := imageFile.Modified.Format("20060102150405")
	imageData, imageType, Err := mediux.GetImage(ctx, imageFile.ID, formatDate, mediux.ImageQualityOriginal)
	if Err.Message != "" {
		return Err
	}

	// Before we download and save the image locally, we need to get a list of the current posters in Plex
	// This is to handle the case where the image is already set in Plex, and we need to replace it
	// with the new image after saving it locally
	// currentImages, logErr := getAllImages(ctx, p.Config, item, itemRatingKey, "Current", imageFile.Type)
	// if logErr.Message != "" {
	// 	return logErr
	// }

	// Save the Image Locally
	Err = localimage.SaveImageLocally(ctx, item, imageFile, imageData, imageType, p.GetMediaItemDetails)
	if Err.Message != "" {
		return Err
	}

	// If Save Image Next to Content is enabled and the Path is set, set the poster in Plex via the MediUX URL
	// When the Path is set, the image is saved in a different location than Plex expects it to be.
	// So we need to upload the image to Plex via the MediUX URL.
	// if isCustomLocalPath {
	Err = applyImageToMediaItemViaMediuxURL(ctx, p.Config, item, itemRatingKey, imageFile)
	if Err.Message != "" {
		return Err
	}
	return Err
	// }
	// else {
	// 	// Refresh the Plex item
	// 	RefreshItemMetadata(ctx, p.Config, item, itemRatingKey, false)

	// 	// Get the Plex Poster Key
	// 	failedToGetPosterKey := false
	// 	imageKey, Err := findNewImage(ctx, p.Config, item, itemRatingKey, imageFile.Type, currentImages)
	// 	if Err.Message != "" {
	// 		failedToGetPosterKey = true
	// 		logAction.AppendWarning("message", "Failed to find new image in Plex after local save")
	// 		logAction.AppendWarning("warning", map[string]any{
	// 			"error": Err,
	// 		})
	// 	} else {
	// 		logAction.AppendResult("new_image_key", imageKey)
	// 	}

	// 	// If failedOnGetPosters is true, use the MediUX URL to set the poster
	// 	if failedToGetPosterKey {
	// 		applyImageToMediaItemViaMediuxURL(ctx, p.Config, item, itemRatingKey, imageFile)
	// 		if Err.Message != "" {
	// 			return Err
	// 		}
	// 		return Err
	// 	}

	// 	// Set the Poster using the Plex Image Key
	// 	Err = applyImageToMediaItem(ctx, p.Config, item, itemRatingKey, imageKey, imageFile.Type)
	// 	if Err.Message != "" {
	// 		return Err
	// 	}
	// }

	// return Err
}

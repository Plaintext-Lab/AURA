package ej

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

func (e *EJ) DownloadApplyImageToMediaItem(ctx context.Context, item *models.MediaItem, imageFile models.ImageFile) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf(
		"%s: Downloading and Applying %s Image for %s",
		config.Current.MediaServer.Type, utils.GetFileDownloadName(item.Title, imageFile), utils.MediaItemInfo(*item),
	), logging.LevelDebug)
	defer logAction.Complete()

	// Get the Image from MediUX
	// mediux.GetImage will handle checking the temp folder and caching based on config
	formatDate := imageFile.Modified.Format("20060102150405")
	imageData, imageType, Err := mediux.GetImage(ctx, imageFile.ID, formatDate, mediux.ImageQualityOriginal)
	if Err.Message != "" {
		return Err
	}

	// If SaveImageLocally is enabled, save the image next to the content as well
	if config.Current.Images.SaveImagesLocally.Enabled {
		Err = localimage.SaveImageLocally(ctx, item, imageFile, imageData, imageType, e.GetMediaItemDetails)
		if Err.Message != "" {
			return Err
		}
	}

	// Apply the Image to the Media Item
	Err = applyImageToMediaItem(ctx, item, imageFile, imageData, imageType)
	if Err.Message != "" {
		return Err
	}

	Err = logging.LogErrorInfo{}
	return Err
}

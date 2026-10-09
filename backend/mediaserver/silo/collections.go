package silo

import (
	"aura/logging"
	"aura/models"
	"context"
)

// Collections are not supported for Silo, these are placeholder functions to satisfy the MediaServerInterface

func collectionsNotSupported() logging.LogErrorInfo {
	return logging.LogErrorInfo{
		Message: "Collections are not supported for Silo",
		Help:    "Collections are only available for Plex, Emby and Jellyfin",
	}
}

func (s *Silo) GetMovieCollections(ctx context.Context, library models.LibrarySection) (collections []models.CollectionItem, Err logging.LogErrorInfo) {
	return []models.CollectionItem{}, collectionsNotSupported()
}

func (s *Silo) GetCollectionItemImage(ctx context.Context, collection *models.CollectionItem, imageType string) (imageData []byte, Err logging.LogErrorInfo) {
	return []byte{}, collectionsNotSupported()
}

func (s *Silo) GetMovieCollectionChildrenItems(ctx context.Context, collection *models.CollectionItem) (Err logging.LogErrorInfo) {
	return collectionsNotSupported()
}

func (s *Silo) ApplyCollectionImage(ctx context.Context, collectionItem *models.CollectionItem, imageFile models.ImageFile) (Err logging.LogErrorInfo) {
	return collectionsNotSupported()
}

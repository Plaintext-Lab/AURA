package cache

import (
	"aura/models"
	"sync"
)

// ---   Cache Global Variables (Backend Library Collections Cache) --- ---
var CollectionsStore *MediaServerCollectionsCache

type MediaServerCollectionsCache struct {
	// libraryTitle -> index -> CollectionItem
	collections    map[string]map[string]*models.CollectionItem
	mu             sync.RWMutex
	LastFullUpdate int64
}

// NewCollectionsCache creates a new CollectionsCache instance
func Cache_NewCollectionsCache() *MediaServerCollectionsCache {
	return &MediaServerCollectionsCache{
		collections:    make(map[string]map[string]*models.CollectionItem),
		LastFullUpdate: 0,
	}
}

func init() {
	CollectionsStore = Cache_NewCollectionsCache()
}

// ClearAllCollections removes every collection
func (msc *MediaServerCollectionsCache) ClearAllCollections() {
	msc.mu.Lock()
	defer msc.mu.Unlock()
	msc.collections = make(map[string]map[string]*models.CollectionItem)
	msc.LastFullUpdate = 0
}

func (msc *MediaServerCollectionsCache) GetAllCollections() []models.CollectionItem {
	msc.mu.RLock()
	defer msc.mu.RUnlock()

	out := []models.CollectionItem{}
	for _, lib := range msc.collections {
		for _, coll := range lib {
			out = append(out, *coll)
		}
	}
	return out
}

func (msc *MediaServerCollectionsCache) GetCollectionByRatingKey(ratingKey string) (*models.CollectionItem, bool) {
	msc.mu.RLock()
	defer msc.mu.RUnlock()

	for _, lib := range msc.collections {
		for _, coll := range lib {
			if coll.RatingKey == ratingKey {
				return coll, true
			}
		}
	}
	return nil, false
}

func (msc *MediaServerCollectionsCache) UpsertCollection(collection *models.CollectionItem) {
	msc.mu.Lock()
	defer msc.mu.Unlock()

	lib := msc.collections[collection.LibraryTitle]
	if lib == nil {
		lib = make(map[string]*models.CollectionItem)
		msc.collections[collection.LibraryTitle] = lib
	}
	lib[collection.RatingKey] = collection
}

func (msc *MediaServerCollectionsCache) UpdateMediaItemInCollectionByIndex(collectionIndex string, item *models.MediaItem) {
	msc.mu.Lock()
	defer msc.mu.Unlock()

	if item == nil || item.LibraryTitle == "" {
		return
	}

	lib := msc.collections[item.LibraryTitle]
	if lib == nil {
		return
	}

	collection, ok := lib[collectionIndex]
	if !ok || collection == nil {
		return
	}

	for i := range collection.MediaItems {
		if collection.MediaItems[i].RatingKey == item.RatingKey {
			// Update existing item
			collection.MediaItems[i] = *item
			return
		}
	}

	// If not found, append new item
	collection.MediaItems = append(collection.MediaItems, *item)
}

func (msc *MediaServerCollectionsCache) UpdateMediaItemInCollectionByTitle(collectionTitle string, item *models.MediaItem) {
	msc.mu.Lock()
	defer msc.mu.Unlock()

	if item == nil || item.LibraryTitle == "" {
		return
	}

	lib := msc.collections[item.LibraryTitle]
	if lib == nil {
		return
	}

	for _, collection := range lib {
		if collection == nil || collection.Title != collectionTitle {
			continue
		}

		for i := range collection.MediaItems {
			if collection.MediaItems[i].RatingKey == item.RatingKey {
				collection.MediaItems[i] = *item
				return
			}
		}

		collection.MediaItems = append(collection.MediaItems, *item)
		return
	}

}

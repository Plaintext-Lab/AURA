package cache

import (
	"aura/models"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ---   Cache Global Variables (Backend Library Cache) --- ---
var LibraryStore *MediaServerLibraryCache

type MediaServerLibraryCache struct {
	sections       map[string]*models.LibrarySection // Key: Library Title
	mu             sync.RWMutex
	LastFullUpdate int64
}

// NewLibraryCache creates a new LibraryCache instance
func Cache_NewLibraryCache() *MediaServerLibraryCache {
	return &MediaServerLibraryCache{
		sections:       make(map[string]*models.LibrarySection),
		LastFullUpdate: 0,
	}
}

func init() {
	LibraryStore = Cache_NewLibraryCache()
}

// UpdateSection updates or adds a LibrarySection in the cache.
// If the section already exists, its metadata and media items are updated.
// New media items are appended to the section.
// If the section does not exist, it is added to the cache.
func (c *MediaServerLibraryCache) UpdateSection(section *models.LibrarySection) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if section already exists
	// If it does, we need to update it
	if existing, exists := c.sections[section.Title]; exists {
		// Update metadata
		existing.Type = section.Type
		existing.ID = section.ID
		existing.Paths = section.Paths

		// Create a map of existing items for O(1) lookup.
		// Keyed by TMDB_ID+Edition so multiple editions of the same TMDB item
		// (e.g. Theatrical vs Director's Cut) don't collapse into one entry.
		existingItems := make(map[string]*models.MediaItem)
		for i := range existing.MediaItems {
			existingItems[mediaItemCacheKey(&existing.MediaItems[i])] = &existing.MediaItems[i]
		}

		// Update existing items and collect new ones
		var newItems []models.MediaItem
		for _, newItem := range section.MediaItems {
			if existingItem, found := existingItems[mediaItemCacheKey(&newItem)]; found {
				// Update existing item
				*existingItem = newItem
			} else {
				// Collect new item for appending
				newItems = append(newItems, newItem)
			}
		}

		// Append new items
		existing.MediaItems = append(existing.MediaItems, newItems...)
		existing.TotalSize = len(existing.MediaItems)
	} else {
		// If section does not exist, add it to the cache
		c.sections[section.Title] = section
	}
}

// UpdateMediaItem updates a specific media item in a section
func (c *MediaServerLibraryCache) UpdateMediaItem(sectionTitle string, item *models.MediaItem) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if section exists
	if section, exists := c.sections[sectionTitle]; exists {
		// Create a map of existing items for O(1) lookup
		existingItems := make(map[string]*models.MediaItem)
		for i := range section.MediaItems {
			existingItems[mediaItemCacheKey(&section.MediaItems[i])] = &section.MediaItems[i]
		}
		if existingItem, found := existingItems[mediaItemCacheKey(item)]; found {
			// Update existing item
			*existingItem = *item
		} else {
			// Append new item
			section.MediaItems = append(section.MediaItems, *item)
			section.TotalSize = len(section.MediaItems)
		}
	}
}

func (c *MediaServerLibraryCache) UpdateMediaItemDBSavedSets(sectionTitle string, item *models.MediaItem, dbSavedSets []models.DBSavedSet) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if section exists
	if section, exists := c.sections[sectionTitle]; exists {
		// Create a map of existing items for O(1) lookup
		existingItems := make(map[string]*models.MediaItem)
		for i := range section.MediaItems {
			existingItems[mediaItemCacheKey(&section.MediaItems[i])] = &section.MediaItems[i]
		}
		if existingItem, found := existingItems[mediaItemCacheKey(item)]; found {
			// Update existing item
			existingItem.DBSavedSets = dbSavedSets
		}
	}
}

// GetSectionByTitle retrieves a section by Title
func (c *MediaServerLibraryCache) GetSectionByTitle(title string) (*models.LibrarySection, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	section, exists := c.sections[title]
	return section, exists
}

func (c *MediaServerLibraryCache) GetRatingKeyByTMDBID(libraryTitle, tmdbID string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	section, exists := c.sections[libraryTitle]
	if !exists {
		return "", false
	}

	for _, item := range section.MediaItems {
		if item.TMDB_ID == tmdbID {
			return item.RatingKey, true
		}
	}

	return "", false
}

// GetAllSectionsSortedByTitle returns all sections sorted by Title
func (c *MediaServerLibraryCache) GetAllSectionsSortedByTitle() []*models.LibrarySection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sections := make([]*models.LibrarySection, 0, len(c.sections))
	for _, section := range c.sections {
		sections = append(sections, section)
	}

	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Title < sections[j].Title
	})

	return sections
}

// RemoveSectionByTitle removes a section from the cache by Title
func (c *MediaServerLibraryCache) RemoveSectionByTitle(title string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.sections, title)
}

// ClearAllSections removes all sections from the cache
func (c *MediaServerLibraryCache) ClearAllSections() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sections = make(map[string]*models.LibrarySection)
}

// mediaItemCacheKey uniquely identifies a media item within a section cache.
// Edition is included so multiple editions of the same TMDB item (e.g.
// Theatrical vs Director's Cut) are tracked as distinct cache entries.
func mediaItemCacheKey(item *models.MediaItem) string {
	return item.TMDB_ID + "|" + item.Edition
}

// GetMediaItemFromSectionByTMDBID retrieves a media item by TMDB ID from a specific section.
// If multiple editions exist for the TMDB ID, an arbitrary one is returned -
// callers that need a specific edition should use GetMediaItemFromSectionByTMDBIDAndEdition.
func (c *MediaServerLibraryCache) GetMediaItemFromSectionByTMDBID(sectionTitle, tmdbID string) (*models.MediaItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	section, exists := c.sections[sectionTitle]
	if !exists {
		return &models.MediaItem{}, false
	}

	var newestItem *models.MediaItem
	for _, item := range section.MediaItems {
		if item.TMDB_ID == tmdbID {
			newestItem = &item
			break
		}
	}

	if newestItem != nil {
		return newestItem, true
	}
	return &models.MediaItem{}, false
}

// GetMediaItemFromSectionByTMDBIDAndEdition retrieves a specific edition of a
// media item by TMDB ID from a specific section.
func (c *MediaServerLibraryCache) GetMediaItemFromSectionByTMDBIDAndEdition(sectionTitle, tmdbID, edition string) (*models.MediaItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	section, exists := c.sections[sectionTitle]
	if !exists {
		return &models.MediaItem{}, false
	}

	for _, item := range section.MediaItems {
		if item.TMDB_ID == tmdbID && item.Edition == edition {
			return &item, true
		}
	}

	return &models.MediaItem{}, false
}

func (c *MediaServerLibraryCache) GetMediaItemByRatingKey(ratingKey string) (*models.MediaItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, section := range c.sections {
		for _, item := range section.MediaItems {
			if item.RatingKey == ratingKey {
				return &item, true
			}
		}
	}

	return &models.MediaItem{}, false
}

// GetMediaItemsByRatingKey retrieves the media item with this rating key from every section it is in.
// Some media servers (e.g. Silo) use the same rating key for an item in every library it is in.
func (c *MediaServerLibraryCache) GetMediaItemsByRatingKey(ratingKey string) []models.MediaItem {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := []models.MediaItem{}
	for _, section := range c.sections {
		for _, item := range section.MediaItems {
			if item.RatingKey == ratingKey {
				items = append(items, item)
			}
		}
	}
	return items
}

// GetMediaItemFromSectionByRatingKey retrieves a media item by rating key from a specific section.
// Some media servers (e.g. Silo) use the same rating key for an item in every library it is in.
func (c *MediaServerLibraryCache) GetMediaItemFromSectionByRatingKey(sectionTitle, ratingKey string) (*models.MediaItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	section, exists := c.sections[sectionTitle]
	if !exists {
		return &models.MediaItem{}, false
	}

	for _, item := range section.MediaItems {
		if item.RatingKey == ratingKey {
			return &item, true
		}
	}

	return &models.MediaItem{}, false
}

// GetMediaItemFromSectionByTitleAndYear retrieves the TMDB ID from a media item by its title and year
func (c *MediaServerLibraryCache) GetMediaItemFromSectionByTitleAndYear(sectionTitle, itemTitle string, year int) (*models.MediaItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	section, exists := c.sections[sectionTitle]
	if !exists {
		return &models.MediaItem{}, false
	}

	cleanedSearchTitle := cleanStringForComparison(stripYearFromTitle(itemTitle))
	for _, item := range section.MediaItems {
		cleanedTitle := cleanStringForComparison(stripYearFromTitle(item.Title))
		if cleanedTitle == cleanedSearchTitle && item.Year == year {
			return &item, true
		}
	}

	return &models.MediaItem{}, false
}

func (c *MediaServerLibraryCache) GetSectionsCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.sections)
}

func (c *MediaServerLibraryCache) GetItemsCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	totalItems := 0
	for _, section := range c.sections {
		totalItems += len(section.MediaItems)
	}
	return totalItems
}

func (c *MediaServerLibraryCache) GetAllMediaItems() []models.MediaItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var allItems []models.MediaItem
	for _, section := range c.sections {
		allItems = append(allItems, section.MediaItems...)
	}
	return allItems
}

// IsEmpty checks if the cache is empty
func (c *MediaServerLibraryCache) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.sections) == 0
}

func cleanStringForComparison(input string) string {
	var b strings.Builder
	input = strings.ToLower(input)
	for _, r := range input {
		switch r {
		case '-', '_', '.', ',', ':', ';', '!', '?', '\'', '(', ')', '[', ']', '{', '}':
			// skip these characters
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func stripYearFromTitle(title string) string {
	parts := strings.Fields(title)
	if len(parts) > 1 {
		last := parts[len(parts)-1]
		if _, err := strconv.Atoi(last); err == nil {
			return strings.Join(parts[:len(parts)-1], " ")
		}
	}
	return title
}

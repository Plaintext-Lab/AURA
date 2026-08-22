import { removeSavedSet, upsertSavedSets } from "@/helper/media-item-update-saved-sets";
import { create } from "zustand";
import { persist } from "zustand/middleware";

import { log } from "@/lib/logger";
import { useMediaStore } from "@/lib/stores/global-store-media-store";
import { GlobalStore } from "@/lib/stores/stores";

import type { LibrarySection, MediaItem, SelectedTypes } from "@/types/media-and-posters/media-item-and-library";

// Max Cache Duration: 1 Hour
export const MAX_CACHE_DURATION = 60 * 60 * 1000;

// Dedupes by TMDB ID + Edition, so multiple editions of the same TMDB item
// (e.g. Theatrical vs Director's Cut) are kept as distinct entries.
const dedupeMediaItemsByTmdbIDAndEdition = (
  mediaItems: LibrarySection["media_items"]
): LibrarySection["media_items"] => {
  const seenKeys = new Set<string>();

  return mediaItems.filter((item) => {
    if (item.tmdb_id === null || item.tmdb_id === undefined || item.tmdb_id === "") {
      return true;
    }

    const key = `${String(item.tmdb_id)}|${item.library_id || ""}|${item.edition || ""}`;
    if (seenKeys.has(key)) {
      return false;
    }

    seenKeys.add(key);
    return true;
  });
};

const sanitizeSection = (section: LibrarySection): LibrarySection => {
  const deduplicatedMediaItems = dedupeMediaItemsByTmdbIDAndEdition(section.media_items || []);

  return {
    ...section,
    media_items: deduplicatedMediaItems,
    total_size: deduplicatedMediaItems.length,
  };
};

const sanitizeSectionsRecord = (sections: Record<string, LibrarySection>): Record<string, LibrarySection> => {
  return Object.entries(sections).reduce<Record<string, LibrarySection>>((acc, [sectionTitle, section]) => {
    acc[sectionTitle] = sanitizeSection(section);
    return acc;
  }, {});
};

interface LibrarySectionsStore {
  sections: Record<string, LibrarySection>;
  setSections: (sections: Record<string, LibrarySection>, timestamp?: number) => void;
  setSection: (sectionTitle: string, record: LibrarySection) => void;

  timestamp?: number;
  setTimestamp: (timestamp: number) => void;

  removeSection: (sectionTitle: string) => void;

  /** Update a media item in a section (e.g. after fetching latest data). */
  updateMediaItem: (mediaItem: MediaItem) => void;

  /** Add/Upsert one saved set onto db_saved_sets (e.g. after adding a set to DB). */
  upsertMediaItemSavedSet: (args: {
    tmdbID: string;
    libraryTitle: string;
    edition?: string;
    setID: string;
    setUser: string;
    selectedTypes: SelectedTypes;
    toDelete?: boolean;
  }) => void;

  /** Clear db_saved_sets (e.g. after deleting entire item from DB). */
  clearMediaItemSavedSets: (args: { tmdbID: string; libraryTitle: string; edition?: string }) => void;

  updateIgnoreStatus: (
    tmdbID: string,
    libraryTitle: string,
    ignored: boolean,
    ignoreMode: string,
    edition?: string
  ) => void;

  getSectionSummaries: () => { title: string; type: string }[];

  clear: () => void;

  hasHydrated: boolean;
  hydrate: () => void;
}

export const useLibrarySectionsStore = create<LibrarySectionsStore>()(
  persist(
    (set, get) => ({
      sections: {},

      setSections: (sections, timestamp) =>
        set({
          sections: sanitizeSectionsRecord(sections),
          timestamp: timestamp ?? Date.now(),
        }),

      setSection: (sectionTitle, record) =>
        set((state) => ({
          sections: {
            ...state.sections,
            [sectionTitle]: sanitizeSection(record),
          },
        })),

      timestamp: undefined,
      setTimestamp: (timestamp) => set({ timestamp }),

      removeSection: (sectionTitle) =>
        set((state) => {
          const next = { ...state.sections };
          delete next[sectionTitle];
          return { sections: next };
        }),

      updateMediaItem: (mediaItem: MediaItem) => {
        const { sections } = get();
        const section = sections[mediaItem.library_title];
        if (!section || !Array.isArray(section.media_items)) {
          log(
            "ERROR",
            "librarySections",
            "updateMediaItem",
            `Section "${mediaItem.library_title}" not found or has no MediaItems`
          );
          return;
        }

        const mediaItems = [...section.media_items];
        const idx = mediaItems.findIndex(
          (m) => m.tmdb_id === mediaItem.tmdb_id && (m.edition || "") === (mediaItem.edition || "")
        );
        if (idx === -1) {
          log(
            "ERROR",
            "librarySections",
            "updateMediaItem",
            `Media item '${mediaItem.title}' (TMDB ID: ${mediaItem.tmdb_id}) not found in section "${mediaItem.library_title}"`
          );
          return;
        }

        set((state) => ({
          sections: {
            ...state.sections,
            [mediaItem.library_title]: {
              ...section,
              media_items: mediaItems,
            },
          },
        }));

        // Only update the active media item if it matches
        const mediaState = useMediaStore.getState() as {
          mediaItem?: MediaItem;
          setMediaItem: (m: MediaItem) => void;
        };
        if (
          mediaState.mediaItem?.tmdb_id === mediaItem.tmdb_id &&
          mediaState.mediaItem?.library_title === mediaItem.library_title &&
          (mediaState.mediaItem?.edition || "") === (mediaItem.edition || "")
        ) {
          mediaState.setMediaItem(mediaItems[idx]);
        }

        log(
          "INFO",
          "librarySections",
          "updateMediaItem",
          `Updated media item ${mediaItem.title} (TMDB ID: ${mediaItem.tmdb_id}) in section "${mediaItem.library_title}"`
        );
      },

      upsertMediaItemSavedSet: ({
        tmdbID,
        libraryTitle,
        edition = "",
        setID,
        setUser,
        selectedTypes,
        toDelete = false,
      }) => {
        const { sections } = get();
        const section = sections[libraryTitle];
        if (!section || !Array.isArray(section.media_items)) {
          log(
            "ERROR",
            "librarySections",
            "upsertMediaItemSavedSet",
            `Section "${libraryTitle}" not found or has no MediaItems`
          );
          return;
        }

        const mediaItems = [...section.media_items];
        const idx = mediaItems.findIndex((m) => m.tmdb_id === tmdbID && (m.edition || "") === edition);
        if (idx === -1) {
          log(
            "ERROR",
            "librarySections",
            "upsertMediaItemSavedSet",
            `Media item (TMDB ID: ${tmdbID}) not found in section "${libraryTitle}"`
          );
          return;
        }

        if (toDelete) {
          mediaItems[idx] = removeSavedSet(mediaItems[idx], setID);
        } else {
          mediaItems[idx] = upsertSavedSets(mediaItems[idx], setID, setUser, selectedTypes);
        }

        set((state) => ({
          sections: {
            ...state.sections,
            [libraryTitle]: { ...section, media_items: mediaItems },
          },
        }));

        const mediaState = useMediaStore.getState() as {
          mediaItem?: MediaItem;
          setMediaItem: (m: MediaItem) => void;
        };
        if (
          mediaState.mediaItem?.tmdb_id === tmdbID &&
          mediaState.mediaItem?.library_title === libraryTitle &&
          (mediaState.mediaItem?.edition || "") === edition
        ) {
          mediaState.setMediaItem(mediaItems[idx]);
        }
      },

      clearMediaItemSavedSets: ({ tmdbID, libraryTitle, edition = "" }) => {
        const { sections } = get();
        const section = sections[libraryTitle];
        if (!section || !Array.isArray(section.media_items)) {
          log(
            "ERROR",
            "librarySections",
            "clearMediaItemSavedSets",
            `Section "${libraryTitle}" not found or has no MediaItems`
          );
          return;
        }

        const mediaItems = [...section.media_items];
        const idx = mediaItems.findIndex((m) => m.tmdb_id === tmdbID && (m.edition || "") === edition);
        if (idx === -1) {
          log(
            "ERROR",
            "librarySections",
            "clearMediaItemSavedSets",
            `Media item (TMDB ID: ${tmdbID}) not found in section "${libraryTitle}"`
          );
          return;
        }

        mediaItems[idx] = { ...mediaItems[idx], db_saved_sets: [] };

        set((state) => ({
          sections: {
            ...state.sections,
            [libraryTitle]: { ...section, media_items: mediaItems },
          },
        }));

        const mediaState = useMediaStore.getState() as {
          mediaItem?: MediaItem;
          setMediaItem: (m: MediaItem) => void;
        };
        if (
          mediaState.mediaItem?.tmdb_id === tmdbID &&
          mediaState.mediaItem?.library_title === libraryTitle &&
          (mediaState.mediaItem?.edition || "") === edition
        ) {
          mediaState.setMediaItem(mediaItems[idx]);
        }
      },

      updateIgnoreStatus: (tmdbID, libraryTitle, ignored, ignoreMode, edition = "") => {
        const { sections } = get();
        const section = sections[libraryTitle];
        if (!section || !Array.isArray(section.media_items)) {
          log(
            "ERROR",
            "librarySections",
            "updateIgnoreStatus",
            `Section "${libraryTitle}" not found or has no MediaItems`
          );
          return;
        }

        const mediaItems = [...section.media_items];
        const idx = mediaItems.findIndex((m) => m.tmdb_id === tmdbID && (m.edition || "") === edition);
        if (idx === -1) {
          log(
            "ERROR",
            "librarySections",
            "updateIgnoreStatus",
            `Media item with TMDB ID: ${tmdbID} not found in section "${libraryTitle}"`
          );
          return;
        }

        mediaItems[idx] = { ...mediaItems[idx], ignored_in_db: ignored, ignored_mode: ignoreMode };

        set((state) => ({
          sections: {
            ...state.sections,
            [libraryTitle]: {
              ...section,
              media_items: mediaItems,
            },
          },
        }));

        log(
          "INFO",
          "librarySections",
          "updateIgnoreStatus",
          `Updated ignore status for media item (TMDB ID: ${tmdbID}) in section "${libraryTitle}" to ignored: ${ignored}, mode: ${ignoreMode}`
        );
      },

      getSectionSummaries: () => {
        const { sections } = get();
        return Object.values(sections).map((rec) => ({
          title: rec.title,
          type: rec.type,
        }));
      },

      clear: () => {
        set({ sections: {}, timestamp: undefined });
      },

      hasHydrated: false,
      hydrate: () => {
        set({ hasHydrated: true });
      },
    }),
    {
      name: "LibrarySections",
      storage: GlobalStore,
      partialize: (state) => ({
        sections: state.sections,
        timestamp: state.timestamp,
      }),
      onRehydrateStorage: () => (state) => {
        if (state?.sections) {
          state.setSections(state.sections, state.timestamp);
        }
        state?.hydrate();
      },
    }
  )
);

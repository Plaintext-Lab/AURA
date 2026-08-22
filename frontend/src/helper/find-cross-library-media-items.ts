import type { LibrarySection, MediaItem } from "@/types/media-and-posters/media-item-and-library";

export const findMediaItemInOtherLibraries = (
  mediaItem: MediaItem,
  librarySectionsMap: Record<string, LibrarySection>
): MediaItem[] => {
  const tmdbID = mediaItem.guids?.find((guid) => guid.provider === "tmdb")?.id;
  if (!tmdbID) return [];

  const matches: MediaItem[] = [];
  for (const section of Object.values(librarySectionsMap)) {
    if (section.type !== mediaItem.type || section.title === mediaItem.library_title) continue;
    if (!section.media_items || section.media_items.length === 0) continue;

    const match = section.media_items.find((item) =>
      item.guids?.some((guid) => guid.provider === "tmdb" && guid.id === tmdbID)
    );
    if (match) matches.push(match);
  }

  return matches;
};
